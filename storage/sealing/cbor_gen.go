// Code generated by github.com/whyrusleeping/cbor-gen. DO NOT EDIT.

package sealing

import (
	"fmt"
	"io"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/specs-actors/actors/abi"
	cbg "github.com/whyrusleeping/cbor-gen"
	xerrors "golang.org/x/xerrors"
)

var _ = xerrors.Errorf

func (t *Piece) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write([]byte{163}); err != nil {
		return err
	}

	// t.DealID (abi.DealID) (uint64)
	if len("DealID") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"DealID\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("DealID")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("DealID")); err != nil {
		return err
	}

	if t.DealID == nil {
		if _, err := w.Write(cbg.CborNull); err != nil {
			return err
		}
	} else {
		if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajUnsignedInt, uint64(*t.DealID))); err != nil {
			return err
		}
	}

	// t.Size (abi.UnpaddedPieceSize) (uint64)
	if len("Size") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"Size\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("Size")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("Size")); err != nil {
		return err
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajUnsignedInt, uint64(t.Size))); err != nil {
		return err
	}

	// t.CommP (cid.Cid) (struct)
	if len("CommP") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"CommP\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("CommP")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("CommP")); err != nil {
		return err
	}

	if err := cbg.WriteCid(w, t.CommP); err != nil {
		return xerrors.Errorf("failed to write cid field t.CommP: %w", err)
	}

	return nil
}

