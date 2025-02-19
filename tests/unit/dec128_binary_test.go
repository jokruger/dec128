package unit

import (
	"bytes"
	"encoding/gob"
	"testing"

	"github.com/jokruger/dec128"
)

type GobTestStruct struct {
	A dec128.Dec128
	B dec128.Dec128
	C []dec128.Dec128
}

func TestDecimalBinary(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		type test struct {
			d string
			s int
		}
		tests := []test{
			{"not a number", 1},
			{"0", 1},
			{"1", 9},
			{"-1", 9},
			{"1234567890", 9},
			{"-1234567890", 9},
			{"123456789012345678901234567890", 17},
			{"-123456789012345678901234567890", 17},
			{"0.1234", 10},
			{"-0.1234", 10},
			{"1234567890.1234567890", 10},
			{"-1234567890.1234567890", 10},
			{"123456789012345678901234567890.123", 18},
			{"-123456789012345678901234567890.123", 18},
		}

		for _, tc := range tests {
			t.Run(tc.d, func(t *testing.T) {
				d := dec128.FromString(tc.d)
				b, err := d.MarshalBinary()
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(b) != tc.s {
					t.Errorf("expected %d bytes, got %d", tc.s, len(b))
				}

				var d2 dec128.Dec128
				if err := d2.UnmarshalBinary(b); err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				if !d.Equal(d2) {
					t.Errorf("expected %s, got %s", d.String(), d2.String())
				}
			})
		}
	})

	t.Run("struct", func(t *testing.T) {
		var tc GobTestStruct
		tc.A = dec128.FromString("1234567890.1234567890")
		tc.B = dec128.FromString("123")
		tc.C = []dec128.Dec128{
			dec128.FromString("0.123456"),
			dec128.FromString("12345678901234567890.1234567890"),
			dec128.FromString("0.123456789012345678901234567890"),
			dec128.FromString("123456789012345678901234567890.1234567890"),
		}

		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		dec := gob.NewDecoder(&buf)

		if err := enc.Encode(tc); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		bs := buf.Bytes()
		if len(bs) != 161 {
			t.Errorf("expected 161 bytes, got %d", len(bs))
		}

		var tc2 GobTestStruct
		if err := dec.Decode(&tc2); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if !tc.A.Equal(tc2.A) {
			t.Errorf("expected %s, got %s", tc.A.String(), tc2.A.String())
		}
		if !tc.B.Equal(tc2.B) {
			t.Errorf("expected %s, got %s", tc.B.String(), tc2.B.String())
		}
		if len(tc.C) != len(tc2.C) {
			t.Errorf("expected %d elements, got %d", len(tc.C), len(tc2.C))
		}
		if !tc.C[0].Equal(tc2.C[0]) {
			t.Errorf("expected %s, got %s", tc.C[0].String(), tc2.C[0].String())
		}
		if !tc.C[1].Equal(tc2.C[1]) {
			t.Errorf("expected %s, got %s", tc.C[1].String(), tc2.C[1].String())
		}
		if !tc.C[2].Equal(tc2.C[2]) {
			t.Errorf("expected %s, got %s", tc.C[2].String(), tc2.C[2].String())
		}
		if !tc.C[3].Equal(tc2.C[3]) {
			t.Errorf("expected %s, got %s", tc.C[3].String(), tc2.C[3].String())
		}
	})
}
