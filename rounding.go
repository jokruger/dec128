package dec128

import (
	"github.com/jokruger/dec128/errors"
)

// Trunc returns self after truncating the decimal to the specified precision.
//
// Examples:
//
//	Trunc(1.12345, 4) = 1.1234
//	Trunc(1.12335, 4) = 1.1233
func (self Dec128) Trunc(prec uint8) Dec128 {
	if self.err != errors.None {
		return self
	}

	if prec >= self.exp {
		return self
	}

	q, _, err := self.coef.QuoRem64(pow10[self.exp-prec])
	if err != errors.None {
		return NaN(err)
	}

	return Dec128{coef: q, exp: prec, neg: self.neg}
}
