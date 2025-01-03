package uint128

import "math/big"

func (self Uint128) Uint64() (uint64, error) {
	if self.Hi != 0 {
		return 0, ErrOverflow
	}
	return self.Lo, nil
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

	u := self
	buf := []byte("0000000000000000000000000000000000000000")
	for i := len(buf); ; i -= 19 {
		q, r, _ := u.QuoRem64(1e19)
		var n int
		for ; r != 0; r /= 10 {
			n++
			buf[i-n] += byte(r % 10)
		}
		if q.IsZero() {
			return string(buf[i-n:])
		}
		u = q
	}
}