func (t *Piece) UnmarshalCBOR(r io.Reader) error {
	br := cbg.GetPeeker(r)

	maj, extra, err := cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajMap {
		return fmt.Errorf("cbor input should be of type map")
	}

	if extra > cbg.MaxLength {
		return fmt.Errorf("Piece: map struct too large (%d)", extra)
	}

	var name string
	n := extra

	for i := uint64(0); i < n; i++ {

		{
			sval, err := cbg.ReadString(br)
			if err != nil {
				return err
			}

			name = string(sval)
		}

		switch name {
		// t.DealID (abi.DealID) (uint64)
		case "DealID":

			{

				pb, err := br.PeekByte()
				if err != nil {
					return err
				}
				if pb == cbg.CborNull[0] {
					var nbuf [1]byte
					if _, err := br.Read(nbuf[:]); err != nil {
						return err
					}
				} else {
					maj, extra, err = cbg.CborReadHeader(br)
					if err != nil {
						return err
					}
					if maj != cbg.MajUnsignedInt {
						return fmt.Errorf("wrong type for uint64 field")
					}
					typed := abi.DealID(extra)
					t.DealID = &typed
				}

			}
			// t.Size (abi.UnpaddedPieceSize) (uint64)
		case "Size":

			{

				maj, extra, err = cbg.CborReadHeader(br)
				if err != nil {
					return err
				}
				if maj != cbg.MajUnsignedInt {
					return fmt.Errorf("wrong type for uint64 field")
				}
				t.Size = abi.UnpaddedPieceSize(extra)

			}
			// t.CommP (cid.Cid) (struct)
		case "CommP":

			{

				c, err := cbg.ReadCid(br)
				if err != nil {
					return xerrors.Errorf("failed to read cid field t.CommP: %w", err)
				}

				t.CommP = c

			}

		default:
			return fmt.Errorf("unknown struct field %d: '%s'", i, name)
		}
	}

	return nil
}
func (t *SectorInfo) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write([]byte{177}); err != nil {
		return err
	}

	// t.State (api.SectorState) (string)
	if len("State") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"State\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("State")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("State")); err != nil {
		return err
	}

	if len(t.State) > cbg.MaxLength {
		return xerrors.Errorf("Value in field t.State was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len(t.State)))); err != nil {
		return err
	}
	if _, err := w.Write([]byte(t.State)); err != nil {
		return err
	}

	// t.SectorID (abi.SectorNumber) (uint64)
	if len("SectorID") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"SectorID\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("SectorID")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("SectorID")); err != nil {
		return err
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajUnsignedInt, uint64(t.SectorID))); err != nil {
		return err
	}

	// t.Nonce (uint64) (uint64)
	if len("Nonce") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"Nonce\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("Nonce")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("Nonce")); err != nil {
		return err
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajUnsignedInt, uint64(t.Nonce))); err != nil {
		return err
	}

	// t.SectorType (abi.RegisteredProof) (int64)
	if len("SectorType") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"SectorType\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("SectorType")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("SectorType")); err != nil {
		return err
	}

	if t.SectorType >= 0 {
		if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajUnsignedInt, uint64(t.SectorType))); err != nil {
			return err
		}
	} else {
		if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajNegativeInt, uint64(-t.SectorType)-1)); err != nil {
			return err
		}
	}

	// t.Pieces ([]sealing.Piece) (slice)
	if len("Pieces") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"Pieces\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("Pieces")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("Pieces")); err != nil {
		return err
	}

	if len(t.Pieces) > cbg.MaxLength {
		return xerrors.Errorf("Slice value in field t.Pieces was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajArray, uint64(len(t.Pieces)))); err != nil {
		return err
	}
	for _, v := range t.Pieces {
		if err := v.MarshalCBOR(w); err != nil {
			return err
		}
	}

	// t.Ticket (api.SealTicket) (struct)
	if len("Ticket") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"Ticket\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("Ticket")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("Ticket")); err != nil {
		return err
	}

	if err := t.Ticket.MarshalCBOR(w); err != nil {
		return err
	}

	// t.PreCommit1Out (storage.PreCommit1Out) (slice)
	if len("PreCommit1Out") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"PreCommit1Out\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("PreCommit1Out")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("PreCommit1Out")); err != nil {
		return err
	}

	if len(t.PreCommit1Out) > cbg.ByteArrayMaxLen {
		return xerrors.Errorf("Byte array in field t.PreCommit1Out was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajByteString, uint64(len(t.PreCommit1Out)))); err != nil {
		return err
	}
	if _, err := w.Write(t.PreCommit1Out); err != nil {
		return err
	}

	// t.CommD (cid.Cid) (struct)
	if len("CommD") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"CommD\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("CommD")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("CommD")); err != nil {
		return err
	}

	if t.CommD == nil {
		if _, err := w.Write(cbg.CborNull); err != nil {
			return err
		}
	} else {
		if err := cbg.WriteCid(w, *t.CommD); err != nil {
			return xerrors.Errorf("failed to write cid field t.CommD: %w", err)
		}
	}

	// t.CommR (cid.Cid) (struct)
	if len("CommR") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"CommR\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("CommR")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("CommR")); err != nil {
		return err
	}

	if t.CommR == nil {
		if _, err := w.Write(cbg.CborNull); err != nil {
			return err
		}
	} else {
		if err := cbg.WriteCid(w, *t.CommR); err != nil {
			return xerrors.Errorf("failed to write cid field t.CommR: %w", err)
		}
	}

	// t.Proof ([]uint8) (slice)
	if len("Proof") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"Proof\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("Proof")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("Proof")); err != nil {
		return err
	}

	if len(t.Proof) > cbg.ByteArrayMaxLen {
		return xerrors.Errorf("Byte array in field t.Proof was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajByteString, uint64(len(t.Proof)))); err != nil {
		return err
	}
	if _, err := w.Write(t.Proof); err != nil {
		return err
	}

	// t.PreCommitMessage (cid.Cid) (struct)
	if len("PreCommitMessage") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"PreCommitMessage\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("PreCommitMessage")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("PreCommitMessage")); err != nil {
		return err
	}

	if t.PreCommitMessage == nil {
		if _, err := w.Write(cbg.CborNull); err != nil {
			return err
		}
	} else {
		if err := cbg.WriteCid(w, *t.PreCommitMessage); err != nil {
			return xerrors.Errorf("failed to write cid field t.PreCommitMessage: %w", err)
		}
	}

	// t.Seed (api.SealSeed) (struct)
	if len("Seed") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"Seed\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("Seed")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("Seed")); err != nil {
		return err
	}

	if err := t.Seed.MarshalCBOR(w); err != nil {
		return err
	}

	// t.CommitMessage (cid.Cid) (struct)
	if len("CommitMessage") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"CommitMessage\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("CommitMessage")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("CommitMessage")); err != nil {
		return err
	}

	if t.CommitMessage == nil {
		if _, err := w.Write(cbg.CborNull); err != nil {
			return err
		}
	} else {
		if err := cbg.WriteCid(w, *t.CommitMessage); err != nil {
			return xerrors.Errorf("failed to write cid field t.CommitMessage: %w", err)
		}
	}

	// t.InvalidProofs (uint64) (uint64)
	if len("InvalidProofs") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"InvalidProofs\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("InvalidProofs")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("InvalidProofs")); err != nil {
		return err
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajUnsignedInt, uint64(t.InvalidProofs))); err != nil {
		return err
	}

	// t.FaultReportMsg (cid.Cid) (struct)
	if len("FaultReportMsg") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"FaultReportMsg\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("FaultReportMsg")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("FaultReportMsg")); err != nil {
		return err
	}

	if t.FaultReportMsg == nil {
		if _, err := w.Write(cbg.CborNull); err != nil {
			return err
		}
	} else {
		if err := cbg.WriteCid(w, *t.FaultReportMsg); err != nil {
			return xerrors.Errorf("failed to write cid field t.FaultReportMsg: %w", err)
		}
	}

	// t.LastErr (string) (string)
	if len("LastErr") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"LastErr\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("LastErr")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("LastErr")); err != nil {
		return err
	}

	if len(t.LastErr) > cbg.MaxLength {
		return xerrors.Errorf("Value in field t.LastErr was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len(t.LastErr)))); err != nil {
		return err
	}
	if _, err := w.Write([]byte(t.LastErr)); err != nil {
		return err
	}

	// t.Log ([]sealing.Log) (slice)
	if len("Log") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"Log\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("Log")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("Log")); err != nil {
		return err
	}

	if len(t.Log) > cbg.MaxLength {
		return xerrors.Errorf("Slice value in field t.Log was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajArray, uint64(len(t.Log)))); err != nil {
		return err
	}
	for _, v := range t.Log {
		if err := v.MarshalCBOR(w); err != nil {
			return err
		}
	}
	return nil
}

