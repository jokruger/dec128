package state

import "testing"

func TestState(t *testing.T) {
	for _, s := range []State{Default, Neg} {
		if !s.IsOK() {
			t.Errorf("Expected state %d to be OK", s)
		}
		if s.IsError() {
			t.Errorf("Expected state %d not to be an error", s)
		}
		if s.String() == "" {
			t.Errorf("Expected state %d to have a string representation", s)
		}
		if s.Error() != nil {
			t.Errorf("Expected state %d not to have an error", s)
		}
	}

	for _, s := range []State{Error, NaN, DivisionByZero, Overflow, Underflow, NegativeInUnsignedOp, NotEnoughBytes, InvalidFormat, ScaleOutOfRange, RescaleToLowerScale, SqrtNegative} {
		if s.IsOK() {
			t.Errorf("Expected state %d to be an error", s)
		}
		if !s.IsError() {
			t.Errorf("Expected state %d to be an error", s)
		}
		if s.String() == "" {
			t.Errorf("Expected state %d to have a string representation", s)
		}
		if s.Error() == nil {
			t.Errorf("Expected state %d to have an error", s)
		}
	}
}
