package dec128

import "github.com/jokruger/dec128/state"

// RoundDown (or Floor) rounds the decimal to the specified scale using Round Down method (https://en.wikipedia.org/wiki/Rounding#Rounding_down).
//
// Examples:
//
//	RoundDown(1.236, 2) = 1.23
//	RoundDown(1.235, 2) = 1.23
//	RoundDown(1.234, 2) = 1.23
//	RoundDown(-1.234, 2) = -1.24
//	RoundDown(-1.235, 2) = -1.24
//	RoundDown(-1.236, 2) = -1.24
func (d Dec128) RoundDown(scale uint8) Dec128 {
	if d.state >= state.Error || scale >= d.scale {
		return d
	}

	// arg to QuoRem64 will be > 0
	q, r, _ := d.coef.QuoRem64(Pow10Uint64[d.scale-scale])

	// unreachable because QuoRem64 cannot be error for arg > 0
	//if s >= state.Error {
	//	return Dec128{state: s}
	//}

	if d.state == state.Neg && r > 0 {
		q, _ = q.Add64(1)
		// unreachable because Add64(1) cannot overflow at this point
		//if s >= state.Error {
		//	return Dec128{state: s}
		//}
	}

	return Dec128{coef: q, scale: scale, state: d.state}
}

// RoundUp (or Ceil) rounds the decimal to the specified scale using Round Up method (https://en.wikipedia.org/wiki/Rounding#Rounding_up).
//
// Examples:
//
//	RoundUp(1.236, 2) = 1.24
//	RoundUp(1.235, 2) = 1.24
//	RoundUp(1.234, 2) = 1.24
//	RoundUp(-1.234, 2) = -1.23
//	RoundUp(-1.235, 2) = -1.23
//	RoundUp(-1.236, 2) = -1.23
func (d Dec128) RoundUp(scale uint8) Dec128 {
	if d.state >= state.Error || scale >= d.scale {
		return d
	}

	// arg to QuoRem64 will be > 0
	q, r, _ := d.coef.QuoRem64(Pow10Uint64[d.scale-scale])

	// unreachable because QuoRem64 cannot be error for arg > 0
	//if s >= state.Error {
	//	return Dec128{state: s}
	//}

	if d.state != state.Neg && r > 0 {
		q, _ = q.Add64(1)
		// unreachable because Add64(1) cannot overflow at this point
		//if s >= state.Error {
		//	return Dec128{state: s}
		//}
	}

	return Dec128{coef: q, scale: scale, state: d.state}
}

// RoundTowardZero rounds the decimal to the specified scale using Toward Zero method (https://en.wikipedia.org/wiki/Rounding#Rounding_toward_zero).
//
// Examples:
//
//	RoundTowardZero(1.236, 2) = 1.23
//	RoundTowardZero(1.235, 2) = 1.23
//	RoundTowardZero(1.234, 2) = 1.23
//	RoundTowardZero(-1.234, 2) = -1.23
//	RoundTowardZero(-1.235, 2) = -1.23
//	RoundTowardZero(-1.236, 2) = -1.23
func (d Dec128) RoundTowardZero(scale uint8) Dec128 {
	return d.Trunc(scale)
}

// RoundAwayFromZero rounds the decimal to the specified scale using Away From Zero method (https://en.wikipedia.org/wiki/Rounding#Rounding_away_from_zero).
//
// Examples:
//
//	RoundAwayFromZero(1.236, 2) = 1.24
//	RoundAwayFromZero(1.235, 2) = 1.24
//	RoundAwayFromZero(1.234, 2) = 1.24
//	RoundAwayFromZero(-1.234, 2) = -1.24
//	RoundAwayFromZero(-1.235, 2) = -1.24
//	RoundAwayFromZero(-1.236, 2) = -1.24
func (d Dec128) RoundAwayFromZero(scale uint8) Dec128 {
	if d.state >= state.Error || scale >= d.scale {
		return d
	}

	// arg to QuoRem64 will be > 0
	q, r, _ := d.coef.QuoRem64(Pow10Uint64[d.scale-scale])

	// unreachable because QuoRem64 cannot be error for arg > 0
	//if s >= state.Error {
	//	return Dec128{state: s}
	//}

	if r > 0 {
		q, _ = q.Add64(1)
		// unreachable because Add64(1) cannot overflow at this point
		//if s >= state.Error {
		//	return Dec128{state: s}
		//}
	}

	return Dec128{coef: q, scale: scale, state: d.state}
}

