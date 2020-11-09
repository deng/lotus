package modules

import (
	"encoding/binary"
	"github.com/filecoin-project/lotus/node/config"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/storage"
	"github.com/ipfs/go-datastore"
	"go.uber.org/fx"
	"os"
)

var StorageSectorStart = "sector-start"

func minerStartSectorFromDS(fds dtypes.MetadataFDS) (uint64, error) {
	curBytes, err := fds.Get(datastore.NewKey(StorageSectorStart))
	if err != nil {
		return 0, err
	}
	start, _ := binary.Uvarint(curBytes)
	return start, nil
}

func StorageSealer(fc config.MinerFeeConfig) func(params StorageMinerParams) (*storage.Miner, error) {
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
