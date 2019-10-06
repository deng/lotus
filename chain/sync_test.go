package chain_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-lotus/api"
	"github.com/filecoin-project/go-lotus/chain/address"
	"github.com/filecoin-project/go-lotus/chain/gen"
	"github.com/filecoin-project/go-lotus/chain/store"
	"github.com/filecoin-project/go-lotus/chain/types"
	"github.com/filecoin-project/go-lotus/node"
	"github.com/filecoin-project/go-lotus/node/impl"
	"github.com/filecoin-project/go-lotus/node/modules"
	"github.com/filecoin-project/go-lotus/node/repo"
)

const source = 0

func (tu *syncTestUtil) repoWithChain(t testing.TB, h int) (repo.Repo, []byte, []*store.FullTipSet) {
	blks := make([]*store.FullTipSet, h)

	for i := 0; i < h; i++ {
		mts, err := tu.g.NextTipSet()
		require.NoError(t, err)

		fmt.Println("REPO WITH CHAIN NEXT TIPSET: ", mts.TipSet.TipSet().Height())

		blks[i] = mts.TipSet

		ts := mts.TipSet.TipSet()
		fmt.Printf("tipset at H:%d: %s\n", ts.Height(), ts.Cids())
	}

	r, err := tu.g.YieldRepo()
	require.NoError(t, err)

	genb, err := tu.g.GenesisCar()
	require.NoError(t, err)

	return r, genb, blks
}

type syncTestUtil struct {
	t testing.TB

	ctx    context.Context
	cancel func()

	mn mocknet.Mocknet

	g *gen.ChainGen

	genesis []byte
	blocks  []*store.FullTipSet

	nds []api.FullNode
}

func prepSyncTest(t testing.TB, h int) *syncTestUtil {
	logging.SetLogLevel("*", "INFO")

	g, err := gen.NewGenerator()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	tu := &syncTestUtil{
		t:      t,
		ctx:    ctx,
		cancel: cancel,

		mn: mocknet.New(ctx),
		g:  g,
	}

	tu.addSourceNode(h)
	//tu.checkHeight("source", source, h)

	// separate logs
	fmt.Println("\x1b[31m///////////////////////////////////////////////////\x1b[39b")

	return tu
}

func (tu *syncTestUtil) Shutdown() {
	tu.cancel()
}

func (tu *syncTestUtil) pushFtsAndWait(to int, fts *store.FullTipSet) {
	// TODO: would be great if we could pass a whole tipset here...
	for _, fb := range fts.Blocks {
		var b types.BlockMsg

		// -1 to match block.Height
		b.Header = fb.Header
		for _, msg := range fb.SecpkMessages {
			c, err := tu.nds[to].(*impl.FullNodeAPI).ChainAPI.Chain.PutMessage(msg)
			require.NoError(tu.t, err)

			b.SecpkMessages = append(b.SecpkMessages, c)
		}

		for _, msg := range fb.BlsMessages {
			c, err := tu.nds[to].(*impl.FullNodeAPI).ChainAPI.Chain.PutMessage(msg)
			require.NoError(tu.t, err)

			b.BlsMessages = append(b.BlsMessages, c)
		}

		require.NoError(tu.t, tu.nds[to].ChainSubmitBlock(tu.ctx, &b))

	}

	start := time.Now()
	h, err := tu.nds[to].ChainHead(tu.ctx)
	require.NoError(tu.t, err)
	for !h.Equals(fts.TipSet()) {
		time.Sleep(time.Millisecond * 50)
		h, err = tu.nds[to].ChainHead(tu.ctx)
		require.NoError(tu.t, err)

		if time.Since(start) > time.Second*10 {
			tu.t.Fatal("took too long waiting for block to be accepted")
		}
	}
}

func (tu *syncTestUtil) mineOnBlock(blk *store.FullTipSet, src int, miners []int) *store.FullTipSet {
	if miners == nil {
		for i := range tu.g.Miners {
			miners = append(miners, i)
		}
	}

	var maddrs []address.Address
	for _, i := range miners {
		maddrs = append(maddrs, tu.g.Miners[i])
	}

	fmt.Println("Miner mining block: ", maddrs)

	mts, err := tu.g.NextTipSetFromMiners(blk.TipSet(), maddrs)
	require.NoError(tu.t, err)

	tu.pushFtsAndWait(src, mts.TipSet)

	return mts.TipSet
}

