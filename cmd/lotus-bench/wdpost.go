package main

import (
	"github.com/filecoin-project/go-address"
	saproof "github.com/filecoin-project/specs-actors/actors/runtime/proof"
	"github.com/urfave/cli/v2"
)

//var wdpostCmd = &cli.Command{
//	Name:  "wdpost",
//	Usage: "Benchmark a wdpost computation",
//	Flags: []cli.Flag{
//		&cli.StringFlag{
//			Name:  "storage-dir",
//			Value: "~/.lotus-bench",
//			Usage: "storage.json Path to the storage directory that will store sectors long term",
//		},
//		&cli.StringFlag{
//			Name:  "sector-size",
//			Value: "512MiB",
//			Usage: "size of the sectors in bytes, i.e. 32GiB",
//		},
//		&cli.BoolFlag{
//			Name:  "no-gpu",
//			Usage: "disable gpu usage for the benchmark run",
//		},
//		&cli.Uint64Flag{
//			Name:  "deadline-num",
//			Value: 0,
//		},
//		&cli.StringFlag{
//			Name:  "miner-addr",
//			Usage: "pass miner address (only necessary if using existing sectorbuilder)",
//			Value: "t01000",
//		},
//	},
//	Action: func(c *cli.Context) error {
//		if c.Bool("no-gpu") {
//			err := os.Setenv("BELLMAN_NO_GPU", "1")
//			if err != nil {
//				return xerrors.Errorf("setting no-gpu flag: %w", err)
//			}
//		}
//		// sector size
//		sectorSizeInt, err := units.RAMInBytes(c.String("sector-size"))
//		if err != nil {
//			return err
//		}
//		sectorSize := abi.SectorSize(sectorSizeInt)
//		spt, err := ffiwrapper.SealProofTypeFromSectorSize(sectorSize)
//		if err != nil {
//			return err
//		}
//		// 获取时空证明
//		if err := paramfetch.GetParams(lcli.ReqContext(c), build.ParametersJSON(), uint64(sectorSize)); err != nil {
//			return xerrors.Errorf("getting params: %w", err)
//		}
//		// miner address
//		maddr, err := address.NewFromString(c.String("miner-addr"))
//		if err != nil {
//			return err
//		}
//		amid, err := address.IDFromAddress(maddr)
//		if err != nil {
//			return err
//		}
//		mid := abi.ActorID(amid)
//		//创建时空证明提供者
//		cfg := &ffiwrapper.Config{
//			SealProofType: spt,
//		}
//		sbfs := &basicfs.Provider{
//			Root: c.String("storage-dir"),
//		}
//		sb, err := ffiwrapper.New(sbfs, cfg)
//		if err != nil {
//			return err
//		}
//
//		sinfos, err := sectorsForProof(c, maddr)
//		//sinfos, err := getProves(c)
//		if len(sinfos) == 0 {
//			// nothing to prove..
//			return xerrors.Errorf(" nothing to prove..")
//		}
//
//		var challenge [32]byte
//		rand.Read(challenge[:])
//		start := time.Now()
//		log.Info("computing window PoSt start sectors ", len(sinfos))
//		wproof1, _, err := sb.GenerateWindowPoSt(context.TODO(), mid, sinfos, challenge[:])
//		if err != nil {
//			return err
//		}
//
//		wpvi1 := saproof.WindowPoStVerifyInfo{
//			Randomness:        challenge[:],
//			Proofs:            wproof1,
//			ChallengedSectors: sinfos,
//			Prover:            mid,
//		}
//		ok, err := ffiwrapper.ProofVerifier.VerifyWindowPoSt(context.TODO(), wpvi1)
//		if err != nil {
//			return err
//		}
//		if !ok {
//			log.Error("post verification failed")
//		}
//
//		elapsed := time.Since(start)
//
//		log.Infow("computing window PoSt", "elapsed", elapsed)
//		return nil
//	},
//}

func sectorsForProof(c *cli.Context, actor address.Address) ([]saproof.SectorInfo, error) {
	//获取待认证的扇区
	//nodeApi, ncloser, err := lcli.GetFullNodeAPI(c)
	//if err != nil {
	//	return nil, err
	//}
	//defer ncloser()
	//
	//ctx := lcli.DaemonContext(c)
	//partitions, err := nodeApi.StateMinerPartitions(ctx, actor, c.Uint64("deadline-num"), types.EmptyTSK)
	//if err != nil {
	//	return nil, xerrors.Errorf("getting partitions: %w", err)
	//}

	proofSectors := make([]saproof.SectorInfo, 0)
	//for _, partition := range partitions {
	//
	//	//toProve, err := partition.ActiveSectors()
	//	//if err != nil {
	//	//	return nil,xerrors.Errorf("getting active sectors: %w", err)
	//	//}
	//	//toProve, err = bitfield.MergeBitFields(toProve, partition.Recoveries)
	//	//if err != nil {
	//	//	return  nil,xerrors.Errorf("adding recoveries to set of sectors to prove: %w", err)
	//	//}
	//	log.Errorf("toProve ActiveSectors : %v", partition.Sectors)
	//	sset, err := nodeApi.StateMinerSectors(ctx, actor, &partition.Sectors, false, types.EmptyTSK)
	//	if err != nil {
	//		return nil, err
	//	}
	//	if len(sset) == 0 {
	//		continue
	//	}
	//	log.Errorf("toProve StateMinerSectors sectors : %d", len(sset))
	//	sectorByID := make(map[uint64]saproof.SectorInfo, len(sset))
	//	for _, sector := range sset {
	//		switch int64(sector.ID) {
	//		case 110, 150, 194, 201, 368, 390, 666, 737, 754, 1089, 1172, 1277, 1346, 1398, 1639, 1680, 1684, 1760, 1773, 1793, 1795, 1834, 1874, 1945, 246, 2109:
	//			continue
	//		}
	//		sectorByID[uint64(sector.ID)] = saproof.SectorInfo{
	//			SectorNumber: sector.ID,
	//			SealedCID:    sector.Info.SealedCID,
	//			SealProof:    sector.Info.SealProof,
	//		}
	//	}
	//	for _, sector := range sectorByID {
	//		proofSectors = append(proofSectors, sector)
	//	}
	//}
	if len(proofSectors) == 0 {
		log.Error("don't have any sectors")
		return nil, nil
	}
	return proofSectors, nil
}

//func getProves(c *cli.Context) ([]saproof.SectorInfo, error) {
//	minerApi, closer, err := lcli.GetStorageMinerAPI(c,
//		jsonrpc.WithNoReconnect(),
//		jsonrpc.WithTimeout(30*time.Second))
//	if err != nil {
//		return nil, err
//	}
//	defer closer()
//	ctx := lcli.ReqContext(c)
//	ctx, cancel := context.WithCancel(ctx)
//	defer cancel()
//
//	sectors, err := minerApi.SectorsList(ctx)
//	if err != nil {
//		return nil, err
//	}
//	res := make([]saproof.SectorInfo, 0)
//
//	for _, sc := range sectors {
//		st, err := minerApi.SectorsStatus(ctx, sc, false)
//		if err != nil {
//			return nil, err
//		}
//		if st.State == "Proving" {
//			res = append(res, saproof.SectorInfo{
//				SectorNumber: st.SectorID,
//				SealedCID:    *st.CommR,
//				SealProof:    st.SealProof,
//			})
//		}
//	}
//	if len(res) == 0 {
//		return nil, fmt.Errorf("don't have proving section")
//	}
//
//	return res, nil
//}
