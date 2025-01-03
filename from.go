package dec128

import (
	"strings"

	"github.com/jokruger/dec128/errors"
	"github.com/jokruger/dec128/uint128"
)

func FromUint64(coef uint64, prec uint8) Dec128 {
	if prec > maxPrecision {
		return NaN(errors.PrecisionOutOfRange)
	}

	if coef == 0 {
		return Zero
	}

	return Dec128{coef: uint128.FromUint64(coef), prec: prec}
}

func FromUint128(coef uint128.Uint128, prec uint8) Dec128 {
	if prec > maxPrecision {
		return NaN(errors.PrecisionOutOfRange)
	}

	if coef.IsZero() {
		return Zero
	}

	return Dec128{coef: coef, prec: prec}
}

func FromString(s string) Dec128 {
	sz := len(s)

	if sz == 0 {
		return Zero
	}

	switch s {
	case "0", "0.0", "0.00", ".0", ".00":
		return Zero
	case "+", "-", ".", "+.", "-.":
		return NaN(errors.InvalidFormat)
	}

	var i, prec int
	var neg bool

	switch s[0] {
	case '+':
		i++
	case '-':
		neg = true
		i++
	}

	if sz <= uint128.MaxSafeStrLen64 {
		// safe to parse with uint64 as coef
		var u uint64
		for ; i < sz; i++ {
			if s[i] == '.' {
				if prec != 0 {
					return NaN(errors.InvalidFormat)
				}
				prec = sz - i - 1
				continue
			}
			if s[i] < '0' || s[i] > '9' {
				return NaN(errors.InvalidFormat)
			}
			u = u*10 + uint64(s[i]-'0')
		}
		if u == 0 {
			return Zero
		}
		return Dec128{coef: uint128.FromUint64(u), prec: uint8(prec), neg: neg}
	}

	j := strings.IndexByte(s, '.')
	if j == sz-1 {
		return NaN(errors.InvalidFormat)
	}
	if j == -1 {
		coef, err := uint128.FromString(s[i:])
		if err != errors.None {
			return NaN(err)
		}
		return Dec128{coef: coef, prec: 0, neg: neg}
	}

	prec = sz - j - 1
	if prec > uint128.MaxSafeStrLen64 {
		return NaN(errors.PrecisionOutOfRange)
	}

	ipart, err := uint128.FromString(s[i:j])
	if err != errors.None {
		return NaN(err)
	}

	fpart, err := uint128.FromString(s[j+1:])
	if err != errors.None {
		return NaN(err)
	}

	// max prec is 19, so the fpart.Hi is always 0 and prec is always <= len(pow10)
	coef, err := ipart.Mul64(pow10[prec])
	if err != errors.None {
		return NaN(err)
	}

	coef, err = coef.Add64(fpart.Lo)
	if err != errors.None {
		return NaN(err)
	}

	if coef.IsZero() {
		return Zero
	}

	return Dec128{coef: coef, prec: uint8(prec), neg: neg}
}
