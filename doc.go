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
// Canonical to normalize a key.
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
// keeps all its places. AddRound, SubRound, MulRound, DivRound and SqrtRound take the
// scale and the rounding mode per call and return exactly that scale, computed in one
// step with an exact rounding decision:
//
//	rate := annual.DivRound(daysInYear, 12, ROUND_HALF_AWAY_FROM_ZERO)
//	fee := amount.MulRound(rate, 2, ROUND_BANK)
//
// MulAddRound is the fused form of the pair: d*b + c with the product held exactly and
// the sum rounded once, which is what a chain of multiply-accumulate wants. MulDivRound is
// its multiplicative counterpart, d*b/c with the numerator and the divisor kept apart until
// one rounding: that is the only correct way to scale by a rational whose decimal expansion
// does not terminate, because 1/3 and 31/365 cannot be written as a decimal at all and so
// cannot be converted to one first. MulDivRoundInt64 takes that rational as a pair of
// integers, which is how a day count over a year basis arrives.
//
// MulPercent and MulPercentRound are d*f/100 with the division done as a move of the decimal
// point inside the same reduction, so a percentage costs one rounding and not two; MulScaled
// and MulScaledRound are the general form, d*f*10^k, which is a per-mille at k = -3 and a
// basis point at k = -4. MulPercent follows the scale rule as Mul does - the exact product of
// an amount and a rate carries two places more than the two of them together, so a money-sized
// calculation is exact and the rounding to the presented scale stays the caller's, at the end.
//
// Accumulator is MulAddRound over an unbounded number of terms: Add and AddMul are exact, Total is
// the only rounding, and the total does not depend on the order the terms arrived in, so
// "the parts add up to the whole" is an exact assertion rather than an epsilon check.
// Mean divides that exact total by the number of terms with the same single rounding, and
// Reset empties the accumulator for the next batch. Sum and SumSlice are the same idea
// with the scale rule applied at the end instead of a scale per call; they keep a
// narrower register, because every term of a sum is a term rather than a product.
// SumRound, SumSliceRound and AvgRound are their global-free spellings, over that same
// narrow register; AvgRound divides the exact total by the count under a single rounding,
// where Avg rounds the total and then rounds the quotient.
//
// PowIntRound does the same for a power: it keeps the running product at up to 57
// decimal places, half as many again as a coefficient holds, and rounds once at the end.
// PowInt64 shares that core and differs only in taking the scale from the scale rule and
// the mode from SetArithmeticRounding rather than per call.
//
// The Round* methods round to at most the given number of places and leave a shorter
// value unchanged; RescaleRound(places, mode) is the "exactly n places" operation, the
// equivalent of PostgreSQL's round(x, n). Rescale pads or truncates without rounding.
//
// Two grids are not a scale of the type. RoundToPlaces takes a negative number of places,
// for the currencies with no minor unit and the disclosure rounding several jurisdictions
// require, and RoundToMultiple rounds to a multiple of any positive value, which is Swiss
// and Swedish cash rounding to 0.05 and the "up to the next whole ten" of retail lending.
// ScaleByPow10 moves the decimal point exactly, so a percentage or a basis point becomes a
// fraction without a division. RoundToSignificant rounds to a number of significant digits
// rather than to a position, which is how a rate is quoted and how IEEE 754 decimal
// arithmetic works.
//
// # Splitting an amount
//
// Allocate divides a value into shares proportional to a list of ratios, and Split into n
// equal ones, by the largest-remainder method: every share gets the whole quanta its exact
// proportion is worth, and the quanta the truncations leave over go one each to the
// largest fractions, ties to the lowest index. The shares sum to the original value
// exactly, so a reconciliation is an equality and not an epsilon:
//
//	shares, ok := fee.Allocate([]Dec128{partyA, partyB, partyC}, 2)
//
// AllocateResidual and SplitResidual are the other convention, the one an amortization
// schedule and a syndicated facility use: every share but one is its proportion rounded
// the agreed way, and the share at an index the caller names takes what is left. The
// shares still sum to the whole exactly, but only the others are within a quantum of
// their proportions - the named one absorbs all the rounding, and for an amount of a few
// quanta it can even come out on the other side of zero.
//
// AppendAllocate, AppendSplit and their residual forms append to a slice the caller keeps,
// and a split of up to 32 ways then costs no allocation.
//
// # Roots and the shape of a value
//
// SqrtRound is the square root and NthRootRound the n-th, which is the inverse of
// PowIntRound and the primitive of rate conversion: the monthly factor behind an annual
// one is factor.NthRootRound(12, scale, mode). PowRational is the general form, d^(p/q)
// for any reduced fraction, which is the factor of a compound change spread over a
// fractional number of periods; NthRootRound is its p == 1 case and PowIntRound its q == 1
// case, though PowIntRound keeps its own guarded-product core and stays fixed-width.
//
// Both roots are correctly rounded, which is stronger than PowIntRound's faithful
// rounding, because the decision is an exact integer comparison rather than a guarded
// approximation: d^(p/q) is the q-th root of d^p and d^p is an exact rational. They are
// also the two operations here that allocate - see "Operations that allocate" - because
// that comparison is wider than any register the package keeps. An exponent above 16384
// in either half, after the fraction is reduced, is refused rather than computed; see
// maxRootDegree for the measurements behind the bound.
//
// # Exp, Ln and the rest of the completeness set
//
// Exp, Ln, Ln1p, Expm1 and Pow complete the type. Pow dispatches on its exponent: an integer goes
// to PowIntRound and allocates nothing, a fraction that reduces to small halves goes to
// PowRational and is correctly rounded, and anything else is exp(e*ln d). So x.Pow(Half, ...) is
// bit-identical to x.SqrtRound(...) without the caller arranging it.
//
// Ln1p and Expm1 are the usual entry points for one plus something small, and they are here for
// that reason, but the cancellation they exist to prevent in a binary format does not arise on
// this type: a Dec128 carries 39 significant digits, so 1+x is exact for a small x and
// Ln(One.Add(x)) loses nothing. They are the right thing to write and they stay exact at the top
// of the range, where 1+x can exceed a coefficient; they do not recover digits Ln would have lost,
// and the suite pins the two to agree. Log10 and Log2 complete the set. See "Exact, and not exact"
// for the guarantee they carry.
//
// SignificantDigits, IntegerDigits and FitsNumeric report the shape of a value for the
// column it has to be stored in, so that a value too wide for a NUMERIC(p, s) is caught
// where it is computed rather than by the database halfway through a batch; and
// RescaleRoundInexact makes it fit, saying whether that cost anything. IsInteger, IntFrac
// and Clamp are the small value operations a schedule and a limit check need.
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
// # The global-free subset
//
// Three of the five process-global settings can change the value an operation returns:
// SetDefaultScale, SetArithmeticRounding and SetLossPolicy. Code that must produce the
// same bytes in every process, whatever some other package passed to those functions at
// init time, has to keep to the operations that never read them.
//
// These operations do read them, and are therefore outside the subset:
//
//	Add, Sub, Mul, MulScaled, MulPercent and their Int forms, when the exact result does not fit
//	Div, Inv, Sqrt, PowInt, PowInt64 and their Int forms, always
//	Sum, SumSlice, Avg, Prod, ProdSlice
//	EncodeIEEE, when the coefficient needs more than 34 digits
//
// Each has a twin that does not: AddRound, SubRound, MulRound, MulScaledRound and
// MulPercentRound for the first line, DivRound, InvRound, SqrtRound and PowIntRound for the
// second, SumRound, SumSliceRound, AvgRound, ProdRound and ProdSliceRound for the third, and
// EncodeIEEERound for the fourth.
//
// Everything else is global-free. In particular the whole *Round family, which takes the
// scale and the rounding mode per call and is the deterministic spelling of the four
// basic operations:
//
//	AddRound, SubRound, MulRound, MulAddRound, MulDivRound, MulDivRoundInt64, DivRound,
//	DivRoundInexact, AddQuoRound, SqrtRound, PowIntRound, InvRound
//	MulScaledRound, MulPercentRound
//	NthRootRound, PowRational
//	Exp, Ln, Ln1p, Expm1, Log10, Log2, Pow
//	SumRound, SumSliceRound, AvgRound, ProdRound, ProdSliceRound
//	Accumulator and its methods, Total and Mean included
//	QuoRem, Mod, RemainderNear and their Int forms, which are exact
//	Round and the Round* methods, RoundToPlaces, RoundToMultiple, RoundToSignificant,
//	Trunc, Rescale, RescaleRound, RescaleRoundInexact
//	ScaleByPow10, Allocate, Split and their Append and Residual forms
//	Abs, Neg, CopySign, Logb, Compare, CmpTotal, Equal, the comparison predicates,
//	Canonical, Clamp
//	IsInteger, IntFrac, SignificantDigits, IntegerDigits, FitsNumeric
//	Rat, BigInt and FromRat, and Decompose and Compose
//	the From* constructors, the String and Append forms, Format, and the Encode/Decode
//	codecs other than EncodeIEEE, whose per-call form is EncodeIEEERound
//
// The list is a maintained API contract, and TestGlobalFreeSubset holds it to that: it
// runs every method named here under a matrix of the three settings and requires the
// results to be identical.
//
// # Operations that allocate
//
// Everything here works in fixed registers and allocates nothing, with two exceptions, whose
// exact intermediate is wider than any register the package keeps:
//
//	NthRootRound
//	PowRational
//	Exp, Ln, Ln1p, Expm1
//	Log10 and Log2, unless the argument is an exact power of the base, which is answered
//	from the coefficient and allocates nothing
//	Pow, unless the exponent is an integer, where it is PowIntRound and allocates nothing
//	Prod, ProdRound and their Slice forms
//	Rat, BigInt and FromRat, which hand back or take a math/big value
//
// They have no fixed-width form; they are not slower versions of something cheaper. A root's
// rounding decision needs a comparison 39*n digits wide and a rational power's 39*|p| + MaxScale*q,
// and a transcendental has to carry guard digits through a series. The list is a maintained API
// contract like the one above, and TestAllocationGates holds every operation not on it to zero
// allocations while TestAllocatingSetIsNotVacuous checks that these really do allocate, so an
// operation that later gains a fixed-width form is taken off the list rather than left on it.
//
// # Exact, and not exact
//
// Every operation in this package is exact, in the sense that it computes an exact intermediate and
// makes a single rounding decision on it, with seven exceptions: Exp, Ln, Ln1p, Expm1, Log10, Log2
// and Pow. The value of a transcendental is irrational for all but a handful of arguments, so there
// is no exact intermediate to decide on. Those seven are documented as faithfully rounded - at most
// one unit in the last place from the correctly rounded value - and the package documents what was
// measured rather than what is hoped: against a correctly-rounded reference the largest error
// observed over the public methods was zero units in the last place, against a second and
// independent oracle over random arguments it was zero again, and over the general arm of Pow in
// isolation it was one. See the head of transcendental.go for the whole measurement.
//
// Log10 and Log2 have an exact case inside the inexact one: an argument that is an exact power of
// the base is recognized from the coefficient and answered exactly, because a series converging on
// an integer from below would be truncated away from it by a directed rounding mode. The General
// Decimal Arithmetic specification requires that of log10; Log2 does it too, so that the two do not
// disagree about whether an exact answer is worth having.
//
// Prod and its forms are exact and not approximate, but they are not fixed-width: the exact product
// of n coefficients has the sum of their digit counts, so the intermediate is a math/big value and
// the single rounding is made on it.
//
// NthRootRound and PowRational, though they allocate, are correctly rounded: their decision is an
// exact integer comparison. Allocating and approximating are separate properties and this package
// keeps them separate. PowIntRound is the other way round again - fixed-width, allocation-free, and
// faithfully rounded rather than correctly rounded.
//
// One rule governs both, and it is what keeps their results identical on every architecture:
// a float64 may choose a shortcut, never a returned value. nthRootGuess seeds the iteration from
// a float64 logarithm and an exact integer comparison decides the answer; PowRational estimates
// the magnitude of its result in float64 and acts on it only a full decade clear of either
// boundary. math/big is deterministic where binary floating point is not, which is why the
// intermediate is big.Int and never big.Float.
//
// Determinism has one requirement beyond this list: do not use FromFloat64 or
// InexactFloat64 inside a calculation. They are legitimate at a system boundary, but
// binary floating point is where architecture-dependent results come from.
//
// The two remaining globals affect text and SQL rather than arithmetic. Format and the
// Marshal and Value methods read SetTrimOutput, which chooses between the two forms;
// String and StringFixed are each one of those forms and read nothing, so StringFixed is
// the deterministic text output. Scan(nil) and UnmarshalJSON("null") read SetNullValue.
//
// # Text and interchange
//
// String prints the value with trailing zeros removed; StringFixed keeps the scale.
// Format implements fmt.Formatter, so %v and %s are String, %f is StringFixed and %.2f is
// the value at two places rounded half to even, as Go's own %f rounds a float64.
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
