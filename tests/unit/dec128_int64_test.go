package unit

import (
	"fmt"
	"testing"

	"github.com/jokruger/dec128"
)

func TestDecimalInt64Encoding(t *testing.T) {
	dec128.SetDefaultPrecision(19)

	type testCase struct {
		i int64
		p uint8
		s string
	}

	testCases := [...]testCase{
		{0, 0, "0"},
		{0, 1, "0"},
		{1, 0, "1"},
		{-1, 0, "-1"},
		{1, 1, "0.1"},
		{-1, 1, "-0.1"},
		{10, 1, "1"},
		{-10, 1, "-1"},
		{100, 1, "10"},
		{-100, 1, "-10"},
		{1000, 1, "100"},
		{-1000, 1, "-100"},
		{1, 10, "0.0000000001"},
		{-1, 10, "-0.0000000001"},
		{10, 10, "0.000000001"},
		{-10, 10, "-0.000000001"},
		{100, 10, "0.00000001"},
		{-100, 10, "-0.00000001"},
		{1000, 10, "0.0000001"},
		{-1000, 10, "-0.0000001"},
		{9223372036854775807, 0, "9223372036854775807"},
		{9223372036854775807, 1, "922337203685477580.7"},
		{9223372036854775807, 10, "922337203.6854775807"},
		{-9223372036854775808, 0, "-9223372036854775808"},
		{-9223372036854775808, 1, "-922337203685477580.8"},
		{-9223372036854775808, 10, "-922337203.6854775808"},
		{123456789, 3, "123456.789"},
		{-123456789, 3, "-123456.789"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("DecimalInt64Encoding(%v)", tc), func(t *testing.T) {
			d := dec128.DecodeFromInt64(tc.i, tc.p)
			s := d.String()
			if s != tc.s {
				t.Errorf("expected '%s', got: %s", tc.s, s)
			}

			d = dec128.FromString(tc.s)
			i, err := d.EncodeToInt64(tc.p)
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			if i != tc.i {
				t.Errorf("expected %d, got: %d", tc.i, i)
			}
		})
	}
}
