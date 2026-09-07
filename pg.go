package dec128

import (
	"encoding/binary"
	"io"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// PostgreSQL numeric binary format, as produced by numeric_send and consumed by numeric_recv (the binary protocol used
// by pgx and other drivers):
//
//	int16 ndigits | int16 weight | uint16 sign | uint16 dscale | int16 digit[ndigits]
//
// All fields are big-endian. Each digit is a base-10000 group; weight is the power of 10000 of the first group; dscale
// is the display scale, which maps one-to-one onto Dec128's scale, so 1.50 stays 1.50 through the round trip. sign is
// 0x0000 for positive, 0x4000 for negative, 0xC000 for NaN, 0xD000 and 0xF000 for the infinities. The encoding is the
// stripped form PostgreSQL itself produces: no leading or trailing zero groups.

// MaxPgNumericBytes is the largest encoding EncodePgNumeric produces: the 8-byte header plus 11 groups (39 digits,
// whatever the split around the decimal point).
const MaxPgNumericBytes = 8 + 2*11

const (
	pgNumericPos    = 0x0000
	pgNumericNeg    = 0x4000
	pgNumericNaN    = 0xC000
	pgNumericPosInf = 0xD000
	pgNumericNegInf = 0xF000
)

// EncodePgNumeric writes the PostgreSQL binary form of d into buf and returns the number of bytes written, or
// io.ErrShortBuffer if buf is too small (MaxPgNumericBytes always suffices). A NaN encodes as a PostgreSQL NaN; the
// reason does not survive. A NULL value is not a numeric and returns an error: the caller decides how to send SQL NULL.
func (d Dec128) EncodePgNumeric(buf []byte) (int, error) {
	if d.state == state.Null {
		return 0, state.Null.Error()
	}
	if len(buf) < 8 {
		return 0, io.ErrShortBuffer
	}
	if d.state >= state.Error {
		binary.BigEndian.PutUint16(buf[0:], 0)
		binary.BigEndian.PutUint16(buf[2:], 0)
		binary.BigEndian.PutUint16(buf[4:], pgNumericNaN)
		binary.BigEndian.PutUint16(buf[6:], 0)
		return 8, nil
	}

	// Split into the integer part and the fraction padded to whole groups of four digits:
	// with scale <= 19 the padded fraction has at most 5 groups and 20 digits, which fits a Uint128 (10^20 < 2^67).
	intPart, frac, _ := d.coef.QuoRemPow10(d.scale)
	fracGroups := (int(d.scale) + 3) / 4
	pad := uint8(fracGroups*4 - int(d.scale))
	fracCoef, _ := uint128.Uint128{Lo: frac}.Mul64(Pow10Uint64[pad])

	// Extract groups, least significant first, into a scratch array.
	var groups [11]uint16
	n := 0
	for i := 0; i < fracGroups; i++ {
		var g uint64
		fracCoef, g, _ = fracCoef.QuoRemPow10(4)
		groups[n] = uint16(g)
		n++
	}
	weight := int16(-1)
	for !intPart.IsZero() {
		var g uint64
		intPart, g, _ = intPart.QuoRemPow10(4)
		groups[n] = uint16(g)
		n++
		weight++
	}

	// Strip trailing zero groups (the least significant, at the front of the array) and leading zero groups
	// (at the back), adjusting the weight for the latter.
	first := 0
	for first < n && groups[first] == 0 {
		first++
	}
	for n > first && groups[n-1] == 0 {
		n--
		weight--
	}
	ndigits := n - first
	if ndigits == 0 {
		weight = 0
	}

	if len(buf) < 8+2*ndigits {
		return 0, io.ErrShortBuffer
	}
	sign := uint16(pgNumericPos)
	if d.state == state.Neg {
		sign = pgNumericNeg
	}
	binary.BigEndian.PutUint16(buf[0:], uint16(ndigits))
	binary.BigEndian.PutUint16(buf[2:], uint16(weight))
	binary.BigEndian.PutUint16(buf[4:], sign)
	binary.BigEndian.PutUint16(buf[6:], uint16(d.scale))
	pos := 8
	for i := n - 1; i >= first; i-- {
		binary.BigEndian.PutUint16(buf[pos:], groups[i])
		pos += 2
	}

	return pos, nil
}

// AppendPgNumeric appends the PostgreSQL binary form of d to buf. A NULL value appends nothing and returns an error.
func (d Dec128) AppendPgNumeric(buf []byte) ([]byte, error) {
	var tmp [MaxPgNumericBytes]byte
	n, err := d.EncodePgNumeric(tmp[:])
	if err != nil {
		return buf, err
	}
	return append(buf, tmp[:n]...), nil
}

// DecodePgNumeric decodes the PostgreSQL binary form of a numeric from buf, which must hold exactly one value.
// A PostgreSQL NaN decodes to a NaN carrying state.NaN and the infinities to NaN(Overflow). A value with more
// significant digits than fit in 128 bits decodes to NaN(Overflow), and one with more decimals than MaxScale, after
// trailing zeros are removed, to NaN(ScaleOutOfRange): a stored value is never silently rounded on the way in. An error
// is returned only for a malformed buffer.
func (d *Dec128) DecodePgNumeric(buf []byte) error {
	if len(buf) < 8 {
		return io.ErrShortBuffer
	}
	ndigits := int(int16(binary.BigEndian.Uint16(buf[0:])))
	weight := int(int16(binary.BigEndian.Uint16(buf[2:])))
	sign := binary.BigEndian.Uint16(buf[4:])
	dscale := int(binary.BigEndian.Uint16(buf[6:]))
	if ndigits < 0 || len(buf) != 8+2*ndigits {
		return io.ErrUnexpectedEOF
	}

	var st state.State
	switch sign {
	case pgNumericPos:
	case pgNumericNeg:
		st = state.Neg
	case pgNumericNaN:
		*d = Dec128{state: state.NaN}
		return nil
	case pgNumericPosInf, pgNumericNegInf:
		*d = Dec128{state: state.Overflow}
		return nil
	default:
		return state.InvalidFormat.Error()
	}

	// N = the digit groups read as one base-10000 number; the value is then N * 10000^(weight - ndigits + 1), and the
	// coefficient at dscale is N * 10^(4*(weight - ndigits + 1) + dscale). N is accumulated in 192 bits: a 39-digit
	// value padded to whole groups can have 42 digits, and a sender that does not strip zero groups can add more.
	// Anything beyond 192 bits cannot fit anyway.
	var n widened
	for i := 0; i < ndigits; i++ {
		g := binary.BigEndian.Uint16(buf[8+2*i:])
		if g >= 10000 {
			return state.InvalidFormat.Error()
		}
		if n.hi>>50 != 0 {
			*d = Dec128{state: state.Overflow}
			return nil
		}
		lo, hi := n.lo.Mul64Carry(10000)
		lo, carry := lo.AddCarry(uint128.Uint128{Lo: uint64(g)})
		n = widened{lo: lo, hi: hi + n.hi*10000 + carry}
	}
	if n.lo.IsZero() && n.hi == 0 {
		*d = Dec128{scale: uint8(min(dscale, int(MaxScale)))}
		return nil
	}

	shift := 4*(weight-ndigits+1) + dscale
	var coef uint128.Uint128
	switch {
	case shift > 0:
		if n.hi != 0 || shift > 38 {
			*d = Dec128{state: state.Overflow}
			return nil
		}
		var s state.State
		if coef, s = n.lo.Mul(Pow10Uint128[shift]); s >= state.Error {
			*d = Dec128{state: state.Overflow}
			return nil
		}
	case shift < 0:
		// digits below dscale are padding within the last group and must be zero. A stripped encoding has at most three
		// of them; more than 38 cannot come from any numeric PostgreSQL sends and would index past the power table.
		if -shift > 38 {
			return state.InvalidFormat.Error()
		}
		q, r, s := uint128.QuoRem256By128(n.lo, uint128.Uint128{Lo: n.hi}, Pow10Uint128[-shift])
		if s >= state.Error {
			*d = Dec128{state: state.Overflow}
			return nil
		}
		if !r.IsZero() {
			return state.InvalidFormat.Error()
		}
		coef = q
	default:
		if n.hi != 0 {
			*d = Dec128{state: state.Overflow}
			return nil
		}
		coef = n.lo
	}

	scale := dscale
	for scale > int(MaxScale) {
		q, r, _ := coef.QuoRemPow10(1)
		if r != 0 {
			*d = Dec128{state: state.ScaleOutOfRange}
			return nil
		}
		coef = q
		scale--
	}
	*d = Dec128{coef: coef, scale: uint8(scale), state: st}
	return nil
}
