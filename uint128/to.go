package uint128

import (
	"math/big"

	"github.com/jokruger/dec128/state"
)

// Uint64 returns the value as uint64 if it fits, otherwise it returns an error.
func (ui Uint128) Uint64() (uint64, state.State) {
	if ui.Hi > 0 {
		return 0, state.Overflow
	}
	return ui.Lo, state.OK
}

// Bytes returns the value as a [16]byte array.
func (ui Uint128) Bytes() [16]byte {
	bs := [16]byte{}
	ui.PutBytes(bs[:])
	return bs
}

// BytesBigEndian returns the value as a [16]byte array in big-endian order.
func (ui Uint128) BytesBigEndian() [16]byte {
	bs := [16]byte{}
	ui.PutBytesBigEndian(bs[:])
	return bs
}

// BigInt returns the value as a big.Int.
func (ui Uint128) BigInt() *big.Int {
	i := new(big.Int).SetUint64(ui.Hi)
	i = i.Lsh(i, 64)
	i = i.Xor(i, new(big.Int).SetUint64(ui.Lo))
	return i
}

// String returns the value as a string.
func (ui Uint128) String() string {
	if ui.IsZero() {
		return ZeroStr
	}

	buf := [MaxStrLen]byte{}
	sb := ui.StringToBuf(buf[:])

	return string(sb)
}

// digitPairs holds "00".."99": two ASCII digits per value, so that formatting emits two digits per division step.
const digitPairs = "00010203040506070809" +
	"10111213141516171819" +
	"20212223242526272829" +
	"30313233343536373839" +
	"40414243444546474849" +
	"50515253545556575859" +
	"60616263646566676869" +
	"70717273747576777879" +
	"80818283848586878889" +
	"90919293949596979899"

// writeDigits writes exactly n digits of v (v < 10^n) right to left into buf ending just before pos, zero-padded on the
// left and two digits per step, and returns the new pos.
func writeDigits(buf []byte, pos int, v uint64, n int) int {
	for n >= 2 {
		pair := v % 100 * 2
		v /= 100
		pos -= 2
		buf[pos] = digitPairs[pair]
		buf[pos+1] = digitPairs[pair+1]
		n -= 2
	}
	if n == 1 {
		pos--
		buf[pos] = byte('0' + v%10)
	}
	return pos
}

// StringToBuf writes the decimal digits of ui into the end of buf and returns the slice holding them; a zero value
// yields an empty slice. buf must hold MaxStrLen bytes. The value is split into 19-digit chunks with the reciprocal
// division by 10^19 and each chunk is written two digits at a time.
func (ui Uint128) StringToBuf(buf []byte) []byte {
	q := ui
	i := len(buf)
	for q.Hi != 0 {
		var r uint64
		q, r = quoRem128Pow10(q, 19)
		i = writeDigits(buf, i, r, 19)
	}

	// the most significant (or only) limb, un-padded; open-coded so that the common one-limb case makes no call
	v := q.Lo
	for v >= 100 {
		pair := v % 100 * 2
		v /= 100
		i -= 2
		buf[i] = digitPairs[pair]
		buf[i+1] = digitPairs[pair+1]
	}

	if v >= 10 {
		i -= 2
		buf[i] = digitPairs[v*2]
		buf[i+1] = digitPairs[v*2+1]
	} else if v > 0 {
		i--
		buf[i] = byte('0' + v)
	}

	return buf[i:]
}
