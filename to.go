package dec128

import (
	"github.com/jokruger/dec128/errors"
	"github.com/jokruger/dec128/uint128"
)

// Uint64 returns the Dec128 decomposed into uint64 coefficient and uint8 exponent.
// Negative values are not allowed.
func (self Dec128) Uint64() (uint64, uint8, error) {
	if self.err != errors.None {
		return 0, 0, self.err.Value()
	}

	if self.neg {
		return 0, 0, errors.Negative.Value()
	}

	i, err := self.coef.Uint64()
	if err != errors.None {
		return 0, 0, err.Value()
	}

	return i, self.exp, nil
}

// Uint128 returns the Dec128 decomposed into uint128 coefficient and uint8 exponent.
// Negative values are not allowed.
func (self Dec128) Uint128() (uint128.Uint128, uint8, error) {
	if self.err != errors.None {
		return uint128.Zero, 0, self.err.Value()
	}

	if self.neg {
		return uint128.Zero, 0, errors.Negative.Value()
	}

	return self.coef, self.exp, nil
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
	for i := range uint128.MaxStrLen {
		buf[i] = '0'
	}
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
