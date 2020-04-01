package mock

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"math/rand"
	"sync"

	commcid "github.com/filecoin-project/go-fil-commcid"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/filecoin-project/specs-storage/storage"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sector-storage"
	"github.com/filecoin-project/sector-storage/ffiwrapper"
)

var log = logging.Logger("sbmock")

type SectorMgr struct {
	sectors      map[abi.SectorID]*sectorState
	sectorSize   abi.SectorSize
	nextSectorID abi.SectorNumber
	rateLimit    chan struct{}
	proofType    abi.RegisteredProof

	lk sync.Mutex
}

type mockVerif struct{}

func NewMockSectorMgr(threads int, ssize abi.SectorSize) *SectorMgr {
	rt, _, err := ffiwrapper.ProofTypeFromSectorSize(ssize)
	if err != nil {
		panic(err)
	}

	return &SectorMgr{
		sectors:      make(map[abi.SectorID]*sectorState),
		sectorSize:   ssize,
		nextSectorID: 5,
		rateLimit:    make(chan struct{}, threads),
		proofType:    rt,
	}
}

const (
	statePacking = iota
	statePreCommit
	stateCommit // nolint
)

type sectorState struct {
	pieces []cid.Cid
	failed bool

	state int

	lk sync.Mutex
}

func (sb *SectorMgr) RateLimit() func() {
	sb.rateLimit <- struct{}{}

	// TODO: probably want to copy over rate limit code
	return func() {
		<-sb.rateLimit
	}
}

func (sb *SectorMgr) NewSector(ctx context.Context, sector abi.SectorID) error {
	return nil
}

func (sb *SectorMgr) AddPiece(ctx context.Context, sectorId abi.SectorID, existingPieces []abi.UnpaddedPieceSize, size abi.UnpaddedPieceSize, r io.Reader) (abi.PieceInfo, error) {
	log.Warn("Add piece: ", sectorId, size, sb.proofType)
	sb.lk.Lock()
	ss, ok := sb.sectors[sectorId]
	if !ok {
		ss = &sectorState{
			state: statePacking,
		}
		sb.sectors[sectorId] = ss
	}
	sb.lk.Unlock()
	ss.lk.Lock()
	defer ss.lk.Unlock()

	c, err := ffiwrapper.GeneratePieceCIDFromFile(sb.proofType, r, size)
	if err != nil {
		return abi.PieceInfo{}, xerrors.Errorf("failed to generate piece cid: %w", err)
	}

	log.Warn("Generated Piece CID: ", c)

	ss.pieces = append(ss.pieces, c)
	return abi.PieceInfo{
		Size:     size.Padded(),
		PieceCID: c,
	}, nil
}

func (sb *SectorMgr) SectorSize() abi.SectorSize {
	return sb.sectorSize
}

func (sb *SectorMgr) AcquireSectorNumber() (abi.SectorNumber, error) {
	sb.lk.Lock()
	defer sb.lk.Unlock()
	id := sb.nextSectorID
	sb.nextSectorID++
	return id, nil
}

func (sb *SectorMgr) SealPreCommit1(ctx context.Context, sid abi.SectorID, ticket abi.SealRandomness, pieces []abi.PieceInfo) (out storage.PreCommit1Out, err error) {
	sb.lk.Lock()
	ss, ok := sb.sectors[sid]
	sb.lk.Unlock()
	if !ok {
		return nil, xerrors.Errorf("no sector with id %d in storage", sid)
	}

	ss.lk.Lock()
	defer ss.lk.Unlock()

	ussize := abi.PaddedPieceSize(sb.sectorSize).Unpadded()

	// TODO: verify pieces in sinfo.pieces match passed in pieces

	var sum abi.UnpaddedPieceSize
	for _, p := range pieces {
		sum += p.Size.Unpadded()
	}

	if sum != ussize {
		return nil, xerrors.Errorf("aggregated piece sizes don't match up: %d != %d", sum, ussize)
	}

	if ss.state != statePacking {
		return nil, xerrors.Errorf("cannot call pre-seal on sector not in 'packing' state")
	}

	opFinishWait(ctx)

	ss.state = statePreCommit

	pis := make([]abi.PieceInfo, len(ss.pieces))
	for i, piece := range ss.pieces {
		pis[i] = abi.PieceInfo{
			Size:     pieces[i].Size,
			PieceCID: piece,
		}
	}

	commd, err := MockVerifier.GenerateDataCommitment(sb.proofType, pis)
	if err != nil {
		return nil, err
	}

	cc, _, err := commcid.CIDToCommitment(commd)
	if err != nil {
		panic(err)
	}

	cc[0] ^= 'd'

	return cc, nil
}

