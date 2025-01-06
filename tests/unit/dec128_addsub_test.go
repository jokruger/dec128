package unit

import (
	"fmt"
	"testing"

	"github.com/jokruger/dec128"
)

func TestDecimalAdd(t *testing.T) {
	type testCase struct {
		a string
		b string
		s string
	}

	testCases := [...]testCase{
		{"0", "0", "0"},
		{"0", "1", "1"},
		{"1", "0", "1"},
		{"1", "1", "2"},
		{"-1", "0", "-1"},
		{"0", "-1", "-1"},
		{"-1", "-1", "-2"},
		{"-1", "1", "0"},
		{"1", "-1", "0"},
		{"1", "10", "11"},
		{"10", "1", "11"},
		{"-1", "-10", "-11"},
		{"-10", "-1", "-11"},
		{"-1", "10", "9"},
		{"10", "-1", "9"},
		{"1000000", "-0.0000001", "999999.9999999"},
		{"999999.9999999", "0.0000001", "1000000"},
		{"340282366920938463463374607431768211454", "1", "340282366920938463463374607431768211455"},
		{"340282366920938463463374607431768211454", "1.00", "340282366920938463463374607431768211455"}, // overflow due to precision fixed by auto canonicalization
		{"NaN", "1", "NaN"},
		{"1", "NaN", "NaN"},
		{"NaN", "NaN", "NaN"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalAdd(%s + %s)", tc.a, tc.b), func(t *testing.T) {
			a := dec128.FromString(tc.a)
			b := dec128.FromString(tc.b)
			c := a.Add(b)
			s := c.String()
			if s != tc.s {
				t.Errorf("Expected '%s', got: %s", tc.s, s)
			}
		})
	}
}

func TestDecimalSub(t *testing.T) {
	type testCase struct {
		a string
		b string
		s string
	}

	testCases := [...]testCase{
		{"0", "0", "0"},
		{"0", "1", "-1"},
		{"1", "0", "1"},
		{"1", "1", "0"},
		{"-1", "0", "-1"},
		{"0", "-1", "1"},
		{"-1", "-1", "0"},
		{"-1", "1", "-2"},
		{"1", "-1", "2"},
		{"1", "10", "-9"},
		{"10", "1", "9"},
		{"-1", "-10", "9"},
		{"-10", "-1", "-9"},
		{"-1", "10", "-11"},
		{"10", "-1", "11"},
		{"1000000", "0.0000001", "999999.9999999"},
		{"999999.9999999", "-0.0000001", "1000000"},
		{"340282366920938463463374607431768211455", "1", "340282366920938463463374607431768211454"},
		{"340282366920938463463374607431768211455", "1.00", "340282366920938463463374607431768211454"}, // overflow due to precision fixed by auto canonicalization
		{"NaN", "1", "NaN"},
		{"1", "NaN", "NaN"},
		{"NaN", "NaN", "NaN"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalSub(%s - %s)", tc.a, tc.b), func(t *testing.T) {
			a := dec128.FromString(tc.a)
			b := dec128.FromString(tc.b)
			c := a.Sub(b)
			s := c.String()
			if s != tc.s {
				t.Errorf("Expected '%s', got: %s", tc.s, s)
			}
		})
	}
}
