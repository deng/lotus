package api

import (
	"fmt"
	"io"

	"github.com/filecoin-project/lotus/chain/types"
	cbg "github.com/whyrusleeping/cbor-gen"
	xerrors "golang.org/x/xerrors"
)

// Code generated by github.com/whyrusleeping/cbor-gen. DO NOT EDIT.

var _ = xerrors.Errorf

func (t *PaymentInfo) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write([]byte{163}); err != nil {
		return err
	}

	// t.Channel (address.Address) (struct)
	if len("Channel") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"Channel\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("Channel")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("Channel")); err != nil {
		return err
	}

	if err := t.Channel.MarshalCBOR(w); err != nil {
		return err
	}

	// t.ChannelMessage (cid.Cid) (struct)
	if len("ChannelMessage") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"ChannelMessage\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("ChannelMessage")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("ChannelMessage")); err != nil {
		return err
	}

	if t.ChannelMessage == nil {
		if _, err := w.Write(cbg.CborNull); err != nil {
			return err
		}
	} else {
		if err := cbg.WriteCid(w, *t.ChannelMessage); err != nil {
			return xerrors.Errorf("failed to write cid field t.ChannelMessage: %w", err)
		}
	}

	// t.Vouchers ([]*types.SignedVoucher) (slice)
	if len("Vouchers") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"Vouchers\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("Vouchers")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("Vouchers")); err != nil {
		return err
	}

	if len(t.Vouchers) > cbg.MaxLength {
		return xerrors.Errorf("Slice value in field t.Vouchers was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajArray, uint64(len(t.Vouchers)))); err != nil {
		return err
	}
	for _, v := range t.Vouchers {
		if err := v.MarshalCBOR(w); err != nil {
			return err
		}
	}
	return nil
}

func (t *PaymentInfo) UnmarshalCBOR(r io.Reader) error {
	br := cbg.GetPeeker(r)

	maj, extra, err := cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajMap {
		return fmt.Errorf("cbor input should be of type map")
	}

	if extra > cbg.MaxLength {
		return fmt.Errorf("PaymentInfo: map struct too large (%d)", extra)
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
		// t.Channel (address.Address) (struct)
		case "Channel":

			{

				if err := t.Channel.UnmarshalCBOR(br); err != nil {
					return err
				}

			}
			// t.ChannelMessage (cid.Cid) (struct)
		case "ChannelMessage":

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
						return xerrors.Errorf("failed to read cid field t.ChannelMessage: %w", err)
					}

					t.ChannelMessage = &c
				}

			}
			// t.Vouchers ([]*types.SignedVoucher) (slice)
		case "Vouchers":

			maj, extra, err = cbg.CborReadHeader(br)
			if err != nil {
				return err
			}

			if extra > cbg.MaxLength {
				return fmt.Errorf("t.Vouchers: array too large (%d)", extra)
			}

			if maj != cbg.MajArray {
				return fmt.Errorf("expected cbor array")
			}
			if extra > 0 {
				t.Vouchers = make([]*types.SignedVoucher, extra)
			}
			for i := 0; i < int(extra); i++ {

				var v types.SignedVoucher
				if err := v.UnmarshalCBOR(br); err != nil {
					return err
				}

				t.Vouchers[i] = &v
			}

		default:
		}
	}

	return nil
}
func (t *SealedRef) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write([]byte{163}); err != nil {
		return err
	}

	// t.SectorID (uint64) (uint64)
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

	// t.Offset (uint64) (uint64)
	if len("Offset") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"Offset\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("Offset")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("Offset")); err != nil {
		return err
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajUnsignedInt, uint64(t.Offset))); err != nil {
		return err
	}

	// t.Size (uint64) (uint64)
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
	return nil
}

func (t *SealedRef) UnmarshalCBOR(r io.Reader) error {
	br := cbg.GetPeeker(r)

	maj, extra, err := cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajMap {
		return fmt.Errorf("cbor input should be of type map")
	}

	if extra > cbg.MaxLength {
		return fmt.Errorf("SealedRef: map struct too large (%d)", extra)
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
		// t.SectorID (uint64) (uint64)
		case "SectorID":

			maj, extra, err = cbg.CborReadHeader(br)
			if err != nil {
				return err
			}
			if maj != cbg.MajUnsignedInt {
				return fmt.Errorf("wrong type for uint64 field")
			}
			t.SectorID = uint64(extra)
			// t.Offset (uint64) (uint64)
		case "Offset":

			maj, extra, err = cbg.CborReadHeader(br)
			if err != nil {
				return err
			}
			if maj != cbg.MajUnsignedInt {
				return fmt.Errorf("wrong type for uint64 field")
			}
			t.Offset = uint64(extra)
			// t.Size (uint64) (uint64)
		case "Size":

			maj, extra, err = cbg.CborReadHeader(br)
			if err != nil {
				return err
			}
			if maj != cbg.MajUnsignedInt {
				return fmt.Errorf("wrong type for uint64 field")
			}
			t.Size = uint64(extra)

		default:
		}
	}

	return nil
}
func (t *SealedRefs) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write([]byte{161}); err != nil {
		return err
	}

	// t.Refs ([]api.SealedRef) (slice)
	if len("Refs") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"Refs\" was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len("Refs")))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("Refs")); err != nil {
		return err
	}

	if len(t.Refs) > cbg.MaxLength {
		return xerrors.Errorf("Slice value in field t.Refs was too long")
	}

	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajArray, uint64(len(t.Refs)))); err != nil {
		return err
	}
	for _, v := range t.Refs {
		if err := v.MarshalCBOR(w); err != nil {
			return err
		}
	}
	return nil
}

func (t *SealedRefs) UnmarshalCBOR(r io.Reader) error {
	br := cbg.GetPeeker(r)

	maj, extra, err := cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajMap {
		return fmt.Errorf("cbor input should be of type map")
	}

	if extra > cbg.MaxLength {
		return fmt.Errorf("SealedRefs: map struct too large (%d)", extra)
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
		// t.Refs ([]api.SealedRef) (slice)
		case "Refs":

			maj, extra, err = cbg.CborReadHeader(br)
			if err != nil {
				return err
			}

			if extra > cbg.MaxLength {
				return fmt.Errorf("t.Refs: array too large (%d)", extra)
			}

			if maj != cbg.MajArray {
				return fmt.Errorf("expected cbor array")
			}
			if extra > 0 {
				t.Refs = make([]SealedRef, extra)
			}
			for i := 0; i < int(extra); i++ {

				var v SealedRef
				if err := v.UnmarshalCBOR(br); err != nil {
					return err
				}

				t.Refs[i] = v
			}

		default:
		}
	}

	return nil
}