func (sb *SectorMgr) SealPreCommit2(ctx context.Context, sid abi.SectorID, phase1Out storage.PreCommit1Out) (cids storage.SectorCids, err error) {
	db := []byte(string(phase1Out))
	db[0] ^= 'd'

	d := commcid.DataCommitmentV1ToCID(db)

	commr := make([]byte, 32)
	for i := range db {
		commr[32-(i+1)] = db[i]
	}

	commR := commcid.DataCommitmentV1ToCID(commr)

	return storage.SectorCids{
		Unsealed: d,
		Sealed:   commR,
	}, nil
}

func (sb *SectorMgr) SealCommit1(ctx context.Context, sid abi.SectorID, ticket abi.SealRandomness, seed abi.InteractiveSealRandomness, pieces []abi.PieceInfo, cids storage.SectorCids) (output storage.Commit1Out, err error) {
	sb.lk.Lock()
	ss, ok := sb.sectors[sid]
	sb.lk.Unlock()
	if !ok {
		return nil, xerrors.Errorf("no such sector %d", sid)
	}
	ss.lk.Lock()
	defer ss.lk.Unlock()

	if ss.failed {
		return nil, xerrors.Errorf("[mock] cannot commit failed sector %d", sid)
	}

	if ss.state != statePreCommit {
		return nil, xerrors.Errorf("cannot commit sector that has not been precommitted")
	}

	opFinishWait(ctx)

	var out [32]byte
	for i := range out {
		out[i] = cids.Unsealed.Bytes()[i] + cids.Sealed.Bytes()[31-i] - ticket[i]*seed[i] ^ byte(sid.Number&0xff)
	}

	return out[:], nil
}

func (sb *SectorMgr) SealCommit2(ctx context.Context, sid abi.SectorID, phase1Out storage.Commit1Out) (proof storage.Proof, err error) {
	var out [32]byte
	for i := range out {
		out[i] = phase1Out[i] ^ byte(sid.Number&0xff)
	}

	return out[:], nil
}

// Test Instrumentation Methods

func (sb *SectorMgr) FailSector(sid abi.SectorID) error {
	sb.lk.Lock()
	defer sb.lk.Unlock()
	ss, ok := sb.sectors[sid]
	if !ok {
		return fmt.Errorf("no such sector in storage")
	}

	ss.failed = true
	return nil
}

func opFinishWait(ctx context.Context) {
	val, ok := ctx.Value("opfinish").(chan struct{})
	if !ok {
		return
	}
	<-val
}

func AddOpFinish(ctx context.Context) (context.Context, func()) {
	done := make(chan struct{})

	return context.WithValue(ctx, "opfinish", done), func() {
		close(done)
	}
}

func (sb *SectorMgr) GenerateFallbackPoSt(context.Context, abi.ActorID, []abi.SectorInfo, abi.PoStRandomness, []abi.SectorNumber) (storage.FallbackPostOut, error) {
	panic("implement me")
}

func (sb *SectorMgr) ComputeElectionPoSt(ctx context.Context, mid abi.ActorID, sectorInfo []abi.SectorInfo, challengeSeed abi.PoStRandomness, winners []abi.PoStCandidate) ([]abi.PoStProof, error) {
	panic("implement me")
}

