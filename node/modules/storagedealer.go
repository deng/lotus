package modules

import (
	"context"
	"errors"
	"time"

	"go.uber.org/fx"
	"go.uber.org/multierr"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-fil-markets/retrievalmarket"
	retrievalimpl "github.com/filecoin-project/go-fil-markets/retrievalmarket/impl"
	rmnet "github.com/filecoin-project/go-fil-markets/retrievalmarket/network"
	lapi "github.com/filecoin-project/lotus/api"
	sectorstorage "github.com/filecoin-project/lotus/extern/sector-storage"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/lotus/extern/sector-storage/stores"
	sealing "github.com/filecoin-project/lotus/extern/storage-sealing"
	"github.com/filecoin-project/lotus/extern/storage-sealing/sealiface"
	"github.com/filecoin-project/lotus/markets/retrievaladapter"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
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

// RetrievalProvider creates a new retrieval provider attached to the provider blockstore
func RetrievalProviderDealer(h host.Host, sealingApi lapi.StorageMiner, full lapi.FullNode, ds dtypes.MetadataDS, pieceStore dtypes.ProviderPieceStore, mds dtypes.StagingMultiDstore, dt dtypes.ProviderDataTransfer, onlineOk dtypes.ConsiderOnlineRetrievalDealsConfigFunc, offlineOk dtypes.ConsiderOfflineRetrievalDealsConfigFunc) (retrievalmarket.RetrievalProvider, error) {
	adapter := retrievaladapter.NewRetrievalProviderNodeDealer(sealingApi, full)

	maddr, err := minerAddrFromDS(ds)
	if err != nil {
		return nil, err
	}

	netwk := rmnet.NewFromLibp2pHost(h)

	opt := retrievalimpl.DealDeciderOpt(func(ctx context.Context, state retrievalmarket.ProviderDealState) (bool, string, error) {
		b, err := onlineOk()
		if err != nil {
			return false, "dealer error", err
		}

		if !b {
			log.Warn("online retrieval deal consideration disabled; rejecting retrieval deal proposal from client")
			return false, "dealer is not accepting online retrieval deals", nil
		}

		b, err = offlineOk()
		if err != nil {
			return false, "dealer error", err
		}

		if !b {
			log.Info("offline retrieval has not been implemented yet")
		}

		return true, "", nil
	})

	return retrievalimpl.NewProvider(maddr, adapter, netwk, pieceStore, mds, dt, namespace.Wrap(ds, datastore.NewKey("/retrievals/provider")), opt)
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
