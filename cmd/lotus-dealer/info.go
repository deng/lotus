package main

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	cbor "github.com/ipfs/go-ipld-cbor"

	"github.com/filecoin-project/lotus/api/apibstore"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/actors/adt"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/lotus/lib/blockstore"
	"github.com/filecoin-project/lotus/lib/bufbstore"
)

var infoCmd = &cli.Command{
	Name:   "info",
	Usage:  "Print miner info",
	Action: infoCmdAct,
}

func infoCmdAct(cctx *cli.Context) error {
	color.NoColor = !cctx.Bool("color")

	nodeApi, closer, err := lcli.GetStorageDealerAPI(cctx)
	if err != nil {
		return err
	}
	defer closer()

	api, acloser, err := lcli.GetFullNodeAPI(cctx)
	if err != nil {
		return err
	}
	defer acloser()

	ctx := lcli.ReqContext(cctx)

	maddr, err := getActorAddress(ctx, nodeApi, cctx.String("actor"))
	if err != nil {
		return err
	}

	mact, err := api.StateGetActor(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return err
	}

	tbs := bufbstore.NewTieredBstore(apibstore.NewAPIBlockstore(api), blockstore.NewTemporary())
	mas, err := miner.Load(adt.WrapStore(ctx, cbor.NewCborStore(tbs)), mact)
	if err != nil {
		return err
	}

	fmt.Printf("Miner: %s\n", color.BlueString("%s", maddr))

	// Sector size
	mi, err := api.StateMinerInfo(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return err
	}

	fmt.Printf("PeerId: %s\n", color.BlueString("%s", mi.PeerId))

	// Multiaddrs
	fmt.Printf("Multiaddrs: %d\n", len(mi.Multiaddrs))
	for i, a := range mi.Multiaddrs {
		maddr, err := multiaddr.NewMultiaddrBytes(a)
		if err != nil {
			log.Warnf("parsing multiaddr %d (%x): %s", i, a, err)
			continue
		}
		fmt.Printf("multiaddr: %s\n", color.BlueString("%s", maddr))
	}

	fmt.Printf("Sector Size: %s\n", types.SizeStr(types.NewInt(uint64(mi.SectorSize))))

	pow, err := api.StateMinerPower(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return err
	}

	rpercI := types.BigDiv(types.BigMul(pow.MinerPower.RawBytePower, types.NewInt(1000000)), pow.TotalPower.RawBytePower)
	qpercI := types.BigDiv(types.BigMul(pow.MinerPower.QualityAdjPower, types.NewInt(1000000)), pow.TotalPower.QualityAdjPower)

	fmt.Printf("Byte Power:   %s / %s (%0.4f%%)\n",
		color.BlueString(types.SizeStr(pow.MinerPower.RawBytePower)),
		types.SizeStr(pow.TotalPower.RawBytePower),
		float64(rpercI.Int64())/10000)

	fmt.Printf("Actual Power: %s / %s (%0.4f%%)\n",
		color.GreenString(types.DeciStr(pow.MinerPower.QualityAdjPower)),
		types.DeciStr(pow.TotalPower.QualityAdjPower),
		float64(qpercI.Int64())/10000)

	secCounts, err := api.StateMinerSectorCount(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return err
	}

	proving := secCounts.Active + secCounts.Faulty
	nfaults := secCounts.Faulty
	fmt.Printf("\tCommitted: %s\n", types.SizeStr(types.BigMul(types.NewInt(secCounts.Live), types.NewInt(uint64(mi.SectorSize)))))
	if nfaults == 0 {
		fmt.Printf("\tProving: %s\n", types.SizeStr(types.BigMul(types.NewInt(proving), types.NewInt(uint64(mi.SectorSize)))))
	} else {
		var faultyPercentage float64
		if secCounts.Live != 0 {
			faultyPercentage = float64(10000*nfaults/secCounts.Live) / 100.
		}
		fmt.Printf("\tProving: %s (%s Faulty, %.2f%%)\n",
			types.SizeStr(types.BigMul(types.NewInt(proving), types.NewInt(uint64(mi.SectorSize)))),
			types.SizeStr(types.BigMul(types.NewInt(nfaults), types.NewInt(uint64(mi.SectorSize)))),
			faultyPercentage)
	}

	if !pow.HasMinPower {
		fmt.Print("Below minimum power threshold, no blocks will be won")
	} else {
		expWinChance := float64(types.BigMul(qpercI, types.NewInt(build.BlocksPerEpoch)).Int64()) / 1000000
		if expWinChance > 0 {
			if expWinChance > 1 {
				expWinChance = 1
			}
			winRate := time.Duration(float64(time.Second*time.Duration(build.BlockDelaySecs)) / expWinChance)
			winPerDay := float64(time.Hour*24) / float64(winRate)

			fmt.Print("Expected block win rate: ")
			color.Blue("%.4f/day (every %s)", winPerDay, winRate.Truncate(time.Second))
		}
	}

	fmt.Println()

	// NOTE: there's no need to unlock anything here. Funds only
	// vest on deadline boundaries, and they're unlocked by cron.
	lockedFunds, err := mas.LockedFunds()
	if err != nil {
		return xerrors.Errorf("getting locked funds: %w", err)
	}
	availBalance, err := mas.AvailableBalance(mact.Balance)
	if err != nil {
		return xerrors.Errorf("getting available balance: %w", err)
	}
	fmt.Printf("Miner Balance: %s\n", color.YellowString("%s", types.FIL(mact.Balance)))
	fmt.Printf("\tPreCommit:   %s\n", types.FIL(lockedFunds.PreCommitDeposits))
	fmt.Printf("\tPledge:      %s\n", types.FIL(lockedFunds.InitialPledgeRequirement))
	fmt.Printf("\tVesting:     %s\n", types.FIL(lockedFunds.VestingFunds))
	color.Green("\tAvailable:   %s", types.FIL(availBalance))
	wb, err := api.WalletBalance(ctx, mi.Worker)
	if err != nil {
		return xerrors.Errorf("getting worker balance: %w", err)
	}
	color.Cyan("Worker Balance: %s", types.FIL(wb))

	mb, err := api.StateMarketBalance(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return xerrors.Errorf("getting market balance: %w", err)
	}
	fmt.Printf("Market (Escrow):  %s\n", types.FIL(mb.Escrow))
	fmt.Printf("Market (Locked):  %s\n", types.FIL(mb.Locked))

	return nil
}
