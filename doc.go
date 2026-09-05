// Package dec128 provides a 128-bit fixed-point decimal type for financial and banking
// arithmetic.
//
// A Dec128 holds a 128-bit unsigned coefficient, a scale (the number of digits after
// the decimal point, 0 to MaxScale) and a byte carrying both the sign and the error
// state. Values are immutable: every operation returns a new instance.
//
// # Failures are values, not returns
//
// Arithmetic never panics and never returns an error. A failed operation returns a NaN
// carrying the reason, and NaN propagates through everything downstream, so a whole
// calculation can be written as one expression and checked once at the end:
//
//	total := principal.Mul(rate).Add(fee).RoundBank(2)
//	if err := total.ErrorDetails(); err != nil {
//		return err
//	}
//
// ErrorDetails is the terminal check for a chain and returns a plain error; IsNaN is
// the boolean form. Nothing forces you to call either, so unlike an ignored error there
// is no compiler or linter backstop - the check is yours to remember.
//
// Two consequences are easy to miss:
//
//   - IsZero, IsNegative and IsPositive all return false for a NaN, so a guard written
//     as "if !d.IsZero()" does not catch a failed calculation. Test IsNaN first when the
//     difference matters.
//   - MarshalJSON encodes a NaN as the string "NaN". A consumer that accepts strings
//     will take it without complaint, so validate before marshalling if that matters.
//     Value is safer by construction: a numeric column rejects it.
//
// # Scale
//
// Scale is preserved rather than normalised, so 1.5 and 1.50 are different
// representations that compare equal. Add and Sub take the larger scale of the two
// operands, Mul adds them, and Div uses the package default as a floor (see
// SetDefaultScale). QuoRem does not consult it: the quotient is always an integer.
//
// Scale grows to keep precision and is capped at MaxScale. An operation whose exact
// result would need more digits than that returns NaN rather than rounding silently, so
// round explicitly once a calculation is finished:
//
//	rate := annual.DivAtScale(daysInYear, 12)
//	amount := principal.Mul(rate).RoundBank(2)
//
// Because the cap is a hard one, dividing at a scale close to MaxScale leaves no room
// for a later multiplication. DivAtScale and SqrtAtScale take the scale per call and do
// not depend on the global at all.
//
// # Configuration
//
// SetDefaultScale and SetNullValue are process-global and are plain variables: set them
// once during initialisation, before any decimal is used. Changing them while other
// goroutines are calculating is a data race.
package dec128
