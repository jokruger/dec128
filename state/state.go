// Package state provides custom type to encode state and error codes for uint128 and dec128 packages.
package state

import "errors"

type State uint8

// State codes are hard-coded for binary compatibility.
const (
	Default = State(0)
	Neg     = State(1)

	Error = State(2)

	NaN                    = State(3)
	DivisionByZero         = State(4)
	Overflow               = State(5)
	Underflow              = State(6)
	NegativeInUnsignedOp   = State(7)
	NotEnoughBytes         = State(8)
	InvalidFormat          = State(9)
	PrecisionOutOfRange    = State(10)
	RescaleToLessPrecision = State(11)
	SqrtNegative           = State(12)
)

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
	PrecisionOutOfRange:    "precision out of range",
	RescaleToLessPrecision: "rescale to less precision",
	SqrtNegative:           "square root of negative number",
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
	PrecisionOutOfRange:    errors.New("precision out of range"),
	RescaleToLessPrecision: errors.New("rescale to less precision"),
	SqrtNegative:           errors.New("square root of negative number"),
}

var OK = Default

func (s State) IsOK() bool {
	return s < Error
}

func (s State) IsError() bool {
	return s >= Error
}

func (s State) String() string {
	return code2str[s]
}

func (s State) Error() error {
	return code2err[s]
}
