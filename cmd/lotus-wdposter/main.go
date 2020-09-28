package main

import (
	"context"
	"fmt"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/lotus/miner"
	"github.com/filecoin-project/lotus/node/config"
	"github.com/filecoin-project/lotus/node/modules"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/storage"
	"os/signal"
	"syscall"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc/auth"
	paramfetch "github.com/filecoin-project/go-paramfetch"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/types"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/lotus/extern/sector-storage/stores"
	"github.com/filecoin-project/lotus/lib/lotuslog"
	"github.com/filecoin-project/lotus/lib/tracing"
	"github.com/filecoin-project/lotus/node/repo"
	"github.com/gorilla/mux"
	logging "github.com/ipfs/go-log/v2"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/trace"
	"golang.org/x/xerrors"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var log = logging.Logger("main")

const FlagWdposterRepo = "wdposter-repo"

func main() {
	build.RunningNodeType = build.NodeWorker //现在我们把这个wdpost服务当成一个worker来对待
	lotuslog.SetupLogLevels()
	local := []*cli.Command{
		runCmd,
		provingCmd,
	}
	jaeger := tracing.SetupJaegerTracing("lotus-wdposter")
	defer func() {
		if jaeger != nil {
			jaeger.Flush()
		}
	}()

	for _, cmd := range local {
		cmd := cmd
		originBefore := cmd.Before
		cmd.Before = func(cctx *cli.Context) error {
			trace.UnregisterExporter(jaeger)
			jaeger = tracing.SetupJaegerTracing("lotus/" + cmd.Name)
			if !cctx.IsSet("actor") {
				return xerrors.Errorf("could not allow actor empty")
			}
			if originBefore != nil {
				return originBefore(cctx)
			}
			return nil
		}
	}

	app := &cli.App{
		Name:                 "lotus-wdposter",
		Usage:                "Filecoin decentralized window poster",
		Version:              build.UserVersion(),
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    FlagWdposterRepo,
				EnvVars: []string{"LOTUS_WDPOSTER_PATH", "WDPOSTER_PATH"},
				Value:   "~/.lotuswdposter", // TODO: Consider XDG_DATA_HOME
				Usage:   fmt.Sprintf("Specify wdposter repo path. flag %s ", FlagWdposterRepo),
			},
			&cli.BoolFlag{
				Name:  "enable-gpu-proving",
				Usage: "enable use of GPU for wdpost operations",
				Value: true,
			},
			&cli.StringFlag{
				Name:  "actor",
				Usage: "must specify window poster server miner ID",
			},
		},
		Commands: local,
	}
	app.Setup()
	app.Metadata["repoType"] = repo.Worker

	if err := app.Run(os.Args); err != nil {
		log.Warnf("%+v", err)
		return
	}
}

