package dec128

import (
	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

var (
	// precalculated StringFixed values for 0 Dec128 in all possible prec
	zeroStrs = [...]string{
		"0",                     // 10^0
		"0.0",                   // 10^1
		"0.00",                  // 10^2
		"0.000",                 // 10^3
		"0.0000",                // 10^4
		"0.00000",               // 10^5
		"0.000000",              // 10^6
		"0.0000000",             // 10^7
		"0.00000000",            // 10^8
		"0.000000000",           // 10^9
		"0.0000000000",          // 10^10
		"0.00000000000",         // 10^11
		"0.000000000000",        // 10^12
		"0.0000000000000",       // 10^13
		"0.00000000000000",      // 10^14
		"0.000000000000000",     // 10^15
		"0.0000000000000000",    // 10^16
		"0.00000000000000000",   // 10^17
		"0.000000000000000000",  // 10^18
		"0.0000000000000000000", // 10^19
	}

	// precalculated array of zero characters
	zeros = [...]byte{'0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0'}
)

// called only when both are not NaN
func (self Dec128) tryAdd(other Dec128) (Dec128, bool) {
	prec := max(self.exp, other.exp)

	a := self.Rescale(prec)
	if a.state >= state.Error {
		return a, false
	}

	b := other.Rescale(prec)
	if b.state >= state.Error {
		return b, false
	}

	if a.state == b.state {
		coef, s := a.coef.Add(b.coef)
		if s >= state.Error {
			return Dec128{state: s}, false
		}
		return Dec128{coef: coef, exp: prec, state: a.state}, true
	}

	switch a.coef.Compare(b.coef) {
	case 1:
		coef, s := a.coef.Sub(b.coef)
		if s >= state.Error {
			return Dec128{state: s}, false
		}
		return Dec128{coef: coef, exp: prec, state: a.state}, true
	case 0:
		return Zero, true
	default:
		coef, s := b.coef.Sub(a.coef)
		if s >= state.Error {
			return Dec128{state: s}, false
		}
		return Dec128{coef: coef, exp: prec, state: b.state}, true
	}
}

// called only when both are not NaN
func (self Dec128) trySub(other Dec128) (Dec128, bool) {
	prec := max(self.exp, other.exp)

	a := self.Rescale(prec)
	if a.IsNaN() {
		return a, false
	}

	b := other.Rescale(prec)
	if b.IsNaN() {
		return b, false
	}

	if a.state != b.state {
		coef, s := a.coef.Add(b.coef)
		if s >= state.Error {
			return Dec128{state: s}, false
		}
		return Dec128{coef: coef, exp: prec, state: a.state}, true
	}

	switch a.coef.Compare(b.coef) {
	case 1:
		coef, s := a.coef.Sub(b.coef)
		if s >= state.Error {
			return Dec128{state: s}, false
		}
		return Dec128{coef: coef, exp: prec, state: a.state}, true
	case 0:
		return Zero, true
	default:
		coef, s := b.coef.Sub(a.coef)
		if s >= state.Error {
			return Dec128{state: s}, false
		}
		if a.state == state.Neg {
			return Dec128{coef: coef, exp: prec}, true
		}
		return Dec128{coef: coef, exp: prec, state: state.Neg}, true
	}
}

// called only when both are not NaN
func (self Dec128) tryMul(other Dec128) (Dec128, bool) {
	var st state.State
	if self.state != other.state {
		st = state.Neg
	}

	prec := self.exp + other.exp
	rcoef, rcarry := self.coef.MulCarry(other.coef)

	if rcarry.IsZero() {
		r := Dec128{coef: rcoef, exp: prec, state: st}
		if prec <= MaxPrecision {
			return r, true
		}
		r = r.Canonical()
		return r, r.exp <= MaxPrecision
	}

	i := prec
	for {
		if i == 0 {
			return Dec128{state: state.Overflow}, false
		}
		q, r, s := uint128.QuoRem256By128(rcoef, rcarry, Pow10Uint128[i])
		if s < state.Error && r.IsZero() {
			return Dec128{coef: q, exp: prec - i, state: st}, true
		}
		if s >= state.Error {
			return Dec128{state: s}, false
		}
		i--
		if prec-i > MaxPrecision {
			return Dec128{state: state.Overflow}, false
		}
	}
}

// called only when both are not NaN
func (self Dec128) tryDiv(other Dec128) (Dec128, bool) {
	factor := other.exp
	prec := self.exp
	if prec < defaultPrecision {
		factor = factor + defaultPrecision - prec
		prec = defaultPrecision
	}
	u, c := self.coef.MulCarry(Pow10Uint128[factor])
	q, _, s := uint128.QuoRem256By128(u, c, other.coef)
	if s >= state.Error {
		return Dec128{state: s}, false
	}

	if self.state == other.state {
		return Dec128{coef: q, exp: prec}, true
	}

	return Dec128{coef: q, exp: prec, state: state.Neg}, true
}

// called only when both are not NaN
func (self Dec128) tryQuoRem(other Dec128) (Dec128, Dec128, bool) {
	var factor uint8
	var u uint128.Uint128
	var c uint128.Uint128
	var d uint128.Uint128
	var s state.State

	if self.exp == other.exp {
		factor = self.exp
		u = self.coef
		d = other.coef
	} else {
		factor = max(self.exp, other.exp)
		u, c = self.coef.MulCarry(Pow10Uint128[factor-self.exp])
		d, s = other.coef.Mul(Pow10Uint128[factor-other.exp])
		if s >= state.Error {
			return Dec128{state: s}, Dec128{state: s}, false
		}
	}

	q1, r1, s := uint128.QuoRem256By128(u, c, d)
	if s >= state.Error {
		return Dec128{state: s}, Dec128{state: s}, false
	}

	if self.state == other.state {
		return Dec128{coef: q1, exp: 0}, Dec128{coef: r1, exp: factor, state: self.state}, true
	}

	return Dec128{coef: q1, exp: 0, state: state.Neg}, Dec128{coef: r1, exp: factor, state: self.state}, true
}

// appendString appends the string representation of the decimal to sb. Returns the new slice and whether the decimal contains a decimal point.
// called only when self is not NaN
func (self Dec128) appendString(sb []byte) ([]byte, bool) {
	buf := [uint128.MaxStrLen]byte{}
	coef := self.coef.StringToBuf(buf[:])

	if self.state == state.Neg {
		sb = append(sb, '-')
	}

	prec := int(self.exp)
	if prec == 0 {
		return append(sb, coef...), false
	}

	sz := len(coef)
	if prec > sz {
		sb = append(sb, '0', '.')
		sb = append(sb, zeros[:prec-sz]...)
		sb = append(sb, coef...)
	} else if prec == sz {
		sb = append(sb, '0', '.')
		sb = append(sb, coef...)
	} else {
		sb = append(sb, coef[:sz-prec]...)
		sb = append(sb, '.')
		sb = append(sb, coef[sz-prec:]...)
	}

	return sb, true
}

func trimTrailingZeros(sb []byte) []byte {
	i := len(sb)

	for i > 0 && sb[i-1] == '0' {
		i--
	}

	if i > 0 && sb[i-1] == '.' {
		i--
	}

	return sb[:i]
}

// called only when self is not NaN
func (self Dec128) trySqrt() (Dec128, bool) {
	prec := defaultPrecision
	prec2 := prec * 2
	d := self

	if d.exp > prec2 {
		// scale down to prec2
		coef, s := d.coef.Div(Pow10Uint128[d.exp-prec2])
		if s >= state.Error {
			return Dec128{state: s}, false
		}
		d = Dec128{coef: coef, exp: prec2, state: d.state}
	}

	coef, carry := d.coef.MulCarry(Pow10Uint128[prec2-d.exp])
	if carry.Hi > 0 {
		return Dec128{state: state.Overflow}, false
	}

	// 0 <= coef.bitLen() < 256, so it's safe to convert to uint
	bitLen := uint(coef.BitLen() + carry.BitLen())

	// initial guess = 2^((bitLen + 1) / 2) ≥ √coef
	x := uint128.One.Lsh((bitLen + 1) / 2)

	// Newton-Raphson method
	for {
		// calculate x1 = (x + coef/x) / 2
		y, _, s := uint128.QuoRem256By128(coef, carry, x)
		if s >= state.Error {
			return Dec128{state: s}, false
		}

		x1, s := x.Add(y)
		if s >= state.Error {
			return Dec128{state: s}, false
		}

		x1 = x1.Rsh(1)
		if x1.Compare(x) == 0 {
			break
		}

		x = x1
	}

	return Dec128{coef: x, exp: prec}, true
}
