package unit

import (
	"fmt"
	"testing"

	"github.com/jokruger/dec128"
)

func TestDecimalConvUint128(t *testing.T) {
	d := dec128.FromString("NaN")
	_, _, err := d.Uint128()
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	d = dec128.FromString("-1")
	_, _, err = d.Uint128()
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	d = dec128.FromString("340282366920938463463374607431768211456")
	_, _, err = d.Uint128()
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	testCases := [...]string{"0", "1", "1234567890", "1234567890.123456789", "340282366920938463463374607431768211455", "12345678901234567890.123456789"}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalConvUint128(%s)", tc), func(t *testing.T) {
			d := dec128.FromString(tc)
			if d.IsNaN() {
				t.Errorf("Expected no error, got: %v", d.ErrorDetails())
			}
			u, p, err := d.Uint128()
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			d2 := dec128.FromUint128(u, p)
			if !d.Equal(d2) {
				t.Errorf("Expected %s, got: %s", d.String(), d2.String())
			}
			s := d2.String()
			if s != tc {
				t.Errorf("Expected '%s', got: %s", tc, s)
			}
		})
	}
}
