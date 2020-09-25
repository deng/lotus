package apistruct

import (
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-fil-markets/piecestore"
	"github.com/filecoin-project/go-fil-markets/retrievalmarket"
	"github.com/filecoin-project/go-fil-markets/shared"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/extern/sector-storage/fsutil"
	"github.com/filecoin-project/lotus/extern/sector-storage/stores"
	"github.com/filecoin-project/lotus/extern/sector-storage/storiface"
	sealing "github.com/filecoin-project/lotus/extern/storage-sealing"
	"github.com/ipfs/go-cid"
	"io"
	"time"
)

type StorageSealerStruct struct {
	CommonStruct

	Internal struct {
		ActorAddress    func(context.Context) (address.Address, error)                 `perm:"read"`
		ActorSectorSize func(context.Context, address.Address) (abi.SectorSize, error) `perm:"read"`

		MarketImportDealData      func(context.Context, cid.Cid, string) error                                                                                                                                 `perm:"write"`
		MarketListDeals           func(ctx context.Context) ([]storagemarket.StorageDeal, error)                                                                                                               `perm:"read"`
		MarketListRetrievalDeals  func(ctx context.Context) ([]retrievalmarket.ProviderDealState, error)                                                                                                       `perm:"read"`
		MarketGetDealUpdates      func(ctx context.Context) (<-chan storagemarket.MinerDeal, error)                                                                                                            `perm:"read"`
		MarketListIncompleteDeals func(ctx context.Context) ([]storagemarket.MinerDeal, error)                                                                                                                 `perm:"read"`
		MarketSetAsk              func(ctx context.Context, price types.BigInt, verifiedPrice types.BigInt, duration abi.ChainEpoch, minPieceSize abi.PaddedPieceSize, maxPieceSize abi.PaddedPieceSize) error `perm:"admin"`
		MarketGetAsk              func(ctx context.Context) (*storagemarket.SignedStorageAsk, error)                                                                                                           `perm:"read"`
		MarketSetRetrievalAsk     func(ctx context.Context, rask *retrievalmarket.Ask) error                                                                                                                   `perm:"admin"`
		MarketGetRetrievalAsk     func(ctx context.Context) (*retrievalmarket.Ask, error)                                                                                                                      `perm:"read"`
		MarketListDataTransfers   func(ctx context.Context) ([]api.DataTransferChannel, error)                                                                                                                 `perm:"write"`
		MarketDataTransferUpdates func(ctx context.Context) (<-chan api.DataTransferChannel, error)                                                                                                            `perm:"write"`

		PledgeSector func(context.Context) error `perm:"write"`

		SectorsStatus                 func(ctx context.Context, sid abi.SectorNumber, showOnChainInfo bool) (api.SectorInfo, error) `perm:"read"`
		SectorsList                   func(context.Context) ([]abi.SectorNumber, error)                                             `perm:"read"`
		SectorsRefs                   func(context.Context) (map[string][]api.SealedRef, error)                                     `perm:"read"`
		SectorStartSealing            func(context.Context, abi.SectorNumber) error                                                 `perm:"write"`
		SectorSetSealDelay            func(context.Context, time.Duration) error                                                    `perm:"write"`
		SectorGetSealDelay            func(context.Context) (time.Duration, error)                                                  `perm:"read"`
		SectorSetExpectedSealDuration func(context.Context, time.Duration) error                                                    `perm:"write"`
		SectorGetExpectedSealDuration func(context.Context) (time.Duration, error)                                                  `perm:"read"`
		SectorsUpdate                 func(context.Context, abi.SectorNumber, api.SectorState) error                                `perm:"admin"`
		SectorRemove                  func(context.Context, abi.SectorNumber) error                                                 `perm:"admin"`
		SectorMarkForUpgrade          func(ctx context.Context, id abi.SectorNumber) error                                          `perm:"admin"`

		WorkerConnect func(context.Context, string) error                             `perm:"admin"` // TODO: worker perm
		WorkerStats   func(context.Context) (map[uint64]storiface.WorkerStats, error) `perm:"admin"`
		WorkerJobs    func(context.Context) (map[uint64][]storiface.WorkerJob, error) `perm:"admin"`

		SealingSchedDiag func(context.Context) (interface{}, error) `perm:"admin"`

		StorageList          func(context.Context) (map[stores.ID][]stores.Decl, error)                                                                                    `perm:"admin"`
		StorageLocal         func(context.Context) (map[stores.ID]string, error)                                                                                           `perm:"admin"`
		StorageStat          func(context.Context, stores.ID) (fsutil.FsStat, error)                                                                                       `perm:"admin"`
		StorageAttach        func(context.Context, stores.StorageInfo, fsutil.FsStat) error                                                                                `perm:"admin"`
		StorageDeclareSector func(context.Context, stores.ID, abi.SectorID, stores.SectorFileType, bool) error                                                             `perm:"admin"`
		StorageDropSector    func(context.Context, stores.ID, abi.SectorID, stores.SectorFileType) error                                                                   `perm:"admin"`
		StorageFindSector    func(context.Context, abi.SectorID, stores.SectorFileType, abi.RegisteredSealProof, bool) ([]stores.SectorStorageInfo, error)                 `perm:"admin"`
		StorageInfo          func(context.Context, stores.ID) (stores.StorageInfo, error)                                                                                  `perm:"admin"`
		StorageBestAlloc     func(ctx context.Context, allocate stores.SectorFileType, spt abi.RegisteredSealProof, sealing stores.PathType) ([]stores.StorageInfo, error) `perm:"admin"`
		StorageReportHealth  func(ctx context.Context, id stores.ID, report stores.HealthReport) error                                                                     `perm:"admin"`
		StorageLock          func(ctx context.Context, sector abi.SectorID, read stores.SectorFileType, write stores.SectorFileType) error                                 `perm:"admin"`
		StorageTryLock       func(ctx context.Context, sector abi.SectorID, read stores.SectorFileType, write stores.SectorFileType) (bool, error)                         `perm:"admin"`

		StorageAddLocal func(ctx context.Context, path string) error `perm:"admin"`

		PiecesListPieces   func(ctx context.Context) ([]cid.Cid, error)                               `perm:"read"`
		PiecesListCidInfos func(ctx context.Context) ([]cid.Cid, error)                               `perm:"read"`
		PiecesGetPieceInfo func(ctx context.Context, pieceCid cid.Cid) (*piecestore.PieceInfo, error) `perm:"read"`
		PiecesGetCIDInfo   func(ctx context.Context, payloadCid cid.Cid) (*piecestore.CIDInfo, error) `perm:"read"`

		AddPieceOnDealComplete         func(ctx context.Context, size abi.UnpaddedPieceSize, r io.Reader, d sealing.DealInfo) (*storagemarket.PackingResult, error)            `perm:"admin"`
		LocatePieceForDealWithinSector func(ctx context.Context, dealID abi.DealID, encodedTs shared.TipSetToken) (*api.LocatePieceResult, error)                              `perm:"admin"`
		UnsealSector                   func(ctx context.Context, sectorID abi.SectorNumber, offset abi.UnpaddedPieceSize, length abi.UnpaddedPieceSize) (io.ReadCloser, error) `perm:"admin"`
	}
}