func (t *SectorInfo) UnmarshalCBOR(r io.Reader) error {
	br := cbg.GetPeeker(r)

	maj, extra, err := cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajMap {
		return fmt.Errorf("cbor input should be of type map")
	}

	if extra > cbg.MaxLength {
		return fmt.Errorf("SectorInfo: map struct too large (%d)", extra)
	}

	var name string
	n := extra

	for i := uint64(0); i < n; i++ {

		{
			sval, err := cbg.ReadString(br)
			if err != nil {
				return err
			}

			name = string(sval)
		}

		switch name {
		// t.State (api.SectorState) (string)
		case "State":

			{
				sval, err := cbg.ReadString(br)
				if err != nil {
					return err
				}

				t.State = api.SectorState(sval)
			}
			// t.SectorID (abi.SectorNumber) (uint64)
		case "SectorID":

			{

				maj, extra, err = cbg.CborReadHeader(br)
				if err != nil {
					return err
				}
				if maj != cbg.MajUnsignedInt {
					return fmt.Errorf("wrong type for uint64 field")
				}
				t.SectorID = abi.SectorNumber(extra)

			}
			// t.Nonce (uint64) (uint64)
		case "Nonce":

			{

				maj, extra, err = cbg.CborReadHeader(br)
				if err != nil {
					return err
				}
				if maj != cbg.MajUnsignedInt {
					return fmt.Errorf("wrong type for uint64 field")
				}
				t.Nonce = uint64(extra)

			}
			// t.SectorType (abi.RegisteredProof) (int64)
		case "SectorType":
			{
				maj, extra, err := cbg.CborReadHeader(br)
				var extraI int64
				if err != nil {
					return err
				}
				switch maj {
				case cbg.MajUnsignedInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 positive overflow")
					}
				case cbg.MajNegativeInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 negative oveflow")
					}
					extraI = -1 - extraI
				default:
					return fmt.Errorf("wrong type for int64 field: %d", maj)
				}

				t.SectorType = abi.RegisteredProof(extraI)
			}
			// t.Pieces ([]sealing.Piece) (slice)
		case "Pieces":

			maj, extra, err = cbg.CborReadHeader(br)
			if err != nil {
				return err
			}

			if extra > cbg.MaxLength {
				return fmt.Errorf("t.Pieces: array too large (%d)", extra)
			}

			if maj != cbg.MajArray {
				return fmt.Errorf("expected cbor array")
			}
			if extra > 0 {
				t.Pieces = make([]Piece, extra)
			}
			for i := 0; i < int(extra); i++ {

				var v Piece
				if err := v.UnmarshalCBOR(br); err != nil {
					return err
				}

				t.Pieces[i] = v
			}

			// t.Ticket (api.SealTicket) (struct)
		case "Ticket":

			{

				if err := t.Ticket.UnmarshalCBOR(br); err != nil {
					return xerrors.Errorf("unmarshaling t.Ticket: %w", err)
				}

			}
			// t.PreCommit1Out (storage.PreCommit1Out) (slice)
		case "PreCommit1Out":

			maj, extra, err = cbg.CborReadHeader(br)
			if err != nil {
				return err
			}

			if extra > cbg.ByteArrayMaxLen {
				return fmt.Errorf("t.PreCommit1Out: byte array too large (%d)", extra)
			}
			if maj != cbg.MajByteString {
				return fmt.Errorf("expected byte array")
			}
			t.PreCommit1Out = make([]byte, extra)
			if _, err := io.ReadFull(br, t.PreCommit1Out); err != nil {
				return err
			}
			// t.CommD (cid.Cid) (struct)
		case "CommD":

			{

				pb, err := br.PeekByte()
				if err != nil {
					return err
				}
				if pb == cbg.CborNull[0] {
					var nbuf [1]byte
					if _, err := br.Read(nbuf[:]); err != nil {
						return err
					}
				} else {

					c, err := cbg.ReadCid(br)
					if err != nil {
						return xerrors.Errorf("failed to read cid field t.CommD: %w", err)
					}

					t.CommD = &c
				}

			}
			// t.CommR (cid.Cid) (struct)
		case "CommR":

			{

				pb, err := br.PeekByte()
				if err != nil {
					return err
				}
				if pb == cbg.CborNull[0] {
					var nbuf [1]byte
					if _, err := br.Read(nbuf[:]); err != nil {
						return err
					}
				} else {

					c, err := cbg.ReadCid(br)
					if err != nil {
						return xerrors.Errorf("failed to read cid field t.CommR: %w", err)
					}

					t.CommR = &c
				}

			}
			// t.Proof ([]uint8) (slice)
		case "Proof":

			maj, extra, err = cbg.CborReadHeader(br)
			if err != nil {
				return err
			}

			if extra > cbg.ByteArrayMaxLen {
				return fmt.Errorf("t.Proof: byte array too large (%d)", extra)
			}
			if maj != cbg.MajByteString {
				return fmt.Errorf("expected byte array")
			}
			t.Proof = make([]byte, extra)
			if _, err := io.ReadFull(br, t.Proof); err != nil {
				return err
			}
			// t.PreCommitMessage (cid.Cid) (struct)
		case "PreCommitMessage":

			{

				pb, err := br.PeekByte()
				if err != nil {
					return err
				}
				if pb == cbg.CborNull[0] {
					var nbuf [1]byte
					if _, err := br.Read(nbuf[:]); err != nil {
						return err
					}
				} else {

					c, err := cbg.ReadCid(br)
					if err != nil {
						return xerrors.Errorf("failed to read cid field t.PreCommitMessage: %w", err)
					}

					t.PreCommitMessage = &c
				}

			}
			// t.Seed (api.SealSeed) (struct)
		case "Seed":

			{

				if err := t.Seed.UnmarshalCBOR(br); err != nil {
					return xerrors.Errorf("unmarshaling t.Seed: %w", err)
				}

			}
			// t.CommitMessage (cid.Cid) (struct)
		case "CommitMessage":

			{

				pb, err := br.PeekByte()
				if err != nil {
					return err
				}
				if pb == cbg.CborNull[0] {
					var nbuf [1]byte
					if _, err := br.Read(nbuf[:]); err != nil {
						return err
					}
				} else {

					c, err := cbg.ReadCid(br)
					if err != nil {
						return xerrors.Errorf("failed to read cid field t.CommitMessage: %w", err)
					}

					t.CommitMessage = &c
				}

			}
			// t.InvalidProofs (uint64) (uint64)
		case "InvalidProofs":

			{

				maj, extra, err = cbg.CborReadHeader(br)
				if err != nil {
					return err
				}
				if maj != cbg.MajUnsignedInt {
					return fmt.Errorf("wrong type for uint64 field")
				}
				t.InvalidProofs = uint64(extra)

			}
			// t.FaultReportMsg (cid.Cid) (struct)
		case "FaultReportMsg":

			{

				pb, err := br.PeekByte()
				if err != nil {
					return err
				}
				if pb == cbg.CborNull[0] {
					var nbuf [1]byte
					if _, err := br.Read(nbuf[:]); err != nil {
						return err
					}
				} else {

					c, err := cbg.ReadCid(br)
					if err != nil {
						return xerrors.Errorf("failed to read cid field t.FaultReportMsg: %w", err)
					}

					t.FaultReportMsg = &c
				}

			}
			// t.LastErr (string) (string)
		case "LastErr":

			{
				sval, err := cbg.ReadString(br)
				if err != nil {
					return err
				}

				t.LastErr = string(sval)
			}
			// t.Log ([]sealing.Log) (slice)
		case "Log":

			maj, extra, err = cbg.CborReadHeader(br)
			if err != nil {
				return err
			}

			if extra > cbg.MaxLength {
				return fmt.Errorf("t.Log: array too large (%d)", extra)
			}

			if maj != cbg.MajArray {
				return fmt.Errorf("expected cbor array")
			}
			if extra > 0 {
				t.Log = make([]Log, extra)
			}
			for i := 0; i < int(extra); i++ {

				var v Log
				if err := v.UnmarshalCBOR(br); err != nil {
					return err
				}

				t.Log[i] = v
			}

		default:
			return fmt.Errorf("unknown struct field %d: '%s'", i, name)
		}
	}

	return nil
}
func (t *Log) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write([]byte{164}); err != nil {
		return err
	}

	// t.Timestamp (uint64) (uint64)
	if len("Timestamp") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"Timestamp\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("Timestamp")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("Timestamp")); err != nil {
		return err
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajUnsignedInt, uint64(t.Timestamp))); err != nil {
		return err
	}

	// t.Trace (string) (string)
	if len("Trace") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"Trace\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("Trace")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("Trace")); err != nil {
		return err
	}

	if len(t.Trace) > cbg.MaxLength {
		return xerrors.Errorf("Value in field t.Trace was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len(t.Trace)))); err != nil {
		return err
	}
	if _, err := w.Write([]byte(t.Trace)); err != nil {
		return err
	}

	// t.Message (string) (string)
	if len("Message") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"Message\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("Message")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("Message")); err != nil {
		return err
	}

	if len(t.Message) > cbg.MaxLength {
		return xerrors.Errorf("Value in field t.Message was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len(t.Message)))); err != nil {
		return err
	}
	if _, err := w.Write([]byte(t.Message)); err != nil {
		return err
	}

	// t.Kind (string) (string)
	if len("Kind") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"Kind\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("Kind")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("Kind")); err != nil {
		return err
	}

	if len(t.Kind) > cbg.MaxLength {
		return xerrors.Errorf("Value in field t.Kind was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len(t.Kind)))); err != nil {
		return err
	}
	if _, err := w.Write([]byte(t.Kind)); err != nil {
		return err
	}
	return nil
}

