package dec128

import (
	"github.com/jokruger/dec128/errors"
	"github.com/jokruger/dec128/uint128"
)

type Dec128 struct {
	coef uint128.Uint128
	prec uint8
	err  errors.Error
	neg  bool
}

func NaN(reason errors.Error) Dec128 {
	if reason == errors.None {
		return Dec128{err: errors.NotANumber}
	}
	return Dec128{err: reason}
}

func (self Dec128) IsZero() bool {
	return self.err == errors.None && self.coef.IsZero()
}

func (self Dec128) IsNeg() bool {
	return self.neg && self.err == errors.None && !self.coef.IsZero()
}

func (self Dec128) IsPos() bool {
	return !self.neg && self.err == errors.None && !self.coef.IsZero()
}

func (self Dec128) IsNaN() bool {
	return self.err != errors.None
}

func (self Dec128) ErrorDetails() error {
	return self.err.Value()
}

// returns self encoded with the given precision
// if new precision is lower than the current precision, the result is an error
func (self Dec128) Rescale(prec uint8) Dec128 {
	if self.err != errors.None {
		return self
	}

	if self.prec == prec {
		return self
	}

	if prec > maxPrecision {
		return NaN(errors.PrecisionOutOfRange)
	}

	if self.prec > prec {
		return NaN(errors.RescaleToLessPrecision)
	}

	diff := prec - self.prec
	coef, err := self.coef.Mul64(pow10[diff])
	if err != errors.None {
		return NaN(err)
	}

	return Dec128{coef: coef, prec: prec, neg: self.neg}
}

func (self Dec128) Equal(other Dec128) bool {
	if self.err != errors.None && other.err != errors.None {
		return true
	}

	if self.err != errors.None || other.err != errors.None {
		return false
	}

	if self.neg != other.neg {
		return false
	}

	if self.prec == other.prec {
		return self.coef.Equal(other.coef)
	}

	prec := max(self.prec, other.prec)
	a := self.Rescale(prec)
	b := other.Rescale(prec)
	if !a.IsNaN() && !b.IsNaN() {
		return a.coef.Equal(b.coef)
	}

	return false
}
