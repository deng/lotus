package testing

import (
	"context"
	"io"
	"os"

	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-car"
	"github.com/ipfs/go-cid"
	offline "github.com/ipfs/go-ipfs-exchange-offline"
	logging "github.com/ipfs/go-log"
	"github.com/ipfs/go-merkledag"

	"github.com/filecoin-project/go-lotus/chain/address"
	"github.com/filecoin-project/go-lotus/chain/gen"
	"github.com/filecoin-project/go-lotus/chain/types"
	"github.com/filecoin-project/go-lotus/chain/wallet"
	"github.com/filecoin-project/go-lotus/node/modules"
	"github.com/filecoin-project/go-lotus/node/modules/dtypes"
)

var glog = logging.Logger("genesis")

func MakeGenesisMem(out io.Writer) func(bs dtypes.ChainBlockstore, w *wallet.Wallet) modules.Genesis {
	return func(bs dtypes.ChainBlockstore, w *wallet.Wallet) modules.Genesis {
		return func() (*types.BlockHeader, error) {
			glog.Warn("Generating new random genesis block, note that this SHOULD NOT happen unless you are setting up new network")
			// TODO: make an address allocation
			b, err := gen.MakeGenesisBlock(bs, nil)
			if err != nil {
				return nil, err
			}
			offl := offline.Exchange(bs)
			blkserv := blockservice.New(bs, offl)
			dserv := merkledag.NewDAGService(blkserv)

			if err := car.WriteCar(context.TODO(), dserv, []cid.Cid{b.Genesis.Cid()}, out); err != nil {
				return nil, err
			}

			return b.Genesis, nil
		}
	}
}

func MakeGenesis(outFile string) func(bs dtypes.ChainBlockstore, w *wallet.Wallet) modules.Genesis {
	return func(bs dtypes.ChainBlockstore, w *wallet.Wallet) modules.Genesis {
		return func() (*types.BlockHeader, error) {
			glog.Warn("Generating new random genesis block, note that this SHOULD NOT happen unless you are setting up new network")
			minerAddr, err := w.GenerateKey(types.KTSecp256k1)
			if err != nil {
				return nil, err
			}

			addrs := map[address.Address]types.BigInt{
				minerAddr: types.NewInt(50000000),
			}

			b, err := gen.MakeGenesisBlock(bs, addrs)
			if err != nil {
				return nil, err
			}

			f, err := os.OpenFile(outFile, os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return nil, err
			}

			offl := offline.Exchange(bs)
			blkserv := blockservice.New(bs, offl)
			dserv := merkledag.NewDAGService(blkserv)

			if err := car.WriteCar(context.TODO(), dserv, []cid.Cid{b.Genesis.Cid()}, f); err != nil {
				return nil, err
			}

			glog.Warnf("WRITING GENESIS FILE AT %s", f.Name())

			if err := f.Close(); err != nil {
				return nil, err
			}

			return b.Genesis, nil
		}
	}
}
