package unit

import (
	"encoding/json"
	"testing"

	"github.com/jokruger/dec128"
)

func TestDecimalJson(t *testing.T) {
	type testStruct struct {
		D dec128.Dec128 `json:"d"`
	}

	type testCase struct {
		t testStruct
		s string
	}

	tests := [...]testCase{
		{testStruct{dec128.Zero}, `{"d":"0"}`},
		{testStruct{dec128.FromString("1")}, `{"d":"1"}`},
		{testStruct{dec128.FromString("1.01")}, `{"d":"1.01"}`},
		{testStruct{dec128.FromString("1.000001")}, `{"d":"1.000001"}`},
		{testStruct{dec128.FromString("12345678901234567890.123456789")}, `{"d":"12345678901234567890.123456789"}`},
		{testStruct{dec128.FromString("-1")}, `{"d":"-1"}`},
		{testStruct{dec128.FromString("-1.01")}, `{"d":"-1.01"}`},
		{testStruct{dec128.FromString("-1.000001")}, `{"d":"-1.000001"}`},
		{testStruct{dec128.FromString("-12345678901234567890.123456789")}, `{"d":"-12345678901234567890.123456789"}`},
	}

	for _, test := range tests {
		s, err := json.Marshal(test.t)
		if err != nil {
			t.Errorf("Error marshalling %v: %v", test, err)
		}
		if string(s) != test.s {
			t.Errorf("Expected '%v', got '%v'", test.s, string(s))
		}
		var q testStruct
		if err := json.Unmarshal(s, &q); err != nil {
			t.Errorf("Error unmarshalling %v: %v", test, err)
		}
		if !q.D.Equal(test.t.D) {
			t.Errorf("Expected '%v', got '%v'", test.t.D, q.D)
		}
	}
}
