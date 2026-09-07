package dec128

import (
	"encoding/binary"
	"io"
	"slices"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// Fixed-point int128 interchange, the model shared by Apache Arrow Decimal128, Parquet DECIMAL, Spark and DuckDB:
// a 16-byte two's-complement integer coefficient whose scale lives in the schema. Arrow stores it little-endian,
// Parquet (FIXED_LEN_BYTE_ARRAY) and most wire formats big-endian; the byte order is a parameter.
//
// The int128 range is |coefficient| < 2^127, one bit narrower than Dec128's, so EncodeInt128 can fail on the widest
// values; DecodeInt128 is total.

// Int128Bytes is the size of an int128 value.
const Int128Bytes = 16

// FitsInt128 reports whether d can be encoded by EncodeInt128 at the given scale: it is not NaN, its coefficient at
// that scale fits in 127 bits, and no digits would be lost.
func (d Dec128) FitsInt128(scale uint8) bool {
	if d.state >= state.Error {
		return false
	}
	r := d.Rescale(scale)
	if r.state >= state.Error || !r.Equal(d) {
		return false
	}
	return r.coef.Hi>>63 == 0 || (r.coef.Hi == 1<<63 && r.coef.Lo == 0 && r.state == state.Neg)
}

// EncodeInt128 writes d at the given scale as a two's-complement int128 into buf using order, and returns the number of
// bytes written (Int128Bytes). The scale is the column's: d is rescaled to it exactly, and an error carrying
// state.Inexact is returned if digits would be lost (round first with RescaleRound), state.Overflow if the coefficient
// does not fit in 127 bits, and the NaN's own state if d is NaN.
func (d Dec128) EncodeInt128(buf []byte, scale uint8, order binary.ByteOrder) (int, error) {
	if len(buf) < Int128Bytes {
		return 0, io.ErrShortBuffer
	}
	if d.state >= state.Error {
		return 0, d.state.Error()
	}
	r := d.Rescale(scale)
	switch {
	case r.state >= state.Error:
		return 0, r.state.Error()
	case !r.Equal(d):
		return 0, state.Inexact.Error()
	}

	c := r.coef
	if r.state == state.Neg {
		c, _ = uint128.Zero.SubBorrow(c) // two's complement
		if c.Hi>>63 == 0 && !c.IsZero() {
			return 0, state.Overflow.Error() // magnitude above 2^127
		}
	} else if c.Hi>>63 == 1 {
		return 0, state.Overflow.Error()
	}

	if order == binary.BigEndian {
		order.PutUint64(buf[0:], c.Hi)
		order.PutUint64(buf[8:], c.Lo)
	} else {
		order.PutUint64(buf[0:], c.Lo)
		order.PutUint64(buf[8:], c.Hi)
	}

	return Int128Bytes, nil
}

// AppendInt128 appends the two's-complement encoding of d at the given scale to buf, as EncodeInt128 writes it into a
// caller buffer, and returns the extended slice. It does not allocate when buf has room. On error buf is returned
// unchanged.
func (d Dec128) AppendInt128(buf []byte, scale uint8, order binary.ByteOrder) ([]byte, error) {
	// encode in place: a scratch array would escape through the ByteOrder interface and cost an allocation
	n := len(buf)
	out := slices.Grow(buf, Int128Bytes)[:n+Int128Bytes]
	if _, err := d.EncodeInt128(out[n:], scale, order); err != nil {
		return buf, err
	}
	return out, nil
}

// DecodeInt128 decodes a two's-complement int128 at the given scale from buf, which must hold exactly Int128Bytes,
// using order. A scale above MaxScale is accepted when the trailing digits are zero, and yields NaN(ScaleOutOfRange)
// otherwise.
func (d *Dec128) DecodeInt128(buf []byte, scale uint8, order binary.ByteOrder) error {
	if len(buf) != Int128Bytes {
		return io.ErrUnexpectedEOF
	}
	var c uint128.Uint128
	if order == binary.BigEndian {
		c.Hi = order.Uint64(buf[0:])
		c.Lo = order.Uint64(buf[8:])
	} else {
		c.Lo = order.Uint64(buf[0:])
		c.Hi = order.Uint64(buf[8:])
	}

	st := state.Default
	if c.Hi>>63 == 1 {
		st = state.Neg
		c, _ = uint128.Zero.SubBorrow(c) // magnitude; -2^127 stays 2^127, which fits
	}
	if c.IsZero() {
		st = state.Default
	}

	s := int(scale)
	for s > int(MaxScale) {
		q, r, _ := c.QuoRemPow10(1)
		if r != 0 {
			*d = Dec128{state: state.ScaleOutOfRange}
			return nil
		}
		c = q
		s--
	}
	*d = Dec128{coef: c, scale: uint8(s), state: st}
	return nil
}
