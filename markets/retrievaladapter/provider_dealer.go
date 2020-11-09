package retrievaladapter

import (
	"context"
	"io"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/actors/builtin/paych"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-fil-markets/retrievalmarket"
	"github.com/filecoin-project/go-fil-markets/shared"
	"github.com/filecoin-project/go-state-types/abi"
)

type retrievalProviderNodeDealer struct {
	sealingApi api.Sealer
	full       api.FullNode
}

// NewRetrievalProviderNode returns a new node adapter for a retrieval provider that talks to the
// Lotus Node
func NewRetrievalProviderNodeDealer(sealingApi api.Sealer, full api.FullNode) retrievalmarket.RetrievalProviderNode {
	return &retrievalProviderNodeDealer{sealingApi, full}
}

func (rpn *retrievalProviderNodeDealer) GetMinerWorkerAddress(ctx context.Context, miner address.Address, tok shared.TipSetToken) (address.Address, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return address.Undef, err
	}

	mi, err := rpn.full.StateMinerInfo(ctx, miner, tsk)
	return mi.Worker, err
}

func (rpn *retrievalProviderNodeDealer) UnsealSector(ctx context.Context, sectorID abi.SectorNumber, offset abi.UnpaddedPieceSize, length abi.UnpaddedPieceSize) (io.ReadCloser, error) {
	return rpn.sealingApi.UnsealSector(ctx, sectorID, offset, length)
	/*
		si, err := rpn.miner.GetSectorInfo(sectorID)
		if err != nil {
			return nil, err
		}

		mid, err := address.IDFromAddress(rpn.miner.Address())
		if err != nil {
			return nil, err
		}

		sid := abi.SectorID{
			Miner:  abi.ActorID(mid),
			Number: sectorID,
		}

		r, w := io.Pipe()
		go func() {
			var commD cid.Cid
			if si.CommD != nil {
				commD = *si.CommD
			}
			err := rpn.sealer.ReadPiece(ctx, w, sid, storiface.UnpaddedByteIndex(offset), length, si.TicketValue, commD)
			_ = w.CloseWithError(err)
		}()

		return r, nil
	*/
}

func (rpn *retrievalProviderNodeDealer) SavePaymentVoucher(ctx context.Context, paymentChannel address.Address, voucher *paych.SignedVoucher, proof []byte, expectedAmount abi.TokenAmount, tok shared.TipSetToken) (abi.TokenAmount, error) {
	// TODO: respect the provided TipSetToken (a serialized TipSetKey) when
	// querying the chain
	added, err := rpn.full.PaychVoucherAdd(ctx, paymentChannel, voucher, proof, expectedAmount)
	return added, err
}

func (rpn *retrievalProviderNodeDealer) GetChainHead(ctx context.Context) (shared.TipSetToken, abi.ChainEpoch, error) {
	head, err := rpn.full.ChainHead(ctx)
	if err != nil {
		return nil, 0, err
	}

	return head.Key().Bytes(), head.Height(), nil
}
