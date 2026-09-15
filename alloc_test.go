package dec128

import (
	"encoding/binary"
	"testing"

	"github.com/jokruger/dec128/state"
)

// Allocation gates. These lock the allocation behavior the library advertises: the numeric core, parsing, comparison,
// rounding, the caller-buffer formatters and every decode path allocate nothing; the functions that return an owned
// string or slice allocate exactly once for it. A change that trips a gate is either a regression or a deliberate
// contract change that must update this table.

var (
	allocSinkDec    Dec128
	allocSinkDec2   Dec128
	allocSinkInt    int
	allocSinkBool   bool
	allocSinkStr    string
	allocSinkBytes  []byte
	allocSinkErr    error
	allocSinkAny    any
	allocSinkShares []Dec128
	allocRatios     = []Dec128{FromInt64(1), FromInt64(2), FromInt64(3)}
)

func TestAllocationGates(t *testing.T) {
	a := FromString("1234.5678")
	b := FromString("8765.4321")
	large := FromString("12345678901234567890.123456789")
	nearMax := FromString("17014118346046923173.1687303715884105727")
	src := "1234.5678"
	srcBytes := []byte(src)
	// Scan takes `any`; box once here, as database/sql does, so the gate measures Scan
	// itself and not the interface conversion at the call site.
	var srcAny any = src
	var srcBytesAny any = srcBytes
	var nilAny any
	jsonBytes := []byte(`"1234.5678"`)
	var bin [MaxBytes]byte
	n, err := a.EncodeBinary(bin[:])
	if err != nil {
		t.Fatal(err)
	}
	binBytes := bin[:n]
	var strBuf [MaxStrLen]byte
	var sciBuf [MaxSciStrLen]byte
	var pgBuf [MaxPgNumericBytes]byte
	pgN, _ := large.EncodePgNumeric(pgBuf[:])
	pgBytes := pgBuf[:pgN]
	var ieeeBuf [IEEEBytes]byte
	_, _ = a.EncodeIEEE(ieeeBuf[:])
	appendBuf := make([]byte, 0, 64)
	allocSinkShares = make([]Dec128, 0, 8)

	zero := []struct {
		name string
		fn   func()
	}{
		{"FromString(string)", func() { allocSinkDec = FromString(src) }},
		{"FromString([]byte)", func() { allocSinkDec = FromString(srcBytes) }},
		{"FromSafeString", func() { allocSinkDec = FromSafeString(src) }},
		{"FromString sci", func() { allocSinkDec = FromString("1.2345678e3") }},
		{"FromString NaN", func() { allocSinkDec = FromString("NaN") }},
		{"FromString invalid", func() { allocSinkDec = FromString("abc") }},
		{"FromInt64", func() { allocSinkDec = FromInt64(-123456789) }},
		{"FromFloat64", func() { allocSinkDec = FromFloat64(1234.5678) }},
		{"Add", func() { allocSinkDec = a.Add(b) }},
		{"AddRound", func() { allocSinkDec = a.AddRound(b, 2, ROUND_BANK) }},
		{"SubRound", func() { allocSinkDec = a.SubRound(b, 2, ROUND_BANK) }},
		{"MulAddRound", func() { allocSinkDec = a.MulAddRound(b, a, 2, ROUND_BANK) }},
		{"MulAddRound wide", func() { allocSinkDec = nearMax.MulAddRound(nearMax, a, 2, ROUND_BANK) }},
		{"PowIntRound", func() { allocSinkDec = FromString("1.05").PowIntRound(360, 10, ROUND_BANK) }},
		{"PowIntRound negative", func() { allocSinkDec = FromString("1.05").PowIntRound(-360, 10, ROUND_BANK) }},
		{"RoundToPlaces", func() { allocSinkDec = large.RoundToPlaces(-3, ROUND_BANK) }},
		{"RoundToMultiple", func() { allocSinkDec = a.RoundToMultiple(FromString("0.05"), ROUND_BANK) }},
		{"ScaleByPow10", func() { allocSinkDec = a.ScaleByPow10(-2) }},
		{"DivRoundInexact", func() { allocSinkDec, allocSinkBool = a.DivRoundInexact(b, 2, ROUND_BANK) }},
		{"AppendAllocate", func() {
			allocSinkShares, allocSinkBool = a.AppendAllocate(allocSinkShares[:0], allocRatios, 4)
		}},
		{"AppendSplit", func() { allocSinkShares, allocSinkBool = a.AppendSplit(allocSinkShares[:0], 3, 4) }},
		{"Add unaligned", func() { allocSinkDec = a.Add(large) }},
		{"Sub", func() { allocSinkDec = a.Sub(b) }},
		{"Mul", func() { allocSinkDec = a.Mul(b) }},
		{"Mul large", func() { allocSinkDec = large.Mul(b) }},
		{"Mul reduce", func() { allocSinkDec = nearMax.Mul(nearMax) }},
		{"MulRound", func() { allocSinkDec = a.MulRound(b, 2, ROUND_BANK) }},
		{"Div", func() { allocSinkDec = a.Div(b) }},
		{"Div near max", func() { allocSinkDec = nearMax.Div(b) }},
		{"QuoRem", func() { allocSinkDec, allocSinkDec2 = a.QuoRem(b) }},
		{"Sqrt", func() { allocSinkDec = a.Sqrt() }},
		{"Sum", func() { allocSinkDec = Sum(a, b, large, nearMax) }},
		{"Accumulator", func() {
			// the accumulator itself is a value the caller places; only NewAccumulator returns a pointer
			var acc Accumulator
			acc.Add(a)
			acc.AddMul(b, large)
			acc.Sub(nearMax)
			allocSinkDec = acc.Total(2, ROUND_BANK)
		}},
		{"DivRound", func() { allocSinkDec = a.DivRound(b, 2, ROUND_HALF_AWAY_FROM_ZERO) }},
		{"Div reduce", func() { allocSinkDec = nearMax.Div(FromString("0.001")) }},
		{"SqrtRound", func() { allocSinkDec = a.SqrtRound(2, ROUND_BANK) }},
		{"Compare", func() { allocSinkInt = a.Compare(b) }},
		{"Compare unaligned", func() { allocSinkInt = a.Compare(large) }},
		{"Equal", func() { allocSinkBool = a.Equal(b) }},
		{"RoundBank", func() { allocSinkDec = a.RoundBank(2) }},
		{"Round dispatcher", func() { allocSinkDec = a.Round(2, ROUND_HALF_AWAY_FROM_ZERO) }},
		{"Round ROUND_NAN", func() { allocSinkDec = a.Round(2, ROUND_NAN) }},
		{"RescaleRound", func() { allocSinkDec = a.RescaleRound(2, ROUND_BANK) }},
		{"Trunc", func() { allocSinkDec = a.Trunc(2) }},
		{"Rescale up", func() { allocSinkDec = a.Rescale(10) }},
		{"Canonical", func() { allocSinkDec = FromString("1.500000").Canonical() }},
		{"StringToBuf", func() { allocSinkBytes = large.StringToBuf(strBuf[:]) }},
		{"StringSciToBuf", func() { allocSinkBytes = large.StringSciToBuf(sciBuf[:]) }},
		{"StringFixedToBuf", func() { allocSinkBytes = large.StringFixedToBuf(strBuf[:]) }},
		{"AppendText", func() { allocSinkBytes, allocSinkErr = large.AppendText(appendBuf[:0]) }},
		{"AppendBinary", func() { allocSinkBytes, allocSinkErr = large.AppendBinary(appendBuf[:0]) }},
		{"AppendPgNumeric", func() { allocSinkBytes, allocSinkErr = large.AppendPgNumeric(appendBuf[:0]) }},
		{"AppendIEEE", func() { allocSinkBytes, allocSinkErr = large.AppendIEEE(appendBuf[:0]) }},
		{"AppendInt128", func() { allocSinkBytes, allocSinkErr = a.AppendInt128(appendBuf[:0], 4, binary.LittleEndian) }},
		{"NextUp", func() { allocSinkDec = a.NextUp() }},
		{"QuoRem", func() { allocSinkDec, allocSinkDec = large.QuoRem(a) }},
		{"Sqrt", func() { allocSinkDec = large.Sqrt() }},
		{"EncodeBinary", func() { allocSinkInt, allocSinkErr = a.EncodeBinary(bin[:]) }},
		{"EncodePgNumeric", func() { allocSinkInt, allocSinkErr = large.EncodePgNumeric(pgBuf[:]) }},
		{"DecodePgNumeric", func() { allocSinkErr = allocSinkDec.DecodePgNumeric(pgBytes) }},
		{"EncodeIEEE", func() { allocSinkInt, allocSinkErr = a.EncodeIEEE(ieeeBuf[:]) }},
		{"DecodeIEEE", func() { allocSinkErr = allocSinkDec.DecodeIEEE(ieeeBuf[:]) }},
		{"EncodeInt128", func() { allocSinkInt, allocSinkErr = a.EncodeInt128(ieeeBuf[:], 4, binary.LittleEndian) }},
		{"DecodeInt128", func() { allocSinkErr = allocSinkDec.DecodeInt128(ieeeBuf[:], 4, binary.LittleEndian) }},
		{"DecodeBinary", func() { allocSinkInt, allocSinkErr = allocSinkDec.DecodeBinary(binBytes) }},
		{"UnmarshalBinary", func() { allocSinkErr = allocSinkDec.UnmarshalBinary(binBytes) }},
		{"UnmarshalJSON", func() { allocSinkErr = allocSinkDec.UnmarshalJSON(jsonBytes) }},
		{"UnmarshalText", func() { allocSinkErr = allocSinkDec.UnmarshalText(srcBytes) }},
		{"Scan(string)", func() { allocSinkErr = allocSinkDec.Scan(srcAny) }},
		{"Scan([]byte)", func() { allocSinkErr = allocSinkDec.Scan(srcBytesAny) }},
		{"Scan(nil)", func() { allocSinkErr = allocSinkDec.Scan(nilAny) }},
		{"Int64", func() { _, allocSinkErr = a.Int64() }},
		{"InexactFloat64", func() { _, allocSinkErr = a.InexactFloat64() }},
	}
	for _, c := range zero {
		if got := testing.AllocsPerRun(100, c.fn); got != 0 {
			t.Errorf("%s: %v allocs/op, want 0", c.name, got)
		}
	}

	one := []struct {
		name string
		fn   func()
	}{
		{"String", func() { allocSinkStr = a.String() }},
		{"StringFixed", func() { allocSinkStr = a.StringFixed() }},
		{"StringSci", func() { allocSinkStr = a.StringSci() }},
		{"MarshalJSON", func() { allocSinkBytes, allocSinkErr = a.MarshalJSON() }},
		{"MarshalText", func() { allocSinkBytes, allocSinkErr = a.MarshalText() }},
		{"MarshalBinary", func() { allocSinkBytes, allocSinkErr = a.MarshalBinary() }},
	}
	for _, c := range one {
		if got := testing.AllocsPerRun(100, c.fn); got != 1 {
			t.Errorf("%s: %v allocs/op, want 1 (the returned value)", c.name, got)
		}
	}

	// The shortcut cases - a zero at scale 0, a NaN, a NULL - allocate the same single owned slice as any other
	// value. They used to return the package's own constant, which cost nothing but let a caller corrupt it
	// (see TestMarshalResultIsPrivate); one allocation is the price of the ownership the contract above states.
	zeroVal, nanVal, nullVal := Zero, NaN(state.Overflow), Null()
	shortcut := []struct {
		name string
		fn   func()
		want float64
	}{
		{"MarshalText/zero", func() { allocSinkBytes, allocSinkErr = zeroVal.MarshalText() }, 1},
		{"MarshalJSON/zero", func() { allocSinkBytes, allocSinkErr = zeroVal.MarshalJSON() }, 1},
		{"MarshalText/NaN", func() { allocSinkBytes, allocSinkErr = nanVal.MarshalText() }, 1},
		{"MarshalJSON/NaN", func() { allocSinkBytes, allocSinkErr = nanVal.MarshalJSON() }, 1},
		{"MarshalJSON/NULL", func() { allocSinkBytes, allocSinkErr = nullVal.MarshalJSON() }, 1},
		// AppendText and AppendBinary write into the caller's buffer, so they stay free of allocation
		{"AppendText/zero", func() { allocSinkBytes, allocSinkErr = zeroVal.AppendText(appendBuf[:0]) }, 0},
		{"AppendText/NaN", func() { allocSinkBytes, allocSinkErr = nanVal.AppendText(appendBuf[:0]) }, 0},
	}
	for _, c := range shortcut {
		if got := testing.AllocsPerRun(100, c.fn); got != c.want {
			t.Errorf("%s: %v allocs/op, want %v", c.name, got, c.want)
		}
	}

	// driver.Value: the string plus boxing it into the interface.
	if got := testing.AllocsPerRun(100, func() { allocSinkAny, allocSinkErr = a.Value() }); got > 2 {
		t.Errorf("Value: %v allocs/op, want at most 2", got)
	}
}
