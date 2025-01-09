package uint128

import "math/bits"

// And returns the bitwise AND of two 128-bit unsigned integers.
func (self Uint128) And(other Uint128) Uint128 {
	return Uint128{self.Lo & other.Lo, self.Hi & other.Hi}
}

// And64 returns the bitwise AND of a 128-bit unsigned integer and a 64-bit unsigned integer.
func (self Uint128) And64(other uint64) Uint128 {
	return Uint128{self.Lo & other, 0}
}

// Or returns the bitwise OR of two 128-bit unsigned integers.
func (self Uint128) Or(other Uint128) Uint128 {
	return Uint128{self.Lo | other.Lo, self.Hi | other.Hi}
}

// Or64 returns the bitwise OR of a 128-bit unsigned integer and a 64-bit unsigned integer.
func (self Uint128) Or64(other uint64) Uint128 {
	return Uint128{self.Lo | other, self.Hi}
}

// Xor returns the bitwise XOR of two 128-bit unsigned integers.
func (self Uint128) Xor(other Uint128) Uint128 {
	return Uint128{self.Lo ^ other.Lo, self.Hi ^ other.Hi}
}

// Xor64 returns the bitwise XOR of a 128-bit unsigned integer and a 64-bit unsigned integer.
func (self Uint128) Xor64(other uint64) Uint128 {
	return Uint128{self.Lo ^ other, self.Hi}
}

// Lsh returns the result of shifting a 128-bit unsigned integer to the left by n bits.
func (self Uint128) Lsh(n uint) Uint128 {
	if n > 64 {
		return Uint128{0, self.Lo << (n - 64)}
	}
	return Uint128{self.Lo << n, self.Hi<<n | self.Lo>>(64-n)}
}

// Rsh returns the result of shifting a 128-bit unsigned integer to the right by n bits.
func (self Uint128) Rsh(n uint) Uint128 {
	if n > 64 {
		return Uint128{self.Hi >> (n - 64), 0}
	}
	return Uint128{self.Lo>>n | self.Hi<<(64-n), self.Hi >> n}
}

// LeadingZeroBitsCount returns the number of leading zero bits in a 128-bit unsigned integer.
func (self Uint128) LeadingZeroBitsCount() int {
	if self.Hi > 0 {
		return bits.LeadingZeros64(self.Hi)
	}
	return 64 + bits.LeadingZeros64(self.Lo)
}

// TrailingZeroBitsCount returns the number of trailing zero bits in a 128-bit unsigned integer.
func (self Uint128) TrailingZeroBitsCount() int {
	if self.Lo > 0 {
		return bits.TrailingZeros64(self.Lo)
	}
	return 64 + bits.TrailingZeros64(self.Hi)
}

// NonZeroBitsCount returns the number of non-zero bits in a 128-bit unsigned integer.
func (self Uint128) NonZeroBitsCount() int {
	return bits.OnesCount64(self.Lo) + bits.OnesCount64(self.Hi)
}

// RotateBitsLeft returns the result of rotating a 128-bit unsigned integer to the left by k bits.
func (self Uint128) RotateBitsLeft(k int) Uint128 {
	const n = 128
	s := uint(k) & (n - 1)
	return self.Lsh(s).Or(self.Rsh(n - s))
}

// RotateBitsRight returns the result of rotating a 128-bit unsigned integer to the right by k bits.
func (self Uint128) RotateBitsRight(k int) Uint128 {
	return self.RotateBitsLeft(-k)
}

// ReverseBits returns the result of reversing the bits of a 128-bit unsigned integer.
func (self Uint128) ReverseBits() Uint128 {
	return Uint128{bits.Reverse64(self.Hi), bits.Reverse64(self.Lo)}
}