func (tu *syncTestUtil) mineNewBlock(src int, miners []int) {
	mts := tu.mineOnBlock(tu.g.CurTipset, src, miners)
	tu.g.CurTipset = mts
}

func fblkToBlkMsg(fb *types.FullBlock) *types.BlockMsg {
	out := &types.BlockMsg{
		Header: fb.Header,
	}

	for _, msg := range fb.BlsMessages {
		out.BlsMessages = append(out.BlsMessages, msg.Cid())
	}
	for _, msg := range fb.SecpkMessages {
		out.SecpkMessages = append(out.SecpkMessages, msg.Cid())
	}
	return out
}

func (tu *syncTestUtil) addSourceNode(gen int) {
	if tu.genesis != nil {
		tu.t.Fatal("source node already exists")
	}

	sourceRepo, genesis, blocks := tu.repoWithChain(tu.t, gen)
	var out api.FullNode

	// TODO: Don't ignore stop
	_, err := node.New(tu.ctx,
		node.FullAPI(&out),
		node.Online(),
		node.Repo(sourceRepo),
		node.MockHost(tu.mn),

		node.Override(new(modules.Genesis), modules.LoadGenesis(genesis)),
	)
	require.NoError(tu.t, err)

	lastTs := blocks[len(blocks)-1].Blocks
	for _, lastB := range lastTs {
		err = out.(*impl.FullNodeAPI).ChainAPI.Chain.AddBlock(lastB.Header)
		require.NoError(tu.t, err)
	}

	tu.genesis = genesis
	tu.blocks = blocks
	tu.nds = append(tu.nds, out) // always at 0
}

func (tu *syncTestUtil) addClientNode() int {
	if tu.genesis == nil {
		tu.t.Fatal("source doesn't exists")
	}

	var out api.FullNode

	// TODO: Don't ignore stop
	_, err := node.New(tu.ctx,
		node.FullAPI(&out),
		node.Online(),
		node.Repo(repo.NewMemory(nil)),
		node.MockHost(tu.mn),

		node.Override(new(modules.Genesis), modules.LoadGenesis(tu.genesis)),
	)
	require.NoError(tu.t, err)

	tu.nds = append(tu.nds, out)
	return len(tu.nds) - 1
}

func (tu *syncTestUtil) pid(n int) peer.ID {
	nal, err := tu.nds[n].NetAddrsListen(tu.ctx)
	require.NoError(tu.t, err)

	return nal.ID
}

func (tu *syncTestUtil) connect(from, to int) {
	toPI, err := tu.nds[to].NetAddrsListen(tu.ctx)
	require.NoError(tu.t, err)

	err = tu.nds[from].NetConnect(tu.ctx, toPI)
	require.NoError(tu.t, err)
}

func (tu *syncTestUtil) disconnect(from, to int) {
	toPI, err := tu.nds[to].NetAddrsListen(tu.ctx)
	require.NoError(tu.t, err)

	err = tu.nds[from].NetDisconnect(tu.ctx, toPI.ID)
	require.NoError(tu.t, err)
}

func (tu *syncTestUtil) checkHeight(name string, n int, h int) {
	b, err := tu.nds[n].ChainHead(tu.ctx)
	require.NoError(tu.t, err)

	require.Equal(tu.t, uint64(h), b.Height())
	fmt.Printf("%s H: %d\n", name, b.Height())
}

func (tu *syncTestUtil) compareSourceState(with int) {
	sourceHead, err := tu.nds[source].ChainHead(tu.ctx)
	require.NoError(tu.t, err)

	targetHead, err := tu.nds[with].ChainHead(tu.ctx)
	require.NoError(tu.t, err)

	if !sourceHead.Equals(targetHead) {
		fmt.Println("different chains: ", sourceHead.Height(), targetHead.Height())
		tu.t.Fatalf("nodes were not synced correctly: %s != %s", sourceHead.Cids(), targetHead.Cids())
	}

	sourceAccounts, err := tu.nds[source].WalletList(tu.ctx)
	require.NoError(tu.t, err)

	for _, addr := range sourceAccounts {
		sourceBalance, err := tu.nds[source].WalletBalance(tu.ctx, addr)
		require.NoError(tu.t, err)
		fmt.Printf("Source state check for %s, expect %s\n", addr, sourceBalance)

		actBalance, err := tu.nds[with].WalletBalance(tu.ctx, addr)
		require.NoError(tu.t, err)

		require.Equal(tu.t, sourceBalance, actBalance)
		fmt.Printf("Source state check <OK> for %s\n", addr)
	}
}

