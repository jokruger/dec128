// Package errors provides custom error type and error codes for uint128 and dec128 packages.
package errors

import "errors"

type Error uint8

const (
	None Error = iota
	NotANumber
	DivisionByZero
	Overflow
	Underflow
	Negative
	NotEnoughBytes
	InvalidFormat
	PrecisionOutOfRange
	RescaleToLessPrecision
	SqrtNegative
)

var code2err = [...]error{
	None:                   nil,
	NotANumber:             errors.New("not a number"),
	DivisionByZero:         errors.New("division by zero"),
	Overflow:               errors.New("overflow"),
	Underflow:              errors.New("underflow"),
	Negative:               errors.New("negative value in unsigned operation"),
	NotEnoughBytes:         errors.New("not enough bytes"),
	InvalidFormat:          errors.New("invalid format"),
	PrecisionOutOfRange:    errors.New("precision out of range"),
	RescaleToLessPrecision: errors.New("rescale to less precision"),
	SqrtNegative:           errors.New("square root of negative number"),
}

func (e Error) Value() error {
	return code2err[e]
}

func (e Error) Error() string {
	return e.Value().Error()
}
