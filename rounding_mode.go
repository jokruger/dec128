package dec128

import "github.com/jokruger/dec128/state"

// RoundingMode selects how digits are discarded when a result must be shortened: by Round and RescaleRound explicitly,
// and by arithmetic implicitly whenever an exact result does not fit (see SetArithmeticRounding).
//
// The constants are named after the Round* methods they correspond to, in ALL_CAPS so that a mode is visibly distinct
// from the method of the same name. Five of them are the IEEE 754-2019 rounding attributes:
// ROUND_TOWARD_ZERO (roundTowardZero), ROUND_DOWN (roundTowardNegative), ROUND_UP (roundTowardPositive),
// ROUND_HALF_AWAY_FROM_ZERO (roundTiesToAway) and ROUND_BANK (roundTiesToEven).
type RoundingMode uint8

const (
	// ROUND_TOWARD_ZERO truncates: discarded digits are dropped and the magnitude never grows. It is the zero value and
	// the package default, and it is what every operation did before rounding modes existed.
	ROUND_TOWARD_ZERO RoundingMode = iota

	// ROUND_NAN refuses to lose digits: if any discarded digit is nonzero the result is a NaN carrying state.Inexact.
	// Use it to detect rounding rather than perform it.
	ROUND_NAN

	// ROUND_DOWN rounds toward negative infinity (floor). See RoundDown.
	ROUND_DOWN

	// ROUND_UP rounds toward positive infinity (ceiling). See RoundUp.
	ROUND_UP

	// ROUND_AWAY_FROM_ZERO rounds any nonzero remainder away from zero. See RoundAwayFromZero.
	ROUND_AWAY_FROM_ZERO

	// ROUND_HALF_TOWARD_ZERO rounds to nearest, ties toward zero. See RoundHalfTowardZero.
	ROUND_HALF_TOWARD_ZERO

	// ROUND_HALF_AWAY_FROM_ZERO rounds to nearest, ties away from zero. See RoundHalfAwayFromZero.
	ROUND_HALF_AWAY_FROM_ZERO

	// ROUND_BANK rounds to nearest, ties to even (banker's rounding). See RoundBank.
	ROUND_BANK
)

var roundingModeNames = [...]string{
	ROUND_TOWARD_ZERO:         "ROUND_TOWARD_ZERO",
	ROUND_NAN:                 "ROUND_NAN",
	ROUND_DOWN:                "ROUND_DOWN",
	ROUND_UP:                  "ROUND_UP",
	ROUND_AWAY_FROM_ZERO:      "ROUND_AWAY_FROM_ZERO",
	ROUND_HALF_TOWARD_ZERO:    "ROUND_HALF_TOWARD_ZERO",
	ROUND_HALF_AWAY_FROM_ZERO: "ROUND_HALF_AWAY_FROM_ZERO",
	ROUND_BANK:                "ROUND_BANK",
}

// IsValid reports whether m is one of the defined rounding modes.
func (m RoundingMode) IsValid() bool {
	return m <= ROUND_BANK
}

// String returns the constant's name, or "RoundingMode(n)" for an undefined value.
func (m RoundingMode) String() string {
	if m.IsValid() {
		return roundingModeNames[m]
	}
	return "RoundingMode(" + string(rune('0'+m/100%10)) + string(rune('0'+m/10%10)) + string(rune('0'+m%10)) + ")"
}

// arithmeticRounding is the mode arithmetic uses when an exact result does not fit and digits must be discarded.
// Process-global; see SetArithmeticRounding.
var arithmeticRounding = ROUND_TOWARD_ZERO

// SetArithmeticRounding sets the rounding mode arithmetic applies when an exact result does not fit in 128 bits at its
// natural scale and digits must be discarded. The default is ROUND_TOWARD_ZERO, which leaves every result that was not
// NaN before rounding modes existed bit-identical. Set ROUND_NAN to restore the previous behavior of returning NaN
// instead of rounding. This is process-global state, like SetDefaultScale: set it once during initialization, before
// any decimal is used. Calling it while other goroutines are calculating is a data race. It panics on an undefined
// mode; this is the only place rounding modes panic, and only at configuration time.
func SetArithmeticRounding(m RoundingMode) {
	if !m.IsValid() {
		panic(state.InvalidRoundingMode.Error())
	}
	arithmeticRounding = m
}

// ArithmeticRounding returns the rounding mode arithmetic applies when digits must be discarded.
func ArithmeticRounding() RoundingMode {
	return arithmeticRounding
}

// Round rounds d to the given scale using the given mode. It dispatches to the Round* method that implements the mode;
// ROUND_NAN returns a NaN carrying state.Inexact if any discarded digit is nonzero. A scale not lower than d's returns
// d unchanged, a NaN operand propagates, and an undefined mode returns a NaN carrying state.InvalidRoundingMode.
func (d Dec128) Round(scale uint8, mode RoundingMode) Dec128 {
	if d.state >= state.Error {
		return d
	}
	switch mode {
	case ROUND_TOWARD_ZERO:
		return d.RoundTowardZero(scale)
	case ROUND_NAN:
		return d.roundNaN(scale)
	case ROUND_DOWN:
		return d.RoundDown(scale)
	case ROUND_UP:
		return d.RoundUp(scale)
	case ROUND_AWAY_FROM_ZERO:
		return d.RoundAwayFromZero(scale)
	case ROUND_HALF_TOWARD_ZERO:
		return d.RoundHalfTowardZero(scale)
	case ROUND_HALF_AWAY_FROM_ZERO:
		return d.RoundHalfAwayFromZero(scale)
	case ROUND_BANK:
		return d.RoundBank(scale)
	default:
		return Dec128{state: state.InvalidRoundingMode}
	}
}

// roundNaN implements ROUND_NAN: the value at the lower scale if no nonzero digit is discarded, otherwise a NaN
// carrying state.Inexact.
func (d Dec128) roundNaN(scale uint8) Dec128 {
	if d.state >= state.Error || scale >= d.scale {
		return d
	}

	// d.scale - scale is in 1..MaxScale, so the state is always OK
	q, r, _ := d.coef.QuoRemPow10(d.scale - scale)
	if r != 0 {
		return Dec128{state: state.Inexact}
	}

	if q.IsZero() {
		return Dec128{scale: scale} // a zero is never negative
	}
	return Dec128{coef: q, scale: scale, state: d.state}
}

// RescaleRound returns d at the given scale. Raising the scale is exact, as with Rescale; lowering it rounds with the
// given mode instead of truncating. Rescale itself always truncates, so callers who want rounding say so here.
func (d Dec128) RescaleRound(scale uint8, mode RoundingMode) Dec128 {
	if d.state >= state.Error || scale >= d.scale {
		return d.Rescale(scale)
	}
	return d.Round(scale, mode)
}
