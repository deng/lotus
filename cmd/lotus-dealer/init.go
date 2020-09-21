package main

import (
	"context"
	"os"

	"github.com/ipfs/go-datastore"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/extern/sector-storage/stores"

	lapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/build"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/lotus/node/repo"
)

var initCmd = &cli.Command{
	Name:  "init",
	Usage: "Initialize a lotus dealer repo",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "actor",
			Usage: "specify the address of an already created miner actor",
		},
		&cli.BoolFlag{
			Name:  "nosync",
			Usage: "don't check full-node sync status",
		},
	},
	Action: func(cctx *cli.Context) error {
		log.Info("Initializing lotus dealer")

		ctx := lcli.ReqContext(cctx)

		log.Info("Trying to connect to full node RPC")

		api, closer, err := lcli.GetFullNodeAPI(cctx) // TODO: consider storing full node address in config
		if err != nil {
			return err
		}
		defer closer()

		log.Info("Checking full node sync status")

		log.Info("Checking if repo exists")

		repoPath := cctx.String(FlagDealerRepo)
		r, err := repo.NewFS(repoPath)
		if err != nil {
			return err
		}

		ok, err := r.Exists()
		if err != nil {
			return err
		}
		if ok {
			return xerrors.Errorf("repo at '%s' is already initialized", cctx.String(FlagDealerRepo))
		}

		log.Info("Checking full node version")

		v, err := api.Version(ctx)
		if err != nil {
			return err
		}

		if !v.APIVersion.EqMajorMinor(build.FullAPIVersion) {
			return xerrors.Errorf("Remote API version didn't match (expected %s, remote %s)", build.FullAPIVersion, v.APIVersion)
		}

		log.Info("Initializing repo")

		if err := r.Init(repo.StorageDealer); err != nil {
			return err
		}

		{
			lr, err := r.Lock(repo.StorageDealer)
			if err != nil {
				return err
			}

			var localPaths []stores.LocalPath

			if err := lr.SetStorage(func(sc *stores.StorageConfig) {
				sc.StoragePaths = append(sc.StoragePaths, localPaths...)
			}); err != nil {
				return xerrors.Errorf("set storage config: %w", err)
			}

			if err := lr.Close(); err != nil {
				return err
			}
		}

		if err := storageDealerInit(ctx, cctx, api, r); err != nil {
			log.Errorf("Failed to initialize lotus-dealer: %+v", err)
			path, err := homedir.Expand(repoPath)
			if err != nil {
				return err
			}
			log.Infof("Cleaning up %s after attempt...", path)
			if err := os.RemoveAll(path); err != nil {
				log.Errorf("Failed to clean up failed storage repo: %s", err)
			}
			return xerrors.Errorf("Storage-dealer init failed")
		}

		// TODO: Point to setting storage price, maybe do it interactively or something
		log.Info("Dealer successfully created, you can now start it with 'lotus-dealer run'")

		return nil
	},
}

func storageDealerInit(ctx context.Context, cctx *cli.Context, api lapi.FullNode, r repo.Repo) error {
	lr, err := r.Lock(repo.StorageDealer)
	if err != nil {
		return err
	}
	defer lr.Close() //nolint:errcheck

	log.Info("Initializing libp2p identity")

	fsmds, err := lr.Datastore("/metadata")
	if err != nil {
		return err
	}

	var addr address.Address
	if act := cctx.String("actor"); act != "" {
		a, err := address.NewFromString(act)
		if err != nil {
			return xerrors.Errorf("failed parsing actor flag value (%q): %w", act, err)
		}

		addr = a
	} else {
		return xerrors.Errorf("actor flag value is required")
	}

	log.Infof("Created new dealer: %s", addr)
	if err := fsmds.Put(datastore.NewKey("miner-address"), addr.Bytes()); err != nil {
		return err
	}
	return nil
}