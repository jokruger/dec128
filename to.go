package dec128

import (
	"github.com/jokruger/dec128/errors"
	"github.com/jokruger/dec128/uint128"
)

// EncodeToUint64 returns the Dec128 encoded as uint64 coefficient with requested exponent.
// Negative and too large values are not allowed.
func (self Dec128) EncodeToUint64(exp uint8) (uint64, error) {
	if self.neg {
		return 0, errors.Negative.Value()
	}

	d := self.Rescale(exp)

	if d.err != errors.None {
		return 0, d.err.Value()
	}

	i, err := d.coef.Uint64()
	if err != errors.None {
		return 0, err.Value()
	}

	return i, nil
}

// EncodeToUint128 returns the Dec128 encoded as uint128 coefficient with requested exponent.
// Negative values are not allowed.
func (self Dec128) EncodeToUint128(exp uint8) (uint128.Uint128, error) {
	if self.neg {
		return uint128.Zero, errors.Negative.Value()
	}

	d := self.Rescale(exp)

	if d.err != errors.None {
		return uint128.Zero, d.err.Value()
	}

	return d.coef, nil
}

// String returns the string representation of the Dec128 with the trailing zeros removed.
// If the Dec128 is zero, the string "0" is returned.
// If the Dec128 is NaN, the string "NaN" is returned.
func (self Dec128) String() string {
	if self.err != errors.None {
		return NaNStr
	}

	if self.IsZero() {
		return ZeroStr
	}

	sb, trim := self.toString()
	i := len(sb)

	if trim {
		for i > 0 && sb[i-1] == '0' {
			i--
		}

		if i > 0 && sb[i-1] == '.' {
			i--
		}
	}

	return sb[:i]
}

// StringFixed returns the string representation of the Dec128 with the trailing zeros preserved.
// If the Dec128 is NaN, the string "NaN" is returned.
func (self Dec128) StringFixed() string {
	if self.err != errors.None {
		return NaNStr
	}

	if self.IsZero() {
		return zeroStrs[self.exp]
	}

	sb, _ := self.toString()

	return sb
}

func (self Dec128) toString() (string, bool) {
	buf := [uint128.MaxStrLen]byte{}
	n := self.coef.StringToBuf(buf[:])
	coef := buf[n:]

	prec := int(self.exp)
	sz := len(coef)
	sb := [uint128.MaxStrLen + 2]byte{}
	i := 0

	if self.neg {
		sb[i] = '-'
		i++
	}

	if prec == 0 {
		for j := 0; j < sz; j++ {
			sb[i] = coef[j]
			i++
		}
		return string(sb[:i]), false
	}

	if prec > sz {
		sb[i] = '0'
		i++
		sb[i] = '.'
		i++
		for j := 0; j < prec-sz; j++ {
			sb[i] = '0'
			i++
		}
		for j := 0; j < sz; j++ {
			sb[i] = coef[j]
			i++
		}
	} else if prec == sz {
		sb[i] = '0'
		i++
		sb[i] = '.'
		i++
		for j := 0; j < sz; j++ {
			sb[i] = coef[j]
			i++
		}
	} else {
		for j := 0; j < sz-prec; j++ {
			sb[i] = coef[j]
			i++
		}
		sb[i] = '.'
		i++
		for j := sz - prec; j < sz; j++ {
			sb[i] = coef[j]
			i++
		}
	}

	return string(sb[:i]), true
}
