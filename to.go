package dec128

import (
	"strings"

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

	coef := self.coef.String()
	prec := int(self.exp)

	if prec == 0 {
		if self.neg {
			return "-" + coef
		}
		return coef
	}

	sz := len(coef)

	if prec > sz {
		coef = strings.Repeat("0", prec-sz) + strings.TrimRight(coef, "0")
		if self.neg {
			coef = "-0." + coef
		} else {
			coef = "0." + coef
		}
	} else if prec == sz {
		if self.neg {
			coef = "-0." + strings.TrimRight(coef, "0")
		} else {
			coef = "0." + strings.TrimRight(coef, "0")
		}
	} else {
		if self.neg {
			coef = "-" + coef[:sz-prec] + "." + strings.TrimRight(coef[sz-prec:], "0")
		} else {
			coef = coef[:sz-prec] + "." + strings.TrimRight(coef[sz-prec:], "0")
		}
	}

	return strings.TrimRight(coef, ".")
}

// StringFixed returns the string representation of the Dec128 with the trailing zeros preserved.
// If the Dec128 is NaN, the string "NaN" is returned.
func (self Dec128) StringFixed() string {
	if self.err != errors.None {
		return NaNStr
	}

	if self.IsZero() {
		if self.exp == 0 {
			return "0"
		}
		return "0." + strings.Repeat("0", int(self.exp))
	}

	coef := self.coef.String()
	prec := int(self.exp)

	if prec == 0 {
		if self.neg {
			return "-" + coef
		}
		return coef
	}

	sz := len(coef)

	if prec > sz {
		coef = strings.Repeat("0", prec-sz) + coef
		if self.neg {
			coef = "-0." + coef
		} else {
			coef = "0." + coef
		}
	} else if prec == sz {
		if self.neg {
			coef = "-0." + coef
		} else {
			coef = "0." + coef
		}
	} else {
		if self.neg {
			coef = "-" + coef[:sz-prec] + "." + coef[sz-prec:]
		} else {
			coef = coef[:sz-prec] + "." + coef[sz-prec:]
		}
	}

	return coef
}