func (c *StorageSealerStruct) ActorAddress(ctx context.Context) (address.Address, error) {
	return c.Internal.ActorAddress(ctx)
}

func (c *StorageSealerStruct) ActorSectorSize(ctx context.Context, addr address.Address) (abi.SectorSize, error) {
	return c.Internal.ActorSectorSize(ctx, addr)
}

func (c *StorageSealerStruct) PledgeSector(ctx context.Context) error {
	return c.Internal.PledgeSector(ctx)
}

// Get the status of a given sector by ID
func (c *StorageSealerStruct) SectorsStatus(ctx context.Context, sid abi.SectorNumber, showOnChainInfo bool) (api.SectorInfo, error) {
	return c.Internal.SectorsStatus(ctx, sid, showOnChainInfo)
}

// List all staged sectors
func (c *StorageSealerStruct) SectorsList(ctx context.Context) ([]abi.SectorNumber, error) {
	return c.Internal.SectorsList(ctx)
}

func (c *StorageSealerStruct) SectorsRefs(ctx context.Context) (map[string][]api.SealedRef, error) {
	return c.Internal.SectorsRefs(ctx)
}

func (c *StorageSealerStruct) SectorStartSealing(ctx context.Context, number abi.SectorNumber) error {
	return c.Internal.SectorStartSealing(ctx, number)
}

func (c *StorageSealerStruct) SectorSetSealDelay(ctx context.Context, delay time.Duration) error {
	return c.Internal.SectorSetSealDelay(ctx, delay)
}

func (c *StorageSealerStruct) SectorGetSealDelay(ctx context.Context) (time.Duration, error) {
	return c.Internal.SectorGetSealDelay(ctx)
}

func (c *StorageSealerStruct) SectorSetExpectedSealDuration(ctx context.Context, delay time.Duration) error {
	return c.Internal.SectorSetExpectedSealDuration(ctx, delay)
}

func (c *StorageSealerStruct) SectorGetExpectedSealDuration(ctx context.Context) (time.Duration, error) {
	return c.Internal.SectorGetExpectedSealDuration(ctx)
}

func (c *StorageSealerStruct) SectorsUpdate(ctx context.Context, id abi.SectorNumber, state api.SectorState) error {
	return c.Internal.SectorsUpdate(ctx, id, state)
}

func (c *StorageSealerStruct) SectorRemove(ctx context.Context, number abi.SectorNumber) error {
	return c.Internal.SectorRemove(ctx, number)
}

func (c *StorageSealerStruct) SectorMarkForUpgrade(ctx context.Context, number abi.SectorNumber) error {
	return c.Internal.SectorMarkForUpgrade(ctx, number)
}

func (c *StorageSealerStruct) WorkerConnect(ctx context.Context, url string) error {
	return c.Internal.WorkerConnect(ctx, url)
}

func (c *StorageSealerStruct) WorkerStats(ctx context.Context) (map[uint64]storiface.WorkerStats, error) {
	return c.Internal.WorkerStats(ctx)
}

