package main

import (
	"context"
	"fmt"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/trace"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/build"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/lotus/lib/lotuslog"
	"github.com/filecoin-project/lotus/lib/tracing"
	"github.com/filecoin-project/lotus/node/repo"
)

var log = logging.Logger("main")

const FlagDealerRepo = "dealer-repo"
const FlagPostgresURL = "postgres-url"
const FlagMinerRepo = "miner-repo"
const FlagSealerRepo = "sealer-repo"

const FlagMinerRepoDeprecation = "storagerepo"

func main() {
	build.RunningNodeType = build.NodeMiner

	lotuslog.SetupLogLevels()

	local := []*cli.Command{
		initCmd,
		runCmd,
		stopCmd,
		configCmd,
		lcli.WithCategory("chain", actorCmd),
		lcli.WithCategory("chain", infoCmd),
		lcli.WithCategory("market", storageDealsCmd),
		lcli.WithCategory("market", retrievalDealsCmd),
		lcli.WithCategory("market", dataTransfersCmd),
		lcli.WithCategory("storage", sectorsCmd),
		lcli.WithCategory("storage", storageCmd),
		lcli.WithCategory("storage", sealingCmd),
		lcli.WithCategory("retrieval", piecesCmd),
	}
	jaeger := tracing.SetupJaegerTracing("lotus-dealer")
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
			jaeger = tracing.SetupJaegerTracing("lotus-dealer/" + cmd.Name)

			if originBefore != nil {
				return originBefore(cctx)
			}
			return nil
		}
	}

	app := &cli.App{
		Name:                 "lotus-dealer",
		Usage:                "Filecoin decentralized storage network dealer",
		Version:              build.UserVersion(),
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "actor",
				Value:   "",
				Usage:   "specify other actor to check state for (read only)",
				Aliases: []string{"a"},
			},
			&cli.BoolFlag{
				Name: "color",
			},
			&cli.StringFlag{
				Name:    "repo",
				EnvVars: []string{"LOTUS_PATH"},
				Hidden:  true,
				Value:   "~/.lotus", // TODO: Consider XDG_DATA_HOME
			},
			&cli.StringFlag{
				Name:    FlagPostgresURL,
				EnvVars: []string{"POSTGRES_URL"},
				Value:   "",
				Usage:   "use PostgreSQL as the Datastore, eg: postgres://postgres:123456@127.0.0.1:5432/postgres?sslmode=disable",
			},
			&cli.StringFlag{
				Name:    FlagMinerRepo,
				Aliases: []string{FlagMinerRepoDeprecation, FlagSealerRepo},
				EnvVars: []string{"LOTUS_MINER_PATH", "LOTUS_STORAGE_PATH", "LOTUS_SEALER_PATH"},
				Value:   "~/.lotusminer", // TODO: Consider XDG_DATA_HOME
				Usage:   fmt.Sprintf("Specify miner/sealer repo path"),
			},
			&cli.StringFlag{
				Name:    FlagDealerRepo,
				EnvVars: []string{"LOTUS_DEALER_PATH"},
				Value:   "~/.lotusdealer", // TODO: Consider XDG_DATA_HOME
				Usage:   fmt.Sprintf("Specify miner repo path"),
			},
		},

		Commands: append(local, lcli.CommonCommands...),
	}
	app.Setup()
	app.Metadata["repoType"] = repo.StorageDealer

	lcli.RunApp(app)
}

func getActorAddress(ctx context.Context, nodeAPI api.StorageSealer, overrideMaddr string) (maddr address.Address, err error) {
	if overrideMaddr != "" {
		maddr, err = address.NewFromString(overrideMaddr)
		if err != nil {
			return maddr, err
		}
		return
	}

	maddr, err = nodeAPI.ActorAddress(ctx)
	if err != nil {
		return maddr, xerrors.Errorf("getting actor address: %w", err)
	}

	return maddr, nil
}