func (t *Log) UnmarshalCBOR(r io.Reader) error {
	br := cbg.GetPeeker(r)

	maj, extra, err := cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajMap {
		return fmt.Errorf("cbor input should be of type map")
	}

	if extra > cbg.MaxLength {
		return fmt.Errorf("Log: map struct too large (%d)", extra)
	}

	var name string
	n := extra

	for i := uint64(0); i < n; i++ {

		{
			sval, err := cbg.ReadString(br)
			if err != nil {
				return err
			}

			name = string(sval)
		}

		switch name {
		// t.Timestamp (uint64) (uint64)
		case "Timestamp":

			{

				maj, extra, err = cbg.CborReadHeader(br)
				if err != nil {
					return err
				}
				if maj != cbg.MajUnsignedInt {
					return fmt.Errorf("wrong type for uint64 field")
				}
				t.Timestamp = uint64(extra)

			}
			// t.Trace (string) (string)
		case "Trace":

			{
				sval, err := cbg.ReadString(br)
				if err != nil {
					return err
				}

				t.Trace = string(sval)
			}
			// t.Message (string) (string)
		case "Message":

			{
				sval, err := cbg.ReadString(br)
				if err != nil {
					return err
				}

				t.Message = string(sval)
			}
			// t.Kind (string) (string)
		case "Kind":

			{
				sval, err := cbg.ReadString(br)
				if err != nil {
					return err
				}

				t.Kind = string(sval)
			}

		default:
			return fmt.Errorf("unknown struct field %d: '%s'", i, name)
		}
	}

	return nil
}
