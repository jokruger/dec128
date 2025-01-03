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
)

var code2err = [...]error{
	None:                nil,
	NotANumber:          errors.New("not a number"),
	DivisionByZero:      errors.New("division by zero"),
	Overflow:            errors.New("overflow"),
	Underflow:           errors.New("underflow"),
	Negative:            errors.New("negative value in unsigned operation"),
	NotEnoughBytes:      errors.New("not enough bytes"),
	InvalidFormat:       errors.New("invalid format"),
	PrecisionOutOfRange: errors.New("precision out of range"),
}

func (e Error) Value() error {
	return code2err[e]
}

func (e Error) Error() string {
	return e.Value().Error()
}
