package dec128

import "github.com/jokruger/dec128/state"

// LossPolicy selects what arithmetic does when an exact result does not fit and digits must be discarded. It answers
// "round or refuse", which is a separate question from RoundingMode's "which direction"; the two are set
// independently, by SetLossPolicy and SetArithmeticRounding.
//
// The distinction matters because losing digits is not one condition but three. A result whose integer part does not
// fit at scale 0 is an overflow and is always NaN(Overflow), whatever the policy. A result that keeps its significant
// digits and drops only the tail is inexact - 1/3 and Sqrt(2) are inexact at every scale, and so is any product that
// needs more than MaxScale places. A result that rounds all the way to zero from non-zero operands has lost every
// significant digit it had; that is an underflow, and it is the one case where a plausible-looking value (zero) hides
// a total loss of information.
type LossPolicy uint8

const (
	// LossRound rounds whatever must be discarded, with the mode set by SetArithmeticRounding. It is the zero value
	// and the package default: every result is a value unless the integer part itself does not fit.
	LossRound LossPolicy = iota

	// LossNaNOnUnderflow returns NaN(Underflow) when a result rounds to zero although neither operand was zero, and
	// rounds everything else. It is the strictest policy that still lets 1/3 and Sqrt(2) be values, and it is the
	// recommended setting for money: a non-zero amount never silently becomes zero, and an inexact division still
	// works.
	LossNaNOnUnderflow

	// LossNaNOnInexact returns NaN(Inexact) whenever a nonzero digit is discarded. It is an exactness guarantee, not
	// a rounding strategy: under it 1/3, Sqrt(2) and most products of two high-scale operands are NaN. Use it to
	// detect rounding rather than to perform it.
	LossNaNOnInexact
)

var lossPolicyNames = [...]string{
	LossRound:          "LossRound",
	LossNaNOnUnderflow: "LossNaNOnUnderflow",
	LossNaNOnInexact:   "LossNaNOnInexact",
}

// IsValid reports whether p is one of the defined loss policies.
func (p LossPolicy) IsValid() bool {
	return p <= LossNaNOnInexact
}

// String returns the constant's name, or "LossPolicy(n)" for an undefined value.
func (p LossPolicy) String() string {
	if p.IsValid() {
		return lossPolicyNames[p]
	}
	return "LossPolicy(" + string(rune('0'+p/100%10)) + string(rune('0'+p/10%10)) + string(rune('0'+p%10)) + ")"
}

// lossPolicy is the policy arithmetic applies when digits must be discarded. Process-global; see SetLossPolicy.
var lossPolicy = LossRound

// SetLossPolicy sets what arithmetic does when an exact result does not fit and digits must be discarded. The default
// is LossRound, which rounds with the mode set by SetArithmeticRounding and leaves every result that was not NaN
// before loss policies existed bit-identical.
//
// Call it after SetArithmeticRounding, not before: SetArithmeticRounding also writes the policy, because passing it
// ROUND_NAN is the deprecated spelling of LossNaNOnInexact.
//
// This is process-global state, like SetDefaultScale: set it once during initialization, before any decimal is used.
// Calling it while other goroutines are calculating is a data race. It panics on an undefined policy, and only at
// configuration time.
func SetLossPolicy(p LossPolicy) {
	if !p.IsValid() {
		panic(state.InvalidLossPolicy.Error())
	}
	lossPolicy = p
}

// CurrentLossPolicy returns the policy arithmetic applies when digits must be discarded (see SetLossPolicy).
func CurrentLossPolicy() LossPolicy {
	return lossPolicy
}

// lossState maps a loss of digits to the NaN state it produces under the policy, or state.OK when the policy accepts
// the loss and the caller should return the rounded value. zero reports that the rounded coefficient is zero, which
// for non-zero operands makes the loss an underflow rather than a plain rounding.
//
// It returns a state rather than a Dec128 so that it inlines into the arithmetic slow arms, where it is reached on
// every inexact result: under the default LossRound it compiles to one comparison.
func lossState(zero bool, p LossPolicy) state.State {
	switch p {
	case LossNaNOnInexact:
		return state.Inexact
	case LossNaNOnUnderflow:
		if zero {
			return state.Underflow
		}
	}
	return state.OK
}
