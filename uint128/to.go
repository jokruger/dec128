package uint128

import (
	"math/big"

	"github.com/jokruger/dec128/state"
)

// Uint64 returns the value as uint64 if it fits, otherwise it returns an error.
func (self Uint128) Uint64() (uint64, state.State) {
	if self.Hi != 0 {
		return 0, state.Overflow
	}
	return self.Lo, state.OK
}

// Bytes returns the value as a [16]byte array.
func (self Uint128) Bytes() [16]byte {
	bs := [16]byte{}
	self.PutBytes(bs[:])
	return bs
}

// BytesBigEndian returns the value as a [16]byte array in big-endian order.
func (self Uint128) BytesBigEndian() [16]byte {
	bs := [16]byte{}
	self.PutBytesBigEndian(bs[:])
	return bs
}

// BigInt returns the value as a big.Int.
func (self Uint128) BigInt() *big.Int {
	i := new(big.Int).SetUint64(self.Hi)
	i = i.Lsh(i, 64)
	i = i.Xor(i, new(big.Int).SetUint64(self.Lo))
	return i
}

// String returns the value as a string.
func (self Uint128) String() string {
	if self.IsZero() {
		return ZeroStr
	}

	buf := [MaxStrLen]byte{}
	sb := self.StringToBuf(buf[:])

	return string(sb)
}

// StringToBuf writes the value as a string to the given buffer (from end to start) and returns a slice containing the string.
func (self Uint128) StringToBuf(buf []byte) []byte {
	q := self
	i := len(buf)
	var r uint64
	var n int

	for {
		if q.Hi == 0 {
			r = q.Lo
			for r != 0 {
				i--
				buf[i] = '0' + byte(r%10)
				r /= 10
			}
			return buf[i:]
		}

		q, r, _ = q.QuoRem64(1e19)
		n = 19
		for r != 0 {
			i--
			buf[i] = '0' + byte(r%10)
			r /= 10
			n--
		}

		if q.IsZero() {
			return buf[i:]
		}

		for n > 0 {
			i--
			buf[i] = '0'
			n--
		}
	}
}
