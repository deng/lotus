package impl

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/host"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	datatransfer "github.com/filecoin-project/go-data-transfer"
	retrievalmarket "github.com/filecoin-project/go-fil-markets/retrievalmarket"
	storagemarket "github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/apistruct"
	"github.com/filecoin-project/lotus/chain/types"
	sectorstorage "github.com/filecoin-project/lotus/extern/sector-storage"
	"github.com/filecoin-project/lotus/node/impl/common"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/storage"
)

type StorageDealerAPI struct {
	common.CommonAPI

	StorageProvider   storagemarket.StorageProvider
	RetrievalProvider retrievalmarket.RetrievalProvider
	Dealer            *storage.Dealer
	Full              api.FullNode
	StorageMgr        *sectorstorage.Manager `optional:"true"`
	DataTransfer      dtypes.ProviderDataTransfer
	Host              host.Host

	ConsiderOnlineStorageDealsConfigFunc       dtypes.ConsiderOnlineStorageDealsConfigFunc
	SetConsiderOnlineStorageDealsConfigFunc    dtypes.SetConsiderOnlineStorageDealsConfigFunc
	ConsiderOnlineRetrievalDealsConfigFunc     dtypes.ConsiderOnlineRetrievalDealsConfigFunc
	SetConsiderOnlineRetrievalDealsConfigFunc  dtypes.SetConsiderOnlineRetrievalDealsConfigFunc
	StorageDealPieceCidBlocklistConfigFunc     dtypes.StorageDealPieceCidBlocklistConfigFunc
	SetStorageDealPieceCidBlocklistConfigFunc  dtypes.SetStorageDealPieceCidBlocklistConfigFunc
	ConsiderOfflineStorageDealsConfigFunc      dtypes.ConsiderOfflineStorageDealsConfigFunc
	SetConsiderOfflineStorageDealsConfigFunc   dtypes.SetConsiderOfflineStorageDealsConfigFunc
	ConsiderOfflineRetrievalDealsConfigFunc    dtypes.ConsiderOfflineRetrievalDealsConfigFunc
	SetConsiderOfflineRetrievalDealsConfigFunc dtypes.SetConsiderOfflineRetrievalDealsConfigFunc
	SetSealingConfigFunc                       dtypes.SetSealingConfigFunc
	GetSealingConfigFunc                       dtypes.GetSealingConfigFunc
	GetExpectedSealDurationFunc                dtypes.GetExpectedSealDurationFunc
	SetExpectedSealDurationFunc                dtypes.SetExpectedSealDurationFunc
}

func (sm *StorageDealerAPI) ActorAddress(context.Context) (address.Address, error) {
	return sm.Dealer.ActorAddress(), nil
}

func (sm *StorageDealerAPI) SectorSetExpectedSealDuration(ctx context.Context, delay time.Duration) error {
	return sm.SetExpectedSealDurationFunc(delay)
}

func (sm *StorageDealerAPI) DealsConsiderOfflineRetrievalDeals(ctx context.Context) (bool, error) {
	return sm.ConsiderOfflineRetrievalDealsConfigFunc()
}

func (sm *StorageDealerAPI) DealsConsiderOfflineStorageDeals(ctx context.Context) (bool, error) {
	return sm.ConsiderOfflineStorageDealsConfigFunc()
}

func (sm *StorageDealerAPI) DealsConsiderOnlineRetrievalDeals(ctx context.Context) (bool, error) {
	return sm.ConsiderOnlineRetrievalDealsConfigFunc()
}

func (sm *StorageDealerAPI) DealsConsiderOnlineStorageDeals(ctx context.Context) (bool, error) {
	return sm.ConsiderOnlineStorageDealsConfigFunc()
}

func (sm *StorageDealerAPI) DealsImportData(ctx context.Context, deal cid.Cid, fname string) error {
	fi, err := os.Open(fname)
	if err != nil {
		return xerrors.Errorf("failed to open given file: %w", err)
	}
	defer fi.Close() //nolint:errcheck

	return sm.StorageProvider.ImportDataForDeal(ctx, deal, fi)
}

func (sm *StorageDealerAPI) DealsList(ctx context.Context) ([]storagemarket.StorageDeal, error) {
	return sm.StorageProvider.ListDeals(ctx)
}

func (sm *StorageDealerAPI) DealsPieceCidBlocklist(ctx context.Context) ([]cid.Cid, error) {
	return sm.StorageDealPieceCidBlocklistConfigFunc()
}

func (sm *StorageDealerAPI) DealsSetConsiderOfflineRetrievalDeals(ctx context.Context, b bool) error {
	return sm.SetConsiderOfflineRetrievalDealsConfigFunc(b)
}

func (sm *StorageDealerAPI) DealsSetConsiderOfflineStorageDeals(ctx context.Context, b bool) error {
	return sm.SetConsiderOfflineStorageDealsConfigFunc(b)
}

func (sm *StorageDealerAPI) DealsSetConsiderOnlineRetrievalDeals(ctx context.Context, b bool) error {
	return sm.SetConsiderOnlineRetrievalDealsConfigFunc(b)
}

