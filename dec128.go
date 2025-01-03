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

func (self Dec128) Equal(other Dec128) bool {
	if self.err != errors.None && other.err != errors.None {
		return true
	}
	if self.err != errors.None || other.err != errors.None {
		return false
	}
	// TODO: adjust precision for comparison
	return self.coef.Equal(other.coef) && self.prec == other.prec && self.neg == other.neg
}
