package uint128

import "math/bits"

// And returns the bitwise AND of two 128-bit unsigned integers.
func (ui Uint128) And(other Uint128) Uint128 {
	return Uint128{ui.Lo & other.Lo, ui.Hi & other.Hi}
}

// And64 returns the bitwise AND of a 128-bit unsigned integer and a 64-bit unsigned integer.
func (ui Uint128) And64(other uint64) Uint128 {
	return Uint128{ui.Lo & other, 0}
}

// Or returns the bitwise OR of two 128-bit unsigned integers.
func (ui Uint128) Or(other Uint128) Uint128 {
	return Uint128{ui.Lo | other.Lo, ui.Hi | other.Hi}
}

// Or64 returns the bitwise OR of a 128-bit unsigned integer and a 64-bit unsigned integer.
func (ui Uint128) Or64(other uint64) Uint128 {
	return Uint128{ui.Lo | other, ui.Hi}
}

// Xor returns the bitwise XOR of two 128-bit unsigned integers.
func (ui Uint128) Xor(other Uint128) Uint128 {
	return Uint128{ui.Lo ^ other.Lo, ui.Hi ^ other.Hi}
}

// Xor64 returns the bitwise XOR of a 128-bit unsigned integer and a 64-bit unsigned integer.
func (ui Uint128) Xor64(other uint64) Uint128 {
	return Uint128{ui.Lo ^ other, ui.Hi}
}

// Lsh returns the result of shifting a 128-bit unsigned integer to the left by n bits.
func (ui Uint128) Lsh(n uint) Uint128 {
	if n > 64 {
		return Uint128{0, ui.Lo << (n - 64)}
	}
	return Uint128{ui.Lo << n, ui.Hi<<n | ui.Lo>>(64-n)}
}

// Rsh returns the result of shifting a 128-bit unsigned integer to the right by n bits.
func (ui Uint128) Rsh(n uint) Uint128 {
	if n > 64 {
		return Uint128{ui.Hi >> (n - 64), 0}
	}
	return Uint128{ui.Lo>>n | ui.Hi<<(64-n), ui.Hi >> n}
}

// LeadingZeroBitsCount returns the number of leading zero bits in a 128-bit unsigned integer.
func (ui Uint128) LeadingZeroBitsCount() int {
	if ui.Hi > 0 {
		return bits.LeadingZeros64(ui.Hi)
	}
	return 64 + bits.LeadingZeros64(ui.Lo)
}

// TrailingZeroBitsCount returns the number of trailing zero bits in a 128-bit unsigned integer.
func (ui Uint128) TrailingZeroBitsCount() int {
	if ui.Lo > 0 {
		return bits.TrailingZeros64(ui.Lo)
	}
	return 64 + bits.TrailingZeros64(ui.Hi)
}

// NonZeroBitsCount returns the number of non-zero bits in a 128-bit unsigned integer.
func (ui Uint128) NonZeroBitsCount() int {
	return bits.OnesCount64(ui.Lo) + bits.OnesCount64(ui.Hi)
}

// RotateBitsLeft returns the result of rotating a 128-bit unsigned integer to the left by k bits.
func (ui Uint128) RotateBitsLeft(k int) Uint128 {
	const n = 128
	s := uint(k) & (n - 1)
	return ui.Lsh(s).Or(ui.Rsh(n - s))
}

// RotateBitsRight returns the result of rotating a 128-bit unsigned integer to the right by k bits.
func (ui Uint128) RotateBitsRight(k int) Uint128 {
	return ui.RotateBitsLeft(-k)
}

// ReverseBits returns the result of reversing the bits of a 128-bit unsigned integer.
func (ui Uint128) ReverseBits() Uint128 {
	return Uint128{bits.Reverse64(ui.Hi), bits.Reverse64(ui.Lo)}
}
