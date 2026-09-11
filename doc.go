// Package dec128 provides a fast, zero-dependency 128-bit fixed-point decimal type for
// money and financial arithmetic, with SQL NUMERIC semantics and no heap allocation.
//
// dec128 is named for its coefficient: a 128-bit unsigned integer with a decimal scale,
// the same fixed-point model as Apache Arrow's and Parquet's Decimal128. It is not IEEE
// 754 decimal128, which is a floating-point format; that is supported as an interchange
// encoding only (see EncodeIEEE).
//
// # The contract
//
// dec128 implements the exact-numeric semantics of SQL NUMERIC (ISO/IEC 9075) as
// realised by PostgreSQL, SQL Server and .NET System.Decimal: addition, subtraction and
// multiplication are exact, the scale is preserved, and when a result would not fit the
// scale is reduced using a chosen IEEE 754-2019 rounding mode to protect the integer
// part. Failures are quiet NaN values, never errors or panics. PostgreSQL's numeric is
// the reference implementation the test suite compares against, digit for digit.
//
// A Dec128 carries 38 significant decimal digits, and 39 while the coefficient is at or
// below 340282366920938463463374607431768211455, shared freely between the integer and
// the fractional part, with at most MaxScale (19) of them after the decimal point. That
// one budget is the whole size contract, and everything below follows from it:
//
//   - the largest value is +/-3.402...x10^38 and the smallest non-zero value is
//     +/-1x10^-19, so 0.00000000000000000001 is not a small number but an unrepresentable
//     one, and FromString rejects it;
//   - precision after the decimal point shrinks as magnitude grows. All 19 places are
//     available up to 34028236692093846346.3374607431768211455, and one place is lost per
//     decade above that until none is left near 10^38;
//   - a result that needs more digits than remain is rounded to fit. That is the design
//     and not a failure: NaN(Overflow) is reserved for a result whose integer part alone
//     will not fit. See "Safe operating range" for where the rounding starts.
//
// Div and Sqrt are bounded by the scale cap rather than by the coefficient: they compute
// to DefaultScale places, so 1/3 has 19 significant digits and not 38.
//
// # Safe operating range
//
// If every operand has at most 9 decimal places and every value, intermediate products
// included, stays below 10^20, then Add, Sub and Mul are exact and nothing is ever
// rounded. TestSafeZoneClaim holds the library to that promise.
//
// The bound on the scale is the one that binds first, and it has nothing to do with
// magnitude: Mul adds the scales, so two operands at 9 places give an 18-place product
// that still fits under MaxScale, while two at 10 places give a 20-place product that
// never does. The same cliff makes small products collapse:
//
//	0.000000001  * 0.000000001   // 0.000000000000000001, exact
//	0.0000000001 * 0.0000000001  // 0, the exact result 1e-20 is not representable
//
// The top of the range is unreachable for money - 3.4x10^36 at two decimal places is
// many orders of magnitude beyond any real balance - but the bottom is not, and products
// of rates or per-unit factors around 10^-10 reach it. SetLossPolicy(LossNaNOnUnderflow)
// turns that silent zero into a NaN. MaxAtScale and QuantumAtScale report the two ends of
// the range at any scale.
//
// A Dec128 holds a 128-bit unsigned coefficient, a scale (the number of digits after
// the decimal point, 0 to MaxScale) and a byte carrying both the sign and the error
// state. Values are immutable: every operation returns a new instance. Do not compare
// values with ==: 1.5 and 1.50 are different representations of one number, and Equal
// and Compare are the comparison operations.
//
// A Dec128 is comparable and can be a map key, but == is not a numeric comparison: 1.5
// and 1.50 differ in scale and are different keys. Use Equal or Compare for values and
// Canonical to normalise a key.
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
// is no compiler or linter backstop - the check is yours to remember. IsZero,
// IsNegative and IsPositive all return false for a NaN, so a guard written as
// "if !d.IsZero()" does not catch a failed calculation.
//
// # Scale
//
// The exact result of Add and Sub has the larger of the two scales, and that of Mul
// the sum of the scales; 1.50 - 1.50 is 0.00. When the exact result does not fit in 128
// bits, or needs more than MaxScale places, the scale is reduced to the largest at
// which the integer part fits and the discarded digits are rounded with the mode set by
// SetArithmeticRounding (truncation by default). NaN(Overflow) is returned only when the
// integer part does not fit even at scale 0, and SetLossPolicy decides whether a
// discarded digit is acceptable at all.
//
// Div and Sqrt compute at the default scale (SetDefaultScale, MaxScale by default) and
// then apply the ideal scale of an exact result: 1/2 is 0.5, 1.00/2 is 0.50, and 1/3
// keeps all its places. MulRound, DivRound and SqrtRound take the scale and the rounding
// mode per call and return exactly that scale, computed in one step with an exact
// rounding decision:
//
//	rate := annual.DivRound(daysInYear, 12, ROUND_HALF_AWAY_FROM_ZERO)
//	fee := amount.MulRound(rate, 2, ROUND_BANK)
//
// The Round* methods round to at most the given number of places and leave a shorter
// value unchanged; RescaleRound(places, mode) is the "exactly n places" operation, the
// equivalent of PostgreSQL's round(x, n). Rescale pads or truncates without rounding.
//
// # Rounding modes
//
// RoundingMode names the eight ways to discard digits: ROUND_TOWARD_ZERO (the default),
// ROUND_DOWN, ROUND_UP, ROUND_AWAY_FROM_ZERO, ROUND_HALF_TOWARD_ZERO,
// ROUND_HALF_AWAY_FROM_ZERO, ROUND_BANK, and ROUND_NAN, which refuses to lose digits and
// returns NaN(Inexact) instead.
//
// # Loss policy
//
// A rounding mode chooses a direction; whether arithmetic may discard a digit at all is
// the separate setting LossPolicy, because losing digits is not one condition but three.
// An integer part that does not fit at scale 0 is an overflow and is always
// NaN(Overflow). A result that keeps its significant digits and drops only the tail is
// inexact, which 1/3 and Sqrt(2) are at every scale. A result that rounds to zero from
// non-zero operands has lost every significant digit it had, which is an underflow, and
// is the one case where a plausible-looking value hides a total loss of information.
//
//	LossRound           // default: round, as every version before v1.2 did
//	LossNaNOnUnderflow  // NaN(Underflow) only when a result rounds to zero; 1/3 still works
//	LossNaNOnInexact    // NaN(Inexact) on any discarded nonzero digit, 1/3 included
//
// LossNaNOnUnderflow is the recommended setting for money: a non-zero amount never
// silently becomes zero, and an inexact division still returns a value.
//
// SetArithmeticRounding(ROUND_NAN) is the deprecated spelling of
// SetLossPolicy(LossNaNOnInexact) and still behaves exactly as it did in v1.1, including
// its one difference from v1.0.20: it applies to Div and Sqrt too, so 1/3 and Sqrt(2)
// become NaN(Inexact) where v1.0.20 truncated them at the default scale. A program that
// sets both must call SetLossPolicy second.
//
// # Text and interchange
//
// String prints the value with trailing zeros removed; StringFixed keeps the scale.
// Value, MarshalJSON and MarshalText emit the fixed form, so the scale survives a round
// trip through a database or a JSON consumer; SetTrimOutput(true) switches them to the
// trimmed form of earlier versions. FromString parses both notations and the special
// values PostgreSQL emits (NaN, Infinity, -Infinity); Scan and the Unmarshal methods
// return an error only for input that is not a number at all.
//
// Three binary interchange formats are supported besides the package's own
// (EncodeBinary): PostgreSQL's numeric wire format (EncodePgNumeric, DecodePgNumeric),
// IEEE 754 decimal128 in the BID encoding (EncodeIEEE, DecodeIEEE) and the int128
// coefficient of Apache Arrow and Parquet (EncodeInt128, DecodeInt128). A value that the
// type cannot hold decodes to a NaN carrying the reason rather than being rounded.
//
// github.com/jokruger/pgxdec128 builds a pgx v5 codec on the PostgreSQL pair, mapping
// numeric and numeric[] columns to Dec128 on the binary wire without allocating.
//
// # Configuration
//
// SetDefaultScale, SetArithmeticRounding, SetLossPolicy, SetTrimOutput and SetNullValue
// are process-global and are plain variables: set them once during initialization,
// before any decimal is used. Changing them while other goroutines are calculating is a
// data race. SetArithmeticRounding also writes the loss policy, so a program that sets
// both must call SetLossPolicy second.
package dec128
