package modules

import (
	"github.com/filecoin-project/lotus/node/config"
	"github.com/filecoin-project/lotus/storage"
	"go.uber.org/fx"
	"os"
)

func StorageDealer(fc config.MinerFeeConfig) func(params StorageMinerParams) (*storage.Miner, error) {
	return func(params StorageMinerParams) (*storage.Miner, error) {
		var (
			ds     = params.MetadataDS
			fds    = params.MetadataFDS
			lc     = params.Lifecycle
			api    = params.API
			sealer = params.Sealer
			h      = params.Host
			sc     = params.SectorIDCounter
			verif  = params.Verifier
			gsd    = params.GetSealingConfigFn
			j      = params.Journal
		)

		maddr, err := minerAddrFromDS(ds)
		if err != nil {
			return nil, err
		}

		if err != nil {
			return nil, err
		}
		var start uint64 = 0
		if _, ok := os.LookupEnv("POSTGRES_URL"); ok {
			start, err = minerStartSectorFromDS(fds)
			if err != nil {
				return nil, err
			}
		}
		sm, err := storage.NewMiner(api, maddr, h, ds, sealer, sc, verif, gsd, fc, j, start)
		if err != nil {
			return nil, err
		}

		lc.Append(fx.Hook{
			OnStart: sm.Run,
			OnStop:  sm.Stop,
		})

		return sm, nil
	}
}
