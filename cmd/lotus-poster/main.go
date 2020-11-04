package main

import (
	"fmt"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/lib/lotuslog"
	"github.com/filecoin-project/lotus/lib/tracing"
	"github.com/filecoin-project/lotus/node/repo"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/trace"
	"golang.org/x/xerrors"
	"net"
	"os"
	"strings"
	"time"
)

var log = logging.Logger("main")

const FlagPosterRepo = "poster-repo"

func main() {
	build.RunningNodeType = build.NodeWorker //现在我们把这个wdpost服务当成一个worker来对待
	lotuslog.SetupLogLevels()
	local := []*cli.Command{
		localCmd,
		provingCmd,
		storageCmd,
	}
	jaeger := tracing.SetupJaegerTracing("lotus-poster")
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
		Name:                 "lotus-poster",
		Usage:                "Filecoin decentralized window poster",
		Version:              build.UserVersion(),
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    FlagPosterRepo,
				EnvVars: []string{"LOTUS_POSTER_PATH", "POSTER_PATH"},
				Value:   "~/.lotusposter", // TODO: Consider XDG_DATA_HOME
				Usage:   fmt.Sprintf("Specify wdposter repo path. flag %s ", FlagPosterRepo),
			},
			&cli.BoolFlag{
				Name:  "enable-gpu-proving",
				Usage: "enable use of GPU for poster operations",
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
