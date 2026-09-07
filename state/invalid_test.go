package state

import "testing"

func TestInvalidStateCode(t *testing.T) {
	s := State(200)
	if s.IsValid() {
		t.Fatal("State(200) must not be valid")
	}
	if s.String() != "invalid state code" {
		t.Errorf("String() = %q", s.String())
	}
	if err := s.Error(); err == nil || err.Error() != "invalid state code" {
		t.Errorf("Error() = %v", err)
	}
	if s.Error() != State(255).Error() {
		t.Error("invalid codes must share one sentinel error")
	}
	if Overflow.Error() != Overflow.Error() || Overflow.String() != "overflow" {
		t.Error("defined codes must keep their sentinel and text")
	}
	if PrecisionOutOfRange != State(10) || RescaleToLessPrecision != State(11) || OK != Default {
		t.Error("reserved codes must keep their numbers")
	}
}
