package dec128

import "github.com/jokruger/dec128/uint128"

type Dec128 struct {
	coef uint128.Uint128
	prec uint8
	neg  bool
	nan  bool
}

func (self Dec128) IsZero() bool {
	return self.coef.IsZero()
}

func (self Dec128) IsNeg() bool {
	return self.neg && !self.coef.IsZero()
}

func (self Dec128) IsPos() bool {
	return !self.neg && !self.coef.IsZero()
}

func (self Dec128) IsNaN() bool {
	return self.nan
}