// RoundHalfTowardZero rounds the decimal to the specified scale using Half Toward Zero method (https://en.wikipedia.org/wiki/Rounding#Rounding_half_toward_zero).
//
// Examples:
//
//	RoundHalfTowardZero(1.236, 2) = 1.24
//	RoundHalfTowardZero(1.235, 2) = 1.23
//	RoundHalfTowardZero(1.234, 2) = 1.23
//	RoundHalfTowardZero(-1.234, 2) = -1.23
//	RoundHalfTowardZero(-1.235, 2) = -1.23
//	RoundHalfTowardZero(-1.236, 2) = -1.24
func (d Dec128) RoundHalfTowardZero(scale uint8) Dec128 {
	if d.state >= state.Error || scale >= d.scale {
		return d
	}

	// factor will be > 0
	factor := Pow10Uint64[d.scale-scale]
	half := factor / 2

	q, r, _ := d.coef.QuoRem64(factor)

	// unreachable because QuoRem64 cannot be error for arg > 0
	//if s >= state.Error {
	//	return Dec128{state: s}
	//}

	if half < r {
		q, _ = q.Add64(1)
		// unreachable because Add64(1) cannot overflow at this point
		//if s >= state.Error {
		//	return Dec128{state: s}
		//}
	}

	return Dec128{coef: q, scale: scale, state: d.state}
}

// RoundHalfAwayFromZero rounds the decimal to the specified scale using Half Away from Zero method (https://en.wikipedia.org/wiki/Rounding#Rounding_half_away_from_zero).
//
// Examples:
//
//	RoundHalfAwayFromZero(1.236, 2) = 1.24
//	RoundHalfAwayFromZero(1.235, 2) = 1.24
//	RoundHalfAwayFromZero(1.234, 2) = 1.23
//	RoundHalfAwayFromZero(-1.234, 2) = -1.23
//	RoundHalfAwayFromZero(-1.235, 2) = -1.24
//	RoundHalfAwayFromZero(-1.236, 2) = -1.24
func (d Dec128) RoundHalfAwayFromZero(scale uint8) Dec128 {
	if d.state >= state.Error || scale >= d.scale {
		return d
	}

	// factor will be > 0
	factor := Pow10Uint64[d.scale-scale]
	half := factor / 2

	q, r, _ := d.coef.QuoRem64(factor)

	// unreachable because QuoRem64 cannot be error for arg > 0
	//if s >= state.Error {
	//	return Dec128{state: s}
	//}

	if half <= r {
		q, _ = q.Add64(1)
		// unreachable because Add64(1) cannot overflow at this point
		//if s >= state.Error {
		//	return Dec128{state: s}
		//}
	}

	return Dec128{coef: q, scale: scale, state: d.state}
}

// RoundBank uses half up to even (banker's rounding) to round the decimal to the specified scale.
//
// Examples:
//
//	RoundBank(2.121, 2) = 2.12 ; rounded down
//	RoundBank(2.125, 2) = 2.12 ; rounded down, rounding digit is an even number
//	RoundBank(2.135, 2) = 2.14 ; rounded up, rounding digit is an odd number
//	RoundBank(2.1351, 2) = 2.14; rounded up
//	RoundBank(2.127, 2) = 2.13 ; rounded up
func (d Dec128) RoundBank(scale uint8) Dec128 {
	if d.state >= state.Error || scale >= d.scale {
		return d
	}

	// factor will be > 0
	factor := Pow10Uint64[d.scale-scale]
	half := factor / 2

	q, r, _ := d.coef.QuoRem64(factor)

	// unreachable because QuoRem64 cannot be error for arg > 0
	//if s >= state.Error {
	//	return Dec128{state: s}
	//}

	if half < r || (half == r && q.Lo%2 == 1) {
		q, _ = q.Add64(1)
		// unreachable because Add64(1) cannot overflow at this point
		//if s >= state.Error {
		//	return Dec128{state: s}
		//}
	}

	return Dec128{coef: q, scale: scale, state: d.state}
}

// Trunc returns d after truncating the decimal to the specified scale.
//
// Examples:
//
//	Trunc(1.12345, 4) = 1.1234
//	Trunc(1.12335, 4) = 1.1233
func (d Dec128) Trunc(scale uint8) Dec128 {
	if d.state >= state.Error || scale >= d.scale {
		return d
	}

	// arg to QuoRem will be > 0
	q, _, _ := d.coef.QuoRem64(Pow10Uint64[d.scale-scale])

	// unreachable because QuoRem64 cannot be error for arg > 0
	//if s >= state.Error {
	//	return Dec128{state: s}
	//}

	return Dec128{coef: q, scale: scale, state: d.state}
}
