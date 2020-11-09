package api

import (
	"context"
	"io"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-fil-markets/piecestore"
	"github.com/filecoin-project/go-fil-markets/shared"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/extern/sector-storage/fsutil"
	"github.com/filecoin-project/lotus/extern/sector-storage/stores"
	"github.com/filecoin-project/lotus/extern/sector-storage/storiface"
	sealing "github.com/filecoin-project/lotus/extern/storage-sealing"
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
)

// StorageSealer is a low-level interface to the Filecoin network storage miner node
type Sealer interface {
	ActorAddress(context.Context) (address.Address, error)

	ActorSectorSize(context.Context, address.Address) (abi.SectorSize, error)

	// Temp api for testing
	PledgeSector(context.Context) error

	// Get the status of a given sector by ID
	SectorsStatus(ctx context.Context, sid abi.SectorNumber, showOnChainInfo bool) (SectorInfo, error)

	// List all staged sectors
	SectorsList(context.Context) ([]abi.SectorNumber, error)

	SectorsRefs(context.Context) (map[string][]SealedRef, error)

	// SectorStartSealing can be called on sectors in Empty or WaitDeals states
	// to trigger sealing early
	SectorStartSealing(context.Context, abi.SectorNumber) error
	// SectorSetSealDelay sets the time that a newly-created sector
	// waits for more deals before it starts sealing
	SectorSetSealDelay(context.Context, time.Duration) error
	// SectorGetSealDelay gets the time that a newly-created sector
	// waits for more deals before it starts sealing
	SectorGetSealDelay(context.Context) (time.Duration, error)
	// SectorSetExpectedSealDuration sets the expected time for a sector to seal
	SectorSetExpectedSealDuration(context.Context, time.Duration) error
	// SectorGetExpectedSealDuration gets the expected time for a sector to seal
	SectorGetExpectedSealDuration(context.Context) (time.Duration, error)
	SectorsUpdate(context.Context, abi.SectorNumber, SectorState) error
	SectorRemove(context.Context, abi.SectorNumber) error
	SectorMarkForUpgrade(ctx context.Context, id abi.SectorNumber) error

	StorageList(ctx context.Context) (map[stores.ID][]stores.Decl, error)
	StorageLocal(ctx context.Context) (map[stores.ID]string, error)
	StorageStat(ctx context.Context, id stores.ID) (fsutil.FsStat, error)

	// WorkerConnect tells the node to connect to workers RPC
	WorkerConnect(context.Context, string) error
	WorkerStats(context.Context) (map[uuid.UUID]storiface.WorkerStats, error)
	WorkerJobs(context.Context) (map[uuid.UUID][]storiface.WorkerJob, error)

	// SealingSchedDiag dumps internal sealing scheduler state
	SealingSchedDiag(context.Context, bool) (interface{}, error)

	storiface.WorkerReturn
	stores.SectorIndex

	StorageAddLocal(ctx context.Context, path string) error

	PiecesListPieces(ctx context.Context) ([]cid.Cid, error)
	PiecesListCidInfos(ctx context.Context) ([]cid.Cid, error)
	PiecesGetPieceInfo(ctx context.Context, pieceCid cid.Cid) (*piecestore.PieceInfo, error)
	PiecesGetCIDInfo(ctx context.Context, payloadCid cid.Cid) (*piecestore.CIDInfo, error)

	AddPieceOnDealComplete(ctx context.Context, dealerPath string, d sealing.DealInfo) (*storagemarket.PackingResult, error)
	LocatePieceForDealWithinSector(ctx context.Context, dealID abi.DealID, encodedTs shared.TipSetToken) (*LocatePieceResult, error)
	UnsealSector(ctx context.Context, sectorID abi.SectorNumber, offset abi.UnpaddedPieceSize, length abi.UnpaddedPieceSize) (io.ReadCloser, error)
}

type LocatePieceResult struct {
	SectorID abi.SectorNumber
	Offset   abi.PaddedPieceSize
	Length   abi.PaddedPieceSize
}

type StorageSealer interface {
	Common
	Sealer
}
