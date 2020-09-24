package retrievaladapter

import (
	"context"
	"io"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-fil-markets/retrievalmarket"
	"github.com/filecoin-project/go-fil-markets/shared"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/actors/builtin/paych"
)

type retrievalProviderNodeDealer struct {
	sealingApi api.StorageMiner
	full       api.FullNode
}

// NewRetrievalProviderNode returns a new node adapter for a retrieval provider that talks to the
// Lotus Node
func NewRetrievalProviderNodeDealer(sealingApi api.StorageMiner, full api.FullNode) retrievalmarket.RetrievalProviderNode {
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
