package main

import (
	"context"
	"fmt"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api/apistruct"
	"github.com/filecoin-project/lotus/extern/sector-storage/stores"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/xerrors"
	"os"
	"path/filepath"

	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/lotus/journal"
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
	"github.com/filecoin-project/lotus/node/repo"
	"github.com/gorilla/mux"

	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/urfave/cli/v2"
	"net"
	"net/http"
	"strings"
	"time"
)

var localCmd = &cli.Command{
	Name:  "local",
	Usage: "Start lotus window poster local",
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
		repoPath := cctx.String(FlagPosterRepo)
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

		index := stores.NewIndex()
		localStore, err := stores.NewLocal(ctx, lr, index, []string{"http://" + addr + "/remote"})
		if err != nil {
			return err
		}
		posterApi := &LocalPoster{
			localStore: localStore,
			ls:         lr,
			spt:        minerInfo.SealProofType,
		}
		mux := mux.NewRouter()
		log.Info("Setting up control endpoint at " + addr)
		rpcServer := jsonrpc.NewServer()
		rpcServer.Register("Filecoin", apistruct.PermissionedPosterAPI(posterApi))
		mux.Handle("/rpc/v0", rpcServer)
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
		cfg, err := lr.GetStorage()
		if err != nil || len(cfg.StoragePaths) == 0 {
			return xerrors.Errorf("getting local storage config: %w", err)
		}

		provider, err := ffiwrapper.New(posterApi, &ffiwrapper.Config{
			SealProofType: minerInfo.SealProofType,
		})
		if err != nil {
			return err
		}
		worker, err := nodeApi.StateAccountKey(ctx, minerInfo.Worker, types.EmptyTSK)
		if err != nil {
			return err
		}
		j, err := journal.OpenFSJournal(lr, journal.EnvDisabledEvents())
		if err != nil {
			return fmt.Errorf("failed to open filesystem journal: %w", err)
		}

		sched, err := storage.NewWindowedPoStScheduler(nodeApi, config.MinerFeeConfig{}, provider, NewLocalFaultTracker(localStore, index), j, maddr, worker)
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
		winMiner := miner.NewMiner(nodeApi, winProver, maddr, modules.NewSlashFilter(ds), j)
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

type LocalFaultTracker struct {
	index      stores.SectorIndex
	localStore *stores.Local
}

func NewLocalFaultTracker(local *stores.Local, sindex stores.SectorIndex) *LocalFaultTracker {
	return &LocalFaultTracker{
		localStore: local,
		index:      sindex,
	}
}

// CheckProvable returns unprovable sectors
func (l *LocalFaultTracker) CheckProvable(ctx context.Context, spt abi.RegisteredSealProof, sectors []abi.SectorID) ([]abi.SectorID, error) {
	var bad []abi.SectorID

	ssize, err := spt.SectorSize()
	if err != nil {
		return nil, err
	}

	// TODO: More better checks
	for _, sector := range sectors {
		err := func() error {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			locked, err := l.index.StorageTryLock(ctx, sector, stores.FTSealed|stores.FTCache, stores.FTNone)
			if err != nil {
				return xerrors.Errorf("acquiring sector lock: %w", err)
			}

			if !locked {
				log.Warnw("CheckProvable Sector FAULT: can't acquire read lock", "sector", sector, "sealed")
				bad = append(bad, sector)
				return nil
			}

			lp, lpDone, err := l.AcquireSector(ctx, spt, sector)
			//lp, _, err := l.localStore.AcquireSector(ctx, sector, spt, stores.FTSealed|stores.FTCache, stores.FTNone, stores.PathStorage, stores.AcquireMove)
			if err != nil {
				log.Warnw("CheckProvable Sector FAULT: acquire sector in checkProvable", "sector", sector, "error", err)
				bad = append(bad, sector)
				return nil
			}

			if lp.Sealed == "" || lp.Cache == "" {
				log.Warnw("CheckProvable Sector FAULT: cache an/or sealed paths not found", "sector", sector, "sealed", lp.Sealed, "cache", lp.Cache)
				bad = append(bad, sector)
				return nil
			}

			toCheck := map[string]int64{
				lp.Sealed:                        1,
				filepath.Join(lp.Cache, "t_aux"): 0,
				filepath.Join(lp.Cache, "p_aux"): 0,
			}

			addCachePathsForSectorSize(toCheck, lp.Cache, ssize)

			for p, sz := range toCheck {
				st, err := os.Stat(p)
				if err != nil {
					log.Warnw("CheckProvable Sector FAULT: sector file stat error", "sector", sector, "sealed", lp.Sealed, "cache", lp.Cache, "file", p, "err", err)
					bad = append(bad, sector)
					return nil
				}

				if sz != 0 {
					if st.Size() != int64(ssize)*sz {
						log.Warnw("CheckProvable Sector FAULT: sector file is wrong size", "sector", sector, "sealed", lp.Sealed, "cache", lp.Cache, "file", p, "size", st.Size(), "expectSize", int64(ssize)*sz)
						bad = append(bad, sector)
						return nil
					}
				}
			}
			if lpDone != nil {
				lpDone()
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}

	return bad, nil
}

func (l *LocalFaultTracker) AcquireSector(ctx context.Context, spt abi.RegisteredSealProof, sid abi.SectorID) (stores.SectorPaths, func(), error) {
	var (
		out     stores.SectorPaths
		err     error
		storeID stores.ID
	)
	out, _, err = l.localStore.AcquireSector(ctx, sid, spt, stores.FTSealed|stores.FTCache, stores.FTNone, stores.PathStorage, stores.AcquireMove)
	if err == nil && (out.Sealed != "" && out.Cache != "") {
		return out, nil, nil
	}

	paths, err := l.localStore.Local(ctx)
	if err != nil {
		return stores.SectorPaths{}, nil, err
	}

	found := false
	var weight uint64 = 0
	for _, path := range paths {
		if !path.CanStore {
			continue
		}
		if path.LocalPath == "" {
			continue
		}
		if weight < path.Weight {
			found = true
			weight = path.Weight
			storeID = path.ID

			stores.SetPathByType(&out, stores.FTSealed, filepath.Join(path.LocalPath, stores.FTSealed.String(), stores.SectorName(sid)))
			stores.SetPathByType(&out, stores.FTCache, filepath.Join(path.LocalPath, stores.FTCache.String(), stores.SectorName(sid)))
		}
	}
	if !found {
		return stores.SectorPaths{}, nil, xerrors.New(fmt.Sprintf("don't find any sector %d", sid))
	}

	return out, func() {
		if err := l.index.StorageDeclareSector(ctx, storeID, sid, stores.FTCache, true); err != nil {
			log.Errorf("declare sector cache error: %+v", err)
		}
		if err := l.index.StorageDeclareSector(ctx, storeID, sid, stores.FTSealed, true); err != nil {
			log.Errorf("declare sector sealed error: %+v", err)
		}
	}, nil
}

func addCachePathsForSectorSize(chk map[string]int64, cacheDir string, ssize abi.SectorSize) {
	switch ssize {
	case 2 << 10:
		fallthrough
	case 8 << 20:
		fallthrough
	case 512 << 20:
		chk[filepath.Join(cacheDir, "sc-02-data-tree-r-last.dat")] = 0
	case 32 << 30:
		for i := 0; i < 8; i++ {
			chk[filepath.Join(cacheDir, fmt.Sprintf("sc-02-data-tree-r-last-%d.dat", i))] = 0
		}
	case 64 << 30:
		for i := 0; i < 16; i++ {
			chk[filepath.Join(cacheDir, fmt.Sprintf("sc-02-data-tree-r-last-%d.dat", i))] = 0
		}
	default:
		log.Warnf("not checking cache files of %s sectors for faults", ssize)
	}
}

type LocalPoster struct {
	localStore *stores.Local
	ls         stores.LocalStorage
	spt        abi.RegisteredSealProof
}

func (l *LocalPoster) AcquireSector(ctx context.Context, sid abi.SectorID, existing stores.SectorFileType, allocate stores.SectorFileType, ptype stores.PathType) (stores.SectorPaths, func(), error) {
	out, _, err := l.localStore.AcquireSector(ctx, sid, l.spt, existing, allocate, ptype, stores.AcquireMove)
	if err != nil {
		return out, nil, err
	}

	done := func() {}

	return out, done, nil
}

func (p *LocalPoster) StorageAddLocal(ctx context.Context, path string) error {
	path, err := homedir.Expand(path)
	if err != nil {
		return xerrors.Errorf("expanding local path: %w", err)
	}

	if err := p.localStore.OpenPath(ctx, path); err != nil {
		return xerrors.Errorf("opening local path: %w", err)
	}

	if err := p.ls.SetStorage(func(sc *stores.StorageConfig) {
		sc.StoragePaths = append(sc.StoragePaths, stores.LocalPath{Path: path})
	}); err != nil {
		return xerrors.Errorf("get storage config: %w", err)
	}

	return nil
}
