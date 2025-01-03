package dec128

import (
	"strings"

	"github.com/jokruger/dec128/uint128"
)

// Returns coef, prec, error
func (self Dec128) Uint64() (uint64, uint8, error) {
	if self.nan {
		return 0, 0, ErrNaN
	}

	if self.neg {
		return 0, 0, ErrNegative
	}

	i, err := self.coef.Uint64()
	if err != nil {
		return 0, 0, err
	}

	return i, self.prec, nil
}

// Returns coef, prec, error
func (self Dec128) Uint128() (uint128.Uint128, uint8, error) {
	if self.nan {
		return uint128.Zero, 0, ErrNaN
	}

	if self.neg {
		return uint128.Zero, 0, ErrNegative
	}

	return self.coef, self.prec, nil
}

func (self Dec128) String() string {
	if self.nan {
		return NaNStr
	}

	if self.IsZero() {
		return ZeroStr
	}

	coef := self.coef.String()
	prec := int(self.prec)

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
