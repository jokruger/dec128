package uint128

import "math/bits"

func (self Uint128) And(other Uint128) Uint128 {
	return Uint128{self.Lo & other.Lo, self.Hi & other.Hi}
}

func (self Uint128) And64(other uint64) Uint128 {
	return Uint128{self.Lo & other, 0}
}

func (self Uint128) Or(other Uint128) Uint128 {
	return Uint128{self.Lo | other.Lo, self.Hi | other.Hi}
}

func (self Uint128) Or64(other uint64) Uint128 {
	return Uint128{self.Lo | other, self.Hi}
}

func (self Uint128) Xor(other Uint128) Uint128 {
	return Uint128{self.Lo ^ other.Lo, self.Hi ^ other.Hi}
}

func (self Uint128) Xor64(other uint64) Uint128 {
	return Uint128{self.Lo ^ other, self.Hi}
}

func (self Uint128) Lsh(n uint) Uint128 {
	if n > 64 {
		return Uint128{0, self.Lo << (n - 64)}
	}
	return Uint128{self.Lo << n, self.Hi<<n | self.Lo>>(64-n)}
}

func (self Uint128) Rsh(n uint) Uint128 {
	if n > 64 {
		return Uint128{self.Hi >> (n - 64), 0}
	}
	return Uint128{self.Lo>>n | self.Hi<<(64-n), self.Hi >> n}
}

func (self Uint128) LeadingZeroBitsCount() int {
	if self.Hi > 0 {
		return bits.LeadingZeros64(self.Hi)
	}
	return 64 + bits.LeadingZeros64(self.Lo)
}

func (self Uint128) TrailingZeroBitsCount() int {
	if self.Lo > 0 {
		return bits.TrailingZeros64(self.Lo)
	}
	return 64 + bits.TrailingZeros64(self.Hi)
}

func (self Uint128) NonZeroBitsCount() int {
	return bits.OnesCount64(self.Lo) + bits.OnesCount64(self.Hi)
}

func (self Uint128) RotateBitsLeft(k int) Uint128 {
	const n = 128
	s := uint(k) & (n - 1)
	return self.Lsh(s).Or(self.Rsh(n - s))
}

func (self Uint128) RotateBitsRight(k int) Uint128 {
	return self.RotateBitsLeft(-k)
}

func (self Uint128) ReverseBits() Uint128 {
	return Uint128{bits.Reverse64(self.Hi), bits.Reverse64(self.Lo)}
}
