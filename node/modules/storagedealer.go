package modules

import (
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/node/config"
	"github.com/filecoin-project/lotus/node/modules/helpers"
	"github.com/filecoin-project/lotus/storage"
	"go.uber.org/fx"
)

func StorageDealer(fc config.MinerFeeConfig) func(params StorageMinerParams) (*storage.Miner, error) {
	return func(params StorageMinerParams) (*storage.Miner, error) {
		var (
			ds     = params.MetadataDS
			fds    = params.MetadataFDS
			mctx   = params.MetricsCtx
			lc     = params.Lifecycle
			api    = params.API
			sealer = params.Sealer
			h      = params.Host
			sc     = params.SectorIDCounter
			verif  = params.Verifier
			gsd    = params.GetSealingConfigFn
		)

		maddr, err := minerAddrFromDS(ds)
		if err != nil {
			return nil, err
		}

		ctx := helpers.LifecycleCtx(mctx, lc)

		mi, err := api.StateMinerInfo(ctx, maddr, types.EmptyTSK)
		if err != nil {
			return nil, err
		}

		worker, err := api.StateAccountKey(ctx, mi.Worker, types.EmptyTSK)
		if err != nil {
			return nil, err
		}
		start, err := minerStartSectorFromDS(fds)
		if err != nil {
			return nil, err
		}
		sm, err := storage.NewMiner(api, maddr, worker, h, ds, sealer, sc, verif, gsd, fc, start)
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
