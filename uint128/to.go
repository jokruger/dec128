package uint128

import (
	"math/big"

	"github.com/jokruger/dec128/errors"
)

func (self Uint128) Uint64() (uint64, errors.Error) {
	if self.Hi != 0 {
		return 0, errors.Overflow
	}
	return self.Lo, errors.None
}

func (self Uint128) Bytes() [16]byte {
	bs := [16]byte{}
	self.PutBytes(bs[:])
	return bs
}

func (self Uint128) BytesBigEndian() [16]byte {
	bs := [16]byte{}
	self.PutBytesBigEndian(bs[:])
	return bs
}

func (self Uint128) BigInt() *big.Int {
	i := new(big.Int).SetUint64(self.Hi)
	i = i.Lsh(i, 64)
	i = i.Xor(i, new(big.Int).SetUint64(self.Lo))
	return i
}

func (self Uint128) String() string {
	if self.IsZero() {
		return "0"
	}

	buf := [MaxStrLen]byte{}
	for i := range MaxStrLen {
		buf[i] = '0'
	}

	n := self.StringToBuf(buf[:])
	return string(buf[n:])
}

func (self Uint128) StringToBuf(buf []byte) int {
	if self.Hi == 0 {
		i := len(buf)
		for u := self.Lo; u != 0; i-- {
			buf[i-1] += byte(u % 10)
			u /= 10
		}
		return i
	}

	u := self
	for i := len(buf); ; i -= 19 {
		q, r, _ := u.QuoRem64(1e19)
		var n int
		for ; r != 0; r /= 10 {
			n++
			buf[i-n] += byte(r % 10)
		}
		if q.IsZero() {
			return i - n
		}
		u = q
	}
}