var runCmd = &cli.Command{
	Name:  "run",
	Usage: "Start lotus window poster ",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "listen",
			Usage: "host address and port the worker api will listen on",
			Value: "0.0.0.0:4567",
		},
		&cli.StringFlag{
			Name:  "storage-dir",
			Usage: "use storage for sector storage",
		},
	},
	Before: func(cctx *cli.Context) error {
		if cctx.IsSet("address") {
			log.Warnf("The '--address' flag is deprecated, it has been replaced by '--listen'")
			if err := cctx.Set("listen", cctx.String("address")); err != nil {
				return err
			}
		}
		if !cctx.Bool("enable-gpu-proving") {
			if err := os.Setenv("BELLMAN_NO_GPU", "true"); err != nil {
				return xerrors.Errorf("could not set no-gpu env: %+v", err)
			}
		}
		return nil
	},
	Action: func(cctx *cli.Context) error {
		log.Info("Starting lotus window poster")
		//todo 断线重连机制
		nodeApi, ncloser, err := lcli.GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer ncloser()

		ctx := lcli.ReqContext(cctx)
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Check params
		maddr, err := address.NewFromString(cctx.String("actor"))
		if err != nil {
			return err
		}
		minerInfo, err := nodeApi.StateMinerInfo(ctx, maddr, types.EmptyTSK)
		if err != nil {
			return err
		}
		if err := paramfetch.GetParams(ctx, build.ParametersJSON(), uint64(minerInfo.SectorSize)); err != nil {
			return xerrors.Errorf("get params: %w", err)
		}

		// Open repo
		repoPath := cctx.String(FlagWdposterRepo)
		r, err := repo.NewFS(repoPath)
		if err != nil {
			return err
		}
		ok, err := r.Exists()
		if err != nil {
			return err
		}
		if !ok {
			//如果是第一次初始化,那么 storage-dir 不能为空
			if cctx.String("storage-dir") == "" {
				return xerrors.Errorf("storage dir don't allow empty")
			}
			if err := r.Init(repo.Worker); err != nil {
				return err
			}

			lr, err := r.Lock(repo.Worker)
			if err != nil {
				return err
			}

			var localPaths []stores.LocalPath

			localPaths = append(localPaths, stores.LocalPath{
				Path: cctx.String("storage-dir"),
			})

			if err := lr.SetStorage(func(sc *stores.StorageConfig) {
				sc.StoragePaths = append(sc.StoragePaths, localPaths...)
			}); err != nil {
				return xerrors.Errorf("set storage config: %w", err)
			}
			{
				// init datastore for r.Exists
				_, err := lr.Datastore("/metadata")
				if err != nil {
					return err
				}
			}
			if err := lr.Close(); err != nil {
				return xerrors.Errorf("close repo: %w", err)
			}
		}

		lr, err := r.Lock(repo.Worker)
		if err != nil {
			return err
		}

		log.Info("Opening local storage; connecting to master")
		//todo 完善api功能
		const unspecifiedAddress = "0.0.0.0"
		addr := cctx.String("listen")
		addressSlice := strings.Split(addr, ":")
		if ip := net.ParseIP(addressSlice[0]); ip != nil {
			if ip.String() == unspecifiedAddress {
				timeout, err := time.ParseDuration("5s")
				if err != nil {
					return err
				}
				rip, err := extractRoutableIP(timeout)
				if err != nil {
					return err
				}
				addr = rip + ":" + addressSlice[1]
			}
		}

		// Create / expose the worker
		mux := mux.NewRouter()
		log.Info("Setting up control endpoint at " + addr)
		//readerHandler, readerServerOpt := rpcenc.ReaderParamDecoder()
		//rpcServer := jsonrpc.NewServer(readerServerOpt)
		//rpcServer.Register("Filecoin", apistruct.PermissionedWorkerAPI(workerApi))
		//mux.Handle("/rpc/v0", rpcServer)
		//mux.Handle("/rpc/streams/v0/push/{uuid}", readerHandler)
		//mux.PathPrefix("/remote").HandlerFunc((&stores.FetchHandler{Local: localStore}).ServeHTTP)
		mux.PathPrefix("/").Handler(http.DefaultServeMux)

		ah := &auth.Handler{
			Verify: nodeApi.AuthVerify,
			Next:   mux.ServeHTTP,
		}

		srv := &http.Server{
			Handler: ah,
			BaseContext: func(listener net.Listener) context.Context {
				return ctx
			},
		}
		nl, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		//set token
		{
			a, err := net.ResolveTCPAddr("tcp", addr)
			if err != nil {
				return xerrors.Errorf("parsing address: %w", err)
			}

			ma, err := manet.FromNetAddr(a)
			if err != nil {
				return xerrors.Errorf("creating api multiaddress: %w", err)
			}

			if err := lr.SetAPIEndpoint(ma); err != nil {
				return xerrors.Errorf("setting api endpoint: %w", err)
			}

			ainfo, err := lcli.GetAPIInfo(cctx, repo.FullNode)
			if err != nil {
				return xerrors.Errorf("could not get miner API info: %w", err)
			}
			// TODO: ideally this would be a token with some permissions dropped
			if err := lr.SetAPIToken(ainfo.Token); err != nil {
				return xerrors.Errorf("setting api token: %w", err)
			}
		}

		//启动时空证明协程
		//todo 能够切分扇区进行时空证明
		index := stores.NewIndex()
		localStore, err := stores.NewLocal(ctx, lr, index, []string{"http://" + addr + "/remote"})
		if err != nil {
			return err
		}
		cfg, err := lr.GetStorage()
		if err != nil || len(cfg.StoragePaths) == 0 {
			return xerrors.Errorf("getting local storage config: %w", err)
		}

		provider, err := ffiwrapper.New(NewLocalProvider(localStore, minerInfo.SealProofType), &ffiwrapper.Config{
			SealProofType: minerInfo.SealProofType,
		})
		if err != nil {
			return err
		}
		worker, err := nodeApi.StateAccountKey(ctx, minerInfo.Worker, types.EmptyTSK)
		if err != nil {
			return err
		}
		sched, err := storage.NewWindowedPoStScheduler(nodeApi, config.MinerFeeConfig{}, provider, NewLocalFaultTracker(localStore, index), maddr, worker)
		if err != nil {
			return err
		}
		go sched.Run(ctx)

		//启动挖矿协程
		mid, err := address.IDFromAddress(maddr)
		if err != nil {
			return xerrors.Errorf("getting id address: %w", err)
		}
		winProver, err := storage.NewWinningPoStProver(nodeApi, provider, ffiwrapper.ProofVerifier, dtypes.MinerID(mid))
		if err != nil {
			return xerrors.Errorf("getting winning post prover: %w", err)
		}
		ds, err := modules.Datastore(lr)
		if err != nil {
			return xerrors.Errorf("get metads err: %w", err)
		}
		winMiner := miner.NewMiner(nodeApi, winProver, maddr, modules.NewSlashFilter(ds))
		if err := winMiner.Start(ctx); err != nil {
			return xerrors.Errorf("winning miner err: %w", err)
		}

		//listen system signal
		sigChan := make(chan os.Signal, 2)
		go func() {
			select {
			case <-sigChan:
			}
			log.Warn("Shutting down...")
			winMiner.Stop(ctx)
			if err := srv.Shutdown(context.TODO()); err != nil {
				log.Errorf("shutting down RPC server failed: %s", err)
			}
			log.Warn("Graceful shutdown successful")
		}()
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

		return srv.Serve(nl)
	},
}

func extractRoutableIP(timeout time.Duration) (string, error) {
	minerMultiAddrKey := "FULLNODE_API_INFO"
	env, ok := os.LookupEnv(minerMultiAddrKey)
	if !ok {
		return "", xerrors.New("FULLNODE_API_INFO environment variable required to extract IP")
	}
	minerAddr := strings.Split(env, "/")
	conn, err := net.DialTimeout("tcp", minerAddr[2]+":"+minerAddr[4], timeout)
	if err != nil {
		return "", err
	}
	defer conn.Close() //nolint:errcheck

	localAddr := conn.LocalAddr().(*net.TCPAddr)

	return strings.Split(localAddr.IP.String(), ":")[0], nil
}
