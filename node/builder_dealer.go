package node

import (
	"errors"

	"github.com/filecoin-project/lotus/api"
	sectorstorage "github.com/filecoin-project/lotus/extern/sector-storage"
	"github.com/filecoin-project/lotus/markets/dealfilter"
	"github.com/filecoin-project/lotus/node/config"
	"github.com/filecoin-project/lotus/node/impl"
	"github.com/filecoin-project/lotus/node/modules"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/repo"
	"github.com/filecoin-project/lotus/storage"
	"go.uber.org/fx"
	"golang.org/x/xerrors"
)

func StorageDealer(out *api.StorageDealer) Option {
	return Options(
		ApplyIf(func(s *Settings) bool { return s.Config },
			Error(errors.New("the StorageDealer option must be set before Config option")),
		),
		ApplyIf(func(s *Settings) bool { return s.Online },
			Error(errors.New("the StorageDealer option must be set before Online option")),
		),

		func(s *Settings) error {
			s.nodeType = repo.StorageDealer
			return nil
		},

		func(s *Settings) error {
			resAPI := &impl.StorageDealerAPI{}
			s.invokes[ExtractApiKey] = fx.Populate(resAPI)
			*out = resAPI
			return nil
		},
	)
}

func ConfigStorageDealer(c interface{}) Option {
	cfg, ok := c.(*config.StorageDealer)
	if !ok {
		return Error(xerrors.Errorf("invalid config from repo, got: %T", c))
	}

	return Options(
		ConfigCommon(&cfg.Common),

		If(cfg.Dealmaking.Filter != "",
			Override(new(dtypes.StorageDealFilter), modules.BasicDealFilter(dealfilter.CliStorageDealFilter(cfg.Dealmaking.Filter))),
		),

		If(cfg.Dealmaking.RetrievalFilter != "",
			Override(new(dtypes.RetrievalDealFilter), modules.RetrievalDealFilter(dealfilter.CliRetrievalDealFilter(cfg.Dealmaking.RetrievalFilter))),
		),

		//内置默认配置，不启用任何密封功能
		Override(new(sectorstorage.SealerConfig), sectorstorage.SealerConfig{
			ParallelFetchLimit: 1,
			AllowAddPiece:      false,
			AllowPreCommit1:    false,
			AllowPreCommit2:    false,
			AllowCommit:        false,
			AllowUnseal:        false,
		}),
		Override(new(*storage.Dealer), modules.StorageDealer(cfg.Fees)),
	)
}