func (sb *SectorMgr) GenerateEPostCandidates(ctx context.Context, mid abi.ActorID, sectorInfo []abi.SectorInfo, challengeSeed abi.PoStRandomness, faults []abi.SectorNumber) ([]storage.PoStCandidateWithTicket, error) {
	if len(faults) > 0 {
		panic("todo")
	}

	n := ffiwrapper.ElectionPostChallengeCount(uint64(len(sectorInfo)), uint64(len(faults)))
	if n > uint64(len(sectorInfo)) {
		n = uint64(len(sectorInfo))
	}

	out := make([]storage.PoStCandidateWithTicket, n)

	seed := big.NewInt(0).SetBytes(challengeSeed[:])
	start := seed.Mod(seed, big.NewInt(int64(len(sectorInfo)))).Int64()

	for i := range out {
		out[i] = storage.PoStCandidateWithTicket{
			Candidate: abi.PoStCandidate{
				SectorID: abi.SectorID{
					Number: abi.SectorNumber((int(start) + i) % len(sectorInfo)),
					Miner:  mid,
				},
				PartialTicket: abi.PartialTicket(challengeSeed),
			},
		}
	}

	return out, nil
}

func (sb *SectorMgr) ReadPieceFromSealedSector(ctx context.Context, sectorID abi.SectorID, offset ffiwrapper.UnpaddedByteIndex, size abi.UnpaddedPieceSize, ticket abi.SealRandomness, commD cid.Cid) (io.ReadCloser, error) {
	if len(sb.sectors[sectorID].pieces) > 1 {
		panic("implme")
	}
	return ioutil.NopCloser(io.LimitReader(bytes.NewReader(sb.sectors[sectorID].pieces[0].Bytes()[offset:]), int64(size))), nil
}

func (sb *SectorMgr) StageFakeData(mid abi.ActorID) (abi.SectorID, []abi.PieceInfo, error) {
	usize := abi.PaddedPieceSize(sb.sectorSize).Unpadded()
	sid, err := sb.AcquireSectorNumber()
	if err != nil {
		return abi.SectorID{}, nil, err
	}

	buf := make([]byte, usize)
	rand.Read(buf)

	id := abi.SectorID{
		Miner:  mid,
		Number: sid,
	}

	pi, err := sb.AddPiece(context.TODO(), id, nil, usize, bytes.NewReader(buf))
	if err != nil {
		return abi.SectorID{}, nil, err
	}

	return id, []abi.PieceInfo{pi}, nil
}

func (sb *SectorMgr) FinalizeSector(context.Context, abi.SectorID) error {
	return nil
}

func (m mockVerif) VerifyElectionPost(ctx context.Context, pvi abi.PoStVerifyInfo) (bool, error) {
	panic("implement me")
}

func (m mockVerif) VerifyFallbackPost(ctx context.Context, pvi abi.PoStVerifyInfo) (bool, error) {
	panic("implement me")
}

func (m mockVerif) VerifySeal(svi abi.SealVerifyInfo) (bool, error) {
	if len(svi.OnChain.Proof) != 32 { // Real ones are longer, but this should be fine
		return false, nil
	}

	for i, b := range svi.OnChain.Proof {
		if b != svi.UnsealedCID.Bytes()[i]+svi.OnChain.SealedCID.Bytes()[31-i]-svi.InteractiveRandomness[i]*svi.Randomness[i] {
			return false, nil
		}
	}

	return true, nil
}

func (m mockVerif) GenerateDataCommitment(pt abi.RegisteredProof, pieces []abi.PieceInfo) (cid.Cid, error) {
	return ffiwrapper.GenerateUnsealedCID(pt, pieces)
}

var MockVerifier = mockVerif{}

var _ ffiwrapper.Verifier = MockVerifier
var _ sectorstorage.SectorManager = &SectorMgr{}
