package dec128

import (
	"github.com/jokruger/dec128/errors"
)

// Ceil returns the smallest integer value greater than or equal to 'self'.
func (self Dec128) Ceil() Dec128 {
	if self.err != errors.None {
		return self
	}

	if self.exp == 0 {
		return self
	}

	q, r, err := self.coef.QuoRem64(pow10[self.exp])
	if err != errors.None {
		return NaN(err)
	}

	if !self.neg && r != 0 {
		q, err = q.Add64(1)
		if err != errors.None {
			return NaN(err)
		}
	}

	return Dec128{coef: q, exp: 0, neg: self.neg}
}

// Floor returns the largest integer value less than or equal to 'self'.
func (self Dec128) Floor() Dec128 {
	if self.err != errors.None {
		return self
	}

	if self.exp == 0 {
		return self
	}

	q, r, err := self.coef.QuoRem64(pow10[self.exp])
	if err != errors.None {
		return NaN(err)
	}

	if self.neg && r != 0 {
		q, err = q.Add64(1)
		if err != errors.None {
			return NaN(err)
		}
	}

	return Dec128{coef: q, exp: 0, neg: self.neg}
}

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