func (sm *StorageDealerAPI) DealsSetConsiderOnlineStorageDeals(ctx context.Context, b bool) error {
	return sm.SetConsiderOnlineStorageDealsConfigFunc(b)
}

func (sm *StorageDealerAPI) DealsSetPieceCidBlocklist(ctx context.Context, cids []cid.Cid) error {
	return sm.SetStorageDealPieceCidBlocklistConfigFunc(cids)
}

func (sm *StorageDealerAPI) MarketDataTransferUpdates(ctx context.Context) (<-chan api.DataTransferChannel, error) {
	channels := make(chan api.DataTransferChannel)

	unsub := sm.DataTransfer.SubscribeToEvents(func(evt datatransfer.Event, channelState datatransfer.ChannelState) {
		channel := api.NewDataTransferChannel(sm.Host.ID(), channelState)
		select {
		case <-ctx.Done():
		case channels <- channel:
		}
	})

	go func() {
		defer unsub()
		<-ctx.Done()
	}()

	return channels, nil
}

func (sm *StorageDealerAPI) MarketGetAsk(ctx context.Context) (*storagemarket.SignedStorageAsk, error) {
	return sm.StorageProvider.GetAsk(), nil
}

func (sm *StorageDealerAPI) MarketGetDealUpdates(ctx context.Context) (<-chan storagemarket.MinerDeal, error) {
	results := make(chan storagemarket.MinerDeal)
	unsub := sm.StorageProvider.SubscribeToEvents(func(evt storagemarket.ProviderEvent, deal storagemarket.MinerDeal) {
		select {
		case results <- deal:
		case <-ctx.Done():
		}
	})
	go func() {
		<-ctx.Done()
		unsub()
		close(results)
	}()
	return results, nil
}

func (sm *StorageDealerAPI) MarketGetRetrievalAsk(ctx context.Context) (*retrievalmarket.Ask, error) {
	return sm.RetrievalProvider.GetAsk(), nil
}

func (sm *StorageDealerAPI) MarketImportDealData(ctx context.Context, propCid cid.Cid, path string) error {
	fi, err := os.Open(path)
	if err != nil {
		return xerrors.Errorf("failed to open file: %w", err)
	}
	defer fi.Close() //nolint:errcheck

	return sm.StorageProvider.ImportDataForDeal(ctx, propCid, fi)
}

func (sm *StorageDealerAPI) MarketListDataTransfers(ctx context.Context) ([]api.DataTransferChannel, error) {
	inProgressChannels, err := sm.DataTransfer.InProgressChannels(ctx)
	if err != nil {
		return nil, err
	}

	apiChannels := make([]api.DataTransferChannel, 0, len(inProgressChannels))
	for _, channelState := range inProgressChannels {
		apiChannels = append(apiChannels, api.NewDataTransferChannel(sm.Host.ID(), channelState))
	}

	return apiChannels, nil
}

func (sm *StorageDealerAPI) MarketListDeals(ctx context.Context) ([]storagemarket.StorageDeal, error) {
	return sm.StorageProvider.ListDeals(ctx)
}

func (sm *StorageDealerAPI) MarketListIncompleteDeals(ctx context.Context) ([]storagemarket.MinerDeal, error) {
	return sm.StorageProvider.ListLocalDeals()
}

func (sm *StorageDealerAPI) MarketListRetrievalDeals(ctx context.Context) ([]retrievalmarket.ProviderDealState, error) {
	var out []retrievalmarket.ProviderDealState
	deals := sm.RetrievalProvider.ListDeals()

	for _, deal := range deals {
		out = append(out, deal)
	}

	return out, nil
}

func (sm *StorageDealerAPI) MarketSetAsk(ctx context.Context, price types.BigInt, verifiedPrice types.BigInt, duration abi.ChainEpoch, minPieceSize abi.PaddedPieceSize, maxPieceSize abi.PaddedPieceSize) error {
	options := []storagemarket.StorageAskOption{
		storagemarket.MinPieceSize(minPieceSize),
		storagemarket.MaxPieceSize(maxPieceSize),
	}

	return sm.StorageProvider.SetAsk(price, verifiedPrice, duration, options...)
}

func (sm *StorageDealerAPI) MarketSetRetrievalAsk(ctx context.Context, rask *retrievalmarket.Ask) error {
	sm.RetrievalProvider.SetAsk(rask)
	return nil
}

func (sm *StorageDealerAPI) ServeRemote(w http.ResponseWriter, r *http.Request) {
	if !auth.HasPerm(r.Context(), nil, apistruct.PermAdmin) {
		w.WriteHeader(401)
		_ = json.NewEncoder(w).Encode(struct{ Error string }{"unauthorized: missing write permission"})
		return
	}

	sm.StorageMgr.ServeHTTP(w, r)
}

func (sm *StorageDealerAPI) ActorSectorSize(ctx context.Context, addr address.Address) (abi.SectorSize, error) {
	mi, err := sm.Full.StateMinerInfo(ctx, addr, types.EmptyTSK)
	if err != nil {
		return 0, err
	}
	return mi.SectorSize, nil
}
