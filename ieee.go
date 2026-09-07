package dec128

import (
	"encoding/binary"
	"io"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// IEEE 754-2019 decimal128 in the BID (binary integer decimal) encoding: 16 bytes, little-endian, as used by MongoDB's
// BSON Decimal128 and Intel's decimal library.
//
//	bit 127        sign
//	bits 126..113  biased exponent (14 bits, bias 6176) when bits 126..125 are not 11
//	bits 112..0    coefficient, at most 10^34 - 1
//	bits 126..122  11110 = infinity, 11111 = NaN (bit 121: 0 quiet, 1 signalling)
//
// Dec128 is wider in digits (up to 39) and narrower in exponent (scale 0..19) than decimal128, so neither direction is
// total: EncodeIEEE rounds a coefficient above 10^34 - 1 to 34 significant digits using the rounding mode set by
// SetArithmeticRounding (ROUND_NAN refuses), and DecodeIEEE returns NaN for a value outside Dec128's range.
// This package does not read or write the DPD encoding.

// IEEEBytes is the size of a decimal128 value.
const IEEEBytes = 16

const (
	ieeeBias    = 6176
	ieeeMaxCoef = 34 // significant digits of a decimal128 coefficient
)

var ieeeCoefLimit = Pow10Uint128[ieeeMaxCoef] // first coefficient that does not fit

// FitsIEEE reports whether d can be encoded as decimal128 exactly: it is a NaN, or its coefficient has at most 34
// significant digits. NULL never fits.
func (d Dec128) FitsIEEE() bool {
	if d.state == state.Null {
		return false
	}
	return d.state >= state.Error || d.coef.Compare(ieeeCoefLimit) < 0
}

// EncodeIEEE writes d as a decimal128 (BID) into buf and returns the number of bytes written (IEEEBytes), or
// io.ErrShortBuffer. A NaN becomes a quiet NaN whose payload carries the state code, so the reason survives a round
// trip through a conforming implementation. A coefficient above 10^34 - 1 is rounded to 34 significant digits with
// the arithmetic rounding mode; under ROUND_NAN that returns an error carrying state.Inexact, and a NULL value returns
// an error carrying state.Null.
func (d Dec128) EncodeIEEE(buf []byte) (int, error) {
	if len(buf) < IEEEBytes {
		return 0, io.ErrShortBuffer
	}
	if d.state == state.Null {
		return 0, state.Null.Error()
	}
	if d.state >= state.Error {
		binary.LittleEndian.PutUint64(buf[0:], uint64(d.state))
		binary.LittleEndian.PutUint64(buf[8:], 0b11111<<58)
		return IEEEBytes, nil
	}

	coef := d.coef
	exp := -int(d.scale)
	if coef.Compare(ieeeCoefLimit) >= 0 {
		// 35 to 39 digits: drop enough to reach 34, rounding; a round-up can carry to 10^34, which then loses one more
		// digit exactly. k = digits - 34, in 1..5: the coefficient is below 10^39, so the largest power that has to be
		// compared is 10^38.
		k := uint8(1)
		for k < 5 && coef.Compare(Pow10Uint128[ieeeMaxCoef+int(k)]) >= 0 {
			k++
		}
		// q < 10^34 + 1 after rounding, so it cannot overflow; q == 10^34 (a carry out of 34 nines) loses one more
		// digit exactly.
		q, _, inexact := reduceWide(coef, uint128.Zero, k, d.state, arithmeticRounding)
		if inexact && arithmeticRounding == ROUND_NAN {
			return 0, state.Inexact.Error()
		}
		if q.Equal(ieeeCoefLimit) {
			q, k = Pow10Uint128[ieeeMaxCoef-1], k+1
		}
		coef = q
		exp += int(k)
	}

	hi := coef.Hi | uint64(exp+ieeeBias)<<49
	if d.state == state.Neg {
		hi |= 1 << 63
	}
	binary.LittleEndian.PutUint64(buf[0:], coef.Lo)
	binary.LittleEndian.PutUint64(buf[8:], hi)
	return IEEEBytes, nil
}

// DecodeIEEE decodes a decimal128 (BID) from buf, which must hold exactly IEEEBytes. Infinities become NaN(Overflow);
// a NaN keeps the state code from its payload when it is one this package produced, and is a NaN carrying state.NaN
// otherwise; a negative zero becomes zero. A value whose exponent is above 0 is multiplied out and must fit in 128 bits
// (else NaN(Overflow)); one whose exponent is below -MaxScale is stripped of trailing zeros and must reach MaxScale
// (else NaN(ScaleOutOfRange)). A non-canonical coefficient (10^34 or more) is zero, as IEEE 754 prescribes for BID.
func (d *Dec128) DecodeIEEE(buf []byte) error {
	if len(buf) != IEEEBytes {
		return io.ErrUnexpectedEOF
	}
	lo := binary.LittleEndian.Uint64(buf[0:])
	hi := binary.LittleEndian.Uint64(buf[8:])
	neg := hi>>63 == 1
	comb := (hi >> 58) & 0b11111

	switch {
	case comb == 0b11111:
		// NaN: the payload is the trailing significand; keep a state code we wrote
		payload := lo
		if hi&(1<<50-1) == 0 && payload >= uint64(state.Error) && payload <= 0xFF && state.State(payload).IsValid() && payload != uint64(state.Null) {
			*d = Dec128{state: state.State(payload)}
		} else {
			*d = Dec128{state: state.NaN}
		}
		return nil
	case comb == 0b11110:
		*d = Dec128{state: state.Overflow}
		return nil
	case comb>>3 == 0b11:
		// non-canonical large-coefficient form: the value is zero by definition
		*d = Dec128{}
		return nil
	}

	exp := int(hi>>49&0x3FFF) - ieeeBias
	coef := uint128.Uint128{Lo: lo, Hi: hi & (1<<49 - 1)}
	if coef.Compare(ieeeCoefLimit) >= 0 {
		*d = Dec128{}
		return nil
	}
	if coef.IsZero() {
		*d = Dec128{scale: uint8(min(max(-exp, 0), int(MaxScale)))}
		return nil
	}
	st := state.Default
	if neg {
		st = state.Neg
	}

	switch {
	case exp > 0:
		if exp > 38 {
			*d = Dec128{state: state.Overflow}
			return nil
		}
		var s state.State
		if coef, s = coef.Mul(Pow10Uint128[exp]); s >= state.Error {
			*d = Dec128{state: state.Overflow}
			return nil
		}
		exp = 0
	case exp < -int(MaxScale):
		for exp < -int(MaxScale) {
			q, r, _ := coef.QuoRemPow10(1)
			if r != 0 {
				*d = Dec128{state: state.ScaleOutOfRange}
				return nil
			}
			coef = q
			exp++
		}
	}
	*d = Dec128{coef: coef, scale: uint8(-exp), state: st}
	return nil
}
