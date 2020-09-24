package modules

import (
	"errors"
	"time"

	"go.uber.org/fx"
	"go.uber.org/multierr"
	"golang.org/x/xerrors"

	lapi "github.com/filecoin-project/lotus/api"
	sectorstorage "github.com/filecoin-project/lotus/extern/sector-storage"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/lotus/extern/sector-storage/stores"
	sealing "github.com/filecoin-project/lotus/extern/storage-sealing"
	"github.com/filecoin-project/lotus/extern/storage-sealing/sealiface"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/host"

	"github.com/filecoin-project/lotus/node/config"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/modules/helpers"
	"github.com/filecoin-project/lotus/node/repo"
)

type StorageDealerParams struct {
	fx.In

	Lifecycle          fx.Lifecycle
	MetricsCtx         helpers.MetricsCtx
	API                lapi.FullNode
	SealingAPI         lapi.StorageMiner
	Host               host.Host
	MetadataDS         dtypes.MetadataDS
	Sealer             sectorstorage.SectorManager
	SectorIDCounter    sealing.SectorIDCounter
	Verifier           ffiwrapper.Verifier
	GetSealingConfigFn dtypes.GetSealingConfigFunc
}

func DealerSectorStorage(mctx helpers.MetricsCtx, lc fx.Lifecycle, ls stores.LocalStorage, si stores.SectorIndex, cfg *ffiwrapper.Config, sc sectorstorage.SealerConfig, urls sectorstorage.URLs, sa sectorstorage.StorageAuth) (*sectorstorage.Manager, error) {
	return nil, nil
}

func DealerNewConsiderOnlineStorageDealsConfigFunc(r repo.LockedRepo) (dtypes.ConsiderOnlineStorageDealsConfigFunc, error) {
	return func() (out bool, err error) {
		err = readDealerCfg(r, func(cfg *config.StorageDealer) {
			out = cfg.Dealmaking.ConsiderOnlineStorageDeals
		})
		return
	}, nil
}

func DealerNewSetConsideringOnlineStorageDealsFunc(r repo.LockedRepo) (dtypes.SetConsiderOnlineStorageDealsConfigFunc, error) {
	return func(b bool) (err error) {
		err = mutateDealerCfg(r, func(cfg *config.StorageDealer) {
			cfg.Dealmaking.ConsiderOnlineStorageDeals = b
		})
		return
	}, nil
}

func DealerNewConsiderOnlineRetrievalDealsConfigFunc(r repo.LockedRepo) (dtypes.ConsiderOnlineRetrievalDealsConfigFunc, error) {
	return func() (out bool, err error) {
		err = readDealerCfg(r, func(cfg *config.StorageDealer) {
			out = cfg.Dealmaking.ConsiderOnlineRetrievalDeals
		})
		return
	}, nil
}

func DealerNewSetConsiderOnlineRetrievalDealsConfigFunc(r repo.LockedRepo) (dtypes.SetConsiderOnlineRetrievalDealsConfigFunc, error) {
	return func(b bool) (err error) {
		err = mutateDealerCfg(r, func(cfg *config.StorageDealer) {
			cfg.Dealmaking.ConsiderOnlineRetrievalDeals = b
		})
		return
	}, nil
}

func DealerNewStorageDealPieceCidBlocklistConfigFunc(r repo.LockedRepo) (dtypes.StorageDealPieceCidBlocklistConfigFunc, error) {
	return func() (out []cid.Cid, err error) {
		err = readDealerCfg(r, func(cfg *config.StorageDealer) {
			out = cfg.Dealmaking.PieceCidBlocklist
		})
		return
	}, nil
}

func DealerNewSetStorageDealPieceCidBlocklistConfigFunc(r repo.LockedRepo) (dtypes.SetStorageDealPieceCidBlocklistConfigFunc, error) {
	return func(blocklist []cid.Cid) (err error) {
		err = mutateDealerCfg(r, func(cfg *config.StorageDealer) {
			cfg.Dealmaking.PieceCidBlocklist = blocklist
		})
		return
	}, nil
}

