package dec128

import (
	"github.com/jokruger/dec128/errors"
)

// RoundHalfAwayFromZero rounds the decimal to the specified prec using Half Away from Zero method (https://en.wikipedia.org/wiki/Rounding#Rounding_half_away_from_zero).
//
// Examples:
//
//	Round(1.12345, 4) = 1.1235
//	Round(1.12335, 4) = 1.1234
//	Round(1.5, 0) = 2
//	Round(-1.5, 0) = -2
func (self Dec128) RoundHalfAwayFromZero(prec uint8) Dec128 {
	if self.err != errors.None {
		return self
	}

	if prec >= self.exp {
		return self
	}

	factor := pow10[self.exp-prec]
	half := factor / 2

	q, r, err := self.coef.QuoRem64(factor)
	if err != errors.None {
		return NaN(err)
	}

	if half <= r {
		q, err = q.Add64(1)
		if err != errors.None {
			return NaN(err)
		}
	}

	return Dec128{coef: q, exp: prec, neg: self.neg}
}

// RoundHalfTowardZero rounds the decimal to the specified prec using Half Toward Zero method (https://en.wikipedia.org/wiki/Rounding#Rounding_half_toward_zero).
//
// Examples:
//
//	Round(1.12345, 4) = 1.1234
//	Round(1.12335, 4) = 1.1233
//	Round(1.5, 0) = 1
//	Round(-1.5, 0) = -1
func (self Dec128) RoundHalfTowardZero(prec uint8) Dec128 {
	if self.err != errors.None {
		return self
	}

	if prec >= self.exp {
		return self
	}

	factor := pow10[self.exp-prec]
	half := factor / 2

	q, r, err := self.coef.QuoRem64(factor)
	if err != errors.None {
		return NaN(err)
	}

	if half < r {
		q, err = q.Add64(1)
		if err != errors.None {
			return NaN(err)
		}
	}

	return Dec128{coef: q, exp: prec, neg: self.neg}
}

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
