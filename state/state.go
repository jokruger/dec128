// Package state provides the State type that encodes the sign and the error condition of a uint128 or dec128 value
// in a single byte.
//
// The ordering is load-bearing: codes below Error (Default and Neg) are valid values, Error and everything above are
// error conditions, which dec128 reports as NaN. Code numbers are persisted by the binary format and are therefore
// never renumbered; new codes are appended.
package state

import "errors"

// State encodes the sign of a valid value (Default or Neg) or, for a code of Error and above, the reason a value is
// invalid.
type State uint8

// State codes are hard-coded for binary compatibility.
const (
	Default = State(0)
	Neg     = State(1)

	Error = State(2)

	NaN                  = State(3)
	DivisionByZero       = State(4)
	Overflow             = State(5)
	Underflow            = State(6)
	NegativeInUnsignedOp = State(7)
	NotEnoughBytes       = State(8)
	InvalidFormat        = State(9)
	SqrtNegative         = State(12)
	ScaleOutOfRange      = State(13)
	RescaleToLowerScale  = State(14)
	Null                 = State(15)
	Inexact              = State(16)
	InvalidRoundingMode  = State(17)
)

// PrecisionOutOfRange is the former name of ScaleOutOfRange; the code stays reserved so it is never reused.
//
// Deprecated: Use ScaleOutOfRange instead.
const PrecisionOutOfRange = State(10)

// RescaleToLessPrecision is the former name of RescaleToLowerScale; the code stays reserved so it is never reused.
//
// Deprecated: Use RescaleToLowerScale instead.
const RescaleToLessPrecision = State(11)

var code2str = [...]string{
	Default: "default",
	Neg:     "negative",

	Error: "error",

	NaN:                    "not a number",
	DivisionByZero:         "division by zero",
	Overflow:               "overflow",
	Underflow:              "underflow",
	NegativeInUnsignedOp:   "negative value in unsigned operation",
	NotEnoughBytes:         "not enough bytes",
	InvalidFormat:          "invalid format",
	PrecisionOutOfRange:    "precision out of range",    // Deprecated
	RescaleToLessPrecision: "rescale to less precision", // Deprecated
	SqrtNegative:           "square root of negative number",
	ScaleOutOfRange:        "scale out of range",
	RescaleToLowerScale:    "rescale to lower scale",
	Null:                   "null",
	Inexact:                "inexact result",
	InvalidRoundingMode:    "invalid rounding mode",
}

var code2err = [...]error{
	Default:                nil,
	Neg:                    nil,
	Error:                  errors.New("logical error"),
	NaN:                    errors.New("not a number"),
	DivisionByZero:         errors.New("division by zero"),
	Overflow:               errors.New("overflow"),
	Underflow:              errors.New("underflow"),
	NegativeInUnsignedOp:   errors.New("negative value in unsigned operation"),
	NotEnoughBytes:         errors.New("not enough bytes"),
	InvalidFormat:          errors.New("invalid format"),
	PrecisionOutOfRange:    errors.New("precision out of range"),    // Deprecated
	RescaleToLessPrecision: errors.New("rescale to less precision"), // Deprecated
	SqrtNegative:           errors.New("square root of negative number"),
	ScaleOutOfRange:        errors.New("scale out of range"),
	RescaleToLowerScale:    errors.New("rescale to lower scale"),
	Null:                   errors.New("null"),
	Inexact:                errors.New("inexact result"),
	InvalidRoundingMode:    errors.New("invalid rounding mode"),
}

// OK is the state of a valid, non-negative value; it is another name for Default.
const OK = Default

var errInvalidState = errors.New("invalid state code")

// IsValid reports whether s is one of the defined state codes.
func (s State) IsValid() bool {
	return int(s) < len(code2str)
}

// IsOK reports whether s denotes a valid value (Default or Neg).
func (s State) IsOK() bool {
	return s < Error
}

// IsError reports whether s denotes an error condition (Error or above).
func (s State) IsError() bool {
	return s >= Error
}

// String returns a short description of the state. A code that is not defined is described as invalid rather than
// indexing out of range.
func (s State) String() string {
	if !s.IsValid() {
		return "invalid state code"
	}
	return code2str[s]
}

// Error returns the error value for the state: nil for a valid value, otherwise a sentinel that is the same for every
// occurrence of the code, so callers can compare with == or errors.Is. A code that is not defined yields a generic
// invalid-state error.
func (s State) Error() error {
	if !s.IsValid() {
		return errInvalidState
	}
	return code2err[s]
}