func DealerNewConsiderOfflineStorageDealsConfigFunc(r repo.LockedRepo) (dtypes.ConsiderOfflineStorageDealsConfigFunc, error) {
	return func() (out bool, err error) {
		err = readDealerCfg(r, func(cfg *config.StorageDealer) {
			out = cfg.Dealmaking.ConsiderOfflineStorageDeals
		})
		return
	}, nil
}

func DealerNewSetConsideringOfflineStorageDealsFunc(r repo.LockedRepo) (dtypes.SetConsiderOfflineStorageDealsConfigFunc, error) {
	return func(b bool) (err error) {
		err = mutateDealerCfg(r, func(cfg *config.StorageDealer) {
			cfg.Dealmaking.ConsiderOfflineStorageDeals = b
		})
		return
	}, nil
}

func DealerNewConsiderOfflineRetrievalDealsConfigFunc(r repo.LockedRepo) (dtypes.ConsiderOfflineRetrievalDealsConfigFunc, error) {
	return func() (out bool, err error) {
		err = readDealerCfg(r, func(cfg *config.StorageDealer) {
			out = cfg.Dealmaking.ConsiderOfflineRetrievalDeals
		})
		return
	}, nil
}

func DealerNewSetConsiderOfflineRetrievalDealsConfigFunc(r repo.LockedRepo) (dtypes.SetConsiderOfflineRetrievalDealsConfigFunc, error) {
	return func(b bool) (err error) {
		err = mutateDealerCfg(r, func(cfg *config.StorageDealer) {
			cfg.Dealmaking.ConsiderOfflineRetrievalDeals = b
		})
		return
	}, nil
}

func DealerNewSetSealConfigFunc(r repo.LockedRepo) (dtypes.SetSealingConfigFunc, error) {
	return func(cfg sealiface.Config) (err error) {
		err = mutateDealerCfg(r, func(c *config.StorageDealer) {})
		return
	}, nil
}

func DealerNewGetSealConfigFunc(r repo.LockedRepo) (dtypes.GetSealingConfigFunc, error) {
	return func() (out sealiface.Config, err error) {
		err = readDealerCfg(r, func(cfg *config.StorageDealer) {})
		return
	}, nil
}

func DealerNewSetExpectedSealDurationFunc(r repo.LockedRepo) (dtypes.SetExpectedSealDurationFunc, error) {
	return func(delay time.Duration) (err error) {
		err = mutateDealerCfg(r, func(cfg *config.StorageDealer) {
			cfg.Dealmaking.ExpectedSealDuration = config.Duration(delay)
		})
		return
	}, nil
}

func DealerNewGetExpectedSealDurationFunc(r repo.LockedRepo) (dtypes.GetExpectedSealDurationFunc, error) {
	return func() (out time.Duration, err error) {
		err = readDealerCfg(r, func(cfg *config.StorageDealer) {
			out = time.Duration(cfg.Dealmaking.ExpectedSealDuration)
		})
		return
	}, nil
}

func readDealerCfg(r repo.LockedRepo, accessor func(*config.StorageDealer)) error {
	raw, err := r.Config()
	if err != nil {
		return err
	}

	cfg, ok := raw.(*config.StorageDealer)
	if !ok {
		return xerrors.New("expected address of config.StorageDealer")
	}

	accessor(cfg)

	return nil
}

func mutateDealerCfg(r repo.LockedRepo, mutator func(*config.StorageDealer)) error {
	var typeErr error

	setConfigErr := r.SetConfig(func(raw interface{}) {
		cfg, ok := raw.(*config.StorageDealer)
		if !ok {
			typeErr = errors.New("expected dealer config")
			return
		}

		mutator(cfg)
	})

	return multierr.Combine(typeErr, setConfigErr)
}