func (c *StorageSealerStruct) WorkerJobs(ctx context.Context) (map[uint64][]storiface.WorkerJob, error) {
	return c.Internal.WorkerJobs(ctx)
}

func (c *StorageSealerStruct) SealingSchedDiag(ctx context.Context) (interface{}, error) {
	return c.Internal.SealingSchedDiag(ctx)
}

func (c *StorageSealerStruct) StorageAttach(ctx context.Context, si stores.StorageInfo, st fsutil.FsStat) error {
	return c.Internal.StorageAttach(ctx, si, st)
}

func (c *StorageSealerStruct) StorageDeclareSector(ctx context.Context, storageId stores.ID, s abi.SectorID, ft stores.SectorFileType, primary bool) error {
	return c.Internal.StorageDeclareSector(ctx, storageId, s, ft, primary)
}

func (c *StorageSealerStruct) StorageDropSector(ctx context.Context, storageId stores.ID, s abi.SectorID, ft stores.SectorFileType) error {
	return c.Internal.StorageDropSector(ctx, storageId, s, ft)
}

func (c *StorageSealerStruct) StorageFindSector(ctx context.Context, si abi.SectorID, types stores.SectorFileType, spt abi.RegisteredSealProof, allowFetch bool) ([]stores.SectorStorageInfo, error) {
	return c.Internal.StorageFindSector(ctx, si, types, spt, allowFetch)
}

func (c *StorageSealerStruct) StorageList(ctx context.Context) (map[stores.ID][]stores.Decl, error) {
	return c.Internal.StorageList(ctx)
}

func (c *StorageSealerStruct) StorageLocal(ctx context.Context) (map[stores.ID]string, error) {
	return c.Internal.StorageLocal(ctx)
}

func (c *StorageSealerStruct) StorageStat(ctx context.Context, id stores.ID) (fsutil.FsStat, error) {
	return c.Internal.StorageStat(ctx, id)
}

func (c *StorageSealerStruct) StorageInfo(ctx context.Context, id stores.ID) (stores.StorageInfo, error) {
	return c.Internal.StorageInfo(ctx, id)
}

func (c *StorageSealerStruct) StorageBestAlloc(ctx context.Context, allocate stores.SectorFileType, spt abi.RegisteredSealProof, pt stores.PathType) ([]stores.StorageInfo, error) {
	return c.Internal.StorageBestAlloc(ctx, allocate, spt, pt)
}

func (c *StorageSealerStruct) StorageReportHealth(ctx context.Context, id stores.ID, report stores.HealthReport) error {
	return c.Internal.StorageReportHealth(ctx, id, report)
}

func (c *StorageSealerStruct) StorageLock(ctx context.Context, sector abi.SectorID, read stores.SectorFileType, write stores.SectorFileType) error {
	return c.Internal.StorageLock(ctx, sector, read, write)
}

func (c *StorageSealerStruct) StorageTryLock(ctx context.Context, sector abi.SectorID, read stores.SectorFileType, write stores.SectorFileType) (bool, error) {
	return c.Internal.StorageTryLock(ctx, sector, read, write)
}

func (c *StorageSealerStruct) StorageAddLocal(ctx context.Context, path string) error {
	return c.Internal.StorageAddLocal(ctx, path)
}

func (c *StorageSealerStruct) PiecesListPieces(ctx context.Context) ([]cid.Cid, error) {
	return c.Internal.PiecesListPieces(ctx)
}

func (c *StorageSealerStruct) PiecesListCidInfos(ctx context.Context) ([]cid.Cid, error) {
	return c.Internal.PiecesListCidInfos(ctx)
}

func (c *StorageSealerStruct) PiecesGetPieceInfo(ctx context.Context, pieceCid cid.Cid) (*piecestore.PieceInfo, error) {
	return c.Internal.PiecesGetPieceInfo(ctx, pieceCid)
}

func (c *StorageSealerStruct) PiecesGetCIDInfo(ctx context.Context, payloadCid cid.Cid) (*piecestore.CIDInfo, error) {
	return c.Internal.PiecesGetCIDInfo(ctx, payloadCid)
}

func (c *StorageSealerStruct) AddPieceOnDealComplete(ctx context.Context, size abi.UnpaddedPieceSize, r io.Reader, d sealing.DealInfo) (*storagemarket.PackingResult, error) {
	return c.Internal.AddPieceOnDealComplete(ctx, size, r, d)
}

func (c *StorageSealerStruct) LocatePieceForDealWithinSector(ctx context.Context, dealID abi.DealID, encodedTs shared.TipSetToken) (*api.LocatePieceResult, error) {
	return c.Internal.LocatePieceForDealWithinSector(ctx, dealID, encodedTs)
}

func (c *StorageSealerStruct) UnsealSector(ctx context.Context, sectorID abi.SectorNumber, offset abi.UnpaddedPieceSize, length abi.UnpaddedPieceSize) (io.ReadCloser, error) {
	return c.Internal.UnsealSector(ctx, sectorID, offset, length)
}