func (tu *syncTestUtil) waitUntilSync(from, to int) {
	target, err := tu.nds[from].ChainHead(tu.ctx)
	if err != nil {
		tu.t.Fatal(err)
	}

	tu.waitUntilSyncTarget(to, target)
}

func (tu *syncTestUtil) waitUntilSyncTarget(to int, target *types.TipSet) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hc, err := tu.nds[to].ChainNotify(ctx)
	if err != nil {
		tu.t.Fatal(err)
	}

	// TODO: some sort of timeout?
	for n := range hc {
		for _, c := range n {
			if c.Val.Equals(target) {
				return
			}
		}
	}
}

func TestSyncSimple(t *testing.T) {
	H := 50
	tu := prepSyncTest(t, H)

	client := tu.addClientNode()
	//tu.checkHeight("client", client, 0)

	require.NoError(t, tu.mn.LinkAll())
	tu.connect(1, 0)
	tu.waitUntilSync(0, client)

	//tu.checkHeight("client", client, H)

	tu.compareSourceState(client)
}

func TestSyncMining(t *testing.T) {
	H := 50
	tu := prepSyncTest(t, H)

	client := tu.addClientNode()
	//tu.checkHeight("client", client, 0)

	require.NoError(t, tu.mn.LinkAll())
	tu.connect(client, 0)
	tu.waitUntilSync(0, client)

	//tu.checkHeight("client", client, H)

	tu.compareSourceState(client)

	for i := 0; i < 5; i++ {
		tu.mineNewBlock(0, nil)
		tu.waitUntilSync(0, client)
		tu.compareSourceState(client)
	}
}

func (tu *syncTestUtil) loadChainToNode(to int) {
	// utility to simulate incoming blocks without miner process
	// TODO: should call syncer directly, this won't work correctly in all cases

	for i := 0; i < len(tu.blocks); i++ {
		tu.pushFtsAndWait(to, tu.blocks[i])
	}
}

func TestSyncFork(t *testing.T) {
	H := 10
	tu := prepSyncTest(t, H)

	p1 := tu.addClientNode()
	p2 := tu.addClientNode()

	fmt.Println("GENESIS: ", tu.g.Genesis().Cid())
	tu.loadChainToNode(p1)
	tu.loadChainToNode(p2)

	phead := func() {
		h1, err := tu.nds[1].ChainHead(tu.ctx)
		require.NoError(tu.t, err)

		h2, err := tu.nds[2].ChainHead(tu.ctx)
		require.NoError(tu.t, err)

		fmt.Println("Node 1: ", h1.Cids(), h1.Parents(), h1.Height())
		fmt.Println("Node 2: ", h2.Cids(), h1.Parents(), h2.Height())
		//time.Sleep(time.Second * 2)
		fmt.Println()
		fmt.Println()
		fmt.Println()
		fmt.Println()
	}

	phead()

	base := tu.g.CurTipset
	fmt.Println("Mining base: ", base.TipSet().Cids(), base.TipSet().Height())

	// The two nodes fork at this point into 'a' and 'b'
	a1 := tu.mineOnBlock(base, p1, []int{0})
	a := tu.mineOnBlock(a1, p1, []int{0})
	a = tu.mineOnBlock(a, p1, []int{0})

	tu.g.ResyncBankerNonce(a1.TipSet())
	// chain B will now be heaviest
	b := tu.mineOnBlock(base, p2, []int{1})
	b = tu.mineOnBlock(b, p2, []int{1})
	b = tu.mineOnBlock(b, p2, []int{1})
	b = tu.mineOnBlock(b, p2, []int{1})

	fmt.Println("A: ", a.Cids(), a.TipSet().Height())
	fmt.Println("B: ", b.Cids(), b.TipSet().Height())

	// Now for the fun part!!

	require.NoError(t, tu.mn.LinkAll())
	tu.connect(p1, p2)
	tu.waitUntilSyncTarget(p1, b.TipSet())
	tu.waitUntilSyncTarget(p2, b.TipSet())

	phead()
}

func BenchmarkSyncBasic(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runSyncBenchLength(b, 100)
	}
}

func runSyncBenchLength(b *testing.B, l int) {
	tu := prepSyncTest(b, l)

	client := tu.addClientNode()
	tu.checkHeight("client", client, 0)

	b.ResetTimer()

	require.NoError(b, tu.mn.LinkAll())
	tu.connect(1, 0)

	tu.waitUntilSync(0, client)
}
