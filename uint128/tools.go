package uint128

import (
	"encoding/binary"
	"math/bits"

	"github.com/jokruger/dec128/errors"
)

// PutBytes writes the Uint128 to the byte slice bs in little-endian order.
func (self Uint128) PutBytes(bs []byte) errors.Error {
	if len(bs) < 16 {
		return errors.NotEnoughBytes
	}

	binary.LittleEndian.PutUint64(bs[:8], self.Lo)
	binary.LittleEndian.PutUint64(bs[8:], self.Hi)

	return errors.None
}

// PutBytesBigEndian writes the Uint128 to the byte slice bs in big-endian order.
func (self Uint128) PutBytesBigEndian(bs []byte) errors.Error {
	if len(bs) < 16 {
		return errors.NotEnoughBytes
	}

	binary.BigEndian.PutUint64(bs[:8], self.Hi)
	binary.BigEndian.PutUint64(bs[8:], self.Lo)

	return errors.None
}

// AppendBytes appends the Uint128 to the byte slice bs in little-endian order.
func (self Uint128) AppendBytes(bs []byte) []byte {
	bs = binary.LittleEndian.AppendUint64(bs, self.Lo)
	bs = binary.LittleEndian.AppendUint64(bs, self.Hi)
	return bs
}

// AppendBytesBigEndian appends the Uint128 to the byte slice bs in big-endian order.
func (self Uint128) AppendBytesBigEndian(bs []byte) []byte {
	bs = binary.BigEndian.AppendUint64(bs, self.Hi)
	bs = binary.BigEndian.AppendUint64(bs, self.Lo)
	return bs
}

// ReverseBytes returns the Uint128 with the byte order reversed.
func (self Uint128) ReverseBytes() Uint128 {
	return Uint128{bits.ReverseBytes64(self.Hi), bits.ReverseBytes64(self.Lo)}
}

// QuoRem256By128 returns quotient, remainder and error.
func QuoRem256By128(u Uint128, carry Uint128, v Uint128) (Uint128, Uint128, errors.Error) {
	if carry.IsZero() {
		return Uint128{Lo: u.Lo, Hi: u.Hi}.QuoRem(v)
	}

	if v.Hi == 0 && carry.Hi == 0 {
		q, r, err := QuoRem192By64(u, carry.Lo, v.Lo)
		return q, Uint128{Lo: r}, err
	}

	// now we have u192 / u128 or u256 / u128
	if carry.Compare(v) >= 0 {
		// obviously the result won't fit into u128
		return Zero, Zero, errors.Overflow
	}

	// perform u256 / u128, where carry < u128
	// based on divllu from https://github.com/ridiculousfish/libdivide
	// algorithm is explained in this blog post: https://ridiculousfish.com/blog/posts/labor-of-division-episode-iv.html
	// normalize v
	n := bits.LeadingZeros64(v.Hi)

	// 0 <= n <= 63, so it's safe to convert to uint
	v = v.Lsh(uint(n))

	// shift u to the left by n bits (n < 64)
	a := [4]uint64{}
	a[0] = u.Lo << n
	a[1] = u.Lo>>(64-n) | u.Hi<<n
	a[2] = u.Hi>>(64-n) | carry.Lo<<n
	a[3] = carry.Lo>>(64-n) | carry.Hi<<n

	// q = a / v
	aLen := 3
	if a[3] != 0 || (a[3] == 0 && a[2] > v.Hi) {
		aLen = 4
	}

	q := [2]uint64{}

	for i := aLen - 3; i >= 0; i-- {
		u2, u1, u0 := a[i+2], a[i+1], a[i]

		// trial quotient tq = [u2,u1,u0] / v ~= [u2,u1] / v.hi
		// tq <= q + 2
		tq, r := bits.Div64(u2, u1, v.Hi)

		c1h, c1l := bits.Mul64(tq, v.Lo)
		c1 := Uint128{Lo: c1l, Hi: c1h}
		c2 := Uint128{Lo: u0, Hi: r}

		// adjust tq
		var k uint64
		if c1.Compare(c2) > 0 {
			k = 1

			// d = c1 - c2
			if SubUnsafe(c1, c2).Compare(v) > 0 {
				k = 2
			}
		}

		q[i] = tq - k

		// true remainder rem = [u2,u1,u0] - q*v = c2 - c1 + k*v (k <= 2)
		var rem Uint128
		switch k {
		case 0:
			// rem = c2 - c1
			rem = SubUnsafe(c2, c1)
		case 1:
			// rem = c2 - c1 + v = v - (c1 - c2) with c1 > c2
			rem = SubUnsafe(c1, c2)
			rem = SubUnsafe(v, rem)
		case 2:
			// rem = c2 - c1 + 2*v = v + v - (c1 - c2) with c1 > c2
			// v = max(u128) - not(v)
			// --> rem = v - not(v) + max(u128) - (c1 - c2)
			//  v >= not(v) because v is normalized. Hence, we can safely caculate rem without checking overflow
			c12 := SubUnsafe(c1, c2)
			c12 = SubUnsafe(Max, c12)
			rem = SubUnsafe(v, Uint128{Lo: ^v.Lo, Hi: ^v.Hi})

			// this also can't overflow because rem < v <= max(u128)
			rem, _ = rem.Add(c12)
		}

		a[i+1], a[i] = rem.Hi, rem.Lo
	}

	// 0 <= n <= 63, so it's safe to convert to uint
	r := Uint128{Lo: a[0], Hi: a[1]}.Rsh(uint(n))

	return Uint128{Lo: q[0], Hi: q[1]}, r, errors.None
}

// QuoRem192By64 return q, r which:
// q must be a u128
// u = q*v + r
// Returns error if u.carry >= v, because the result can't fit into u128
func QuoRem192By64(u Uint128, carry uint64, v uint64) (Uint128, uint64, errors.Error) {
	if carry >= v {
		return Zero, 0, errors.Overflow
	}

	// can't panic because we already check u.carry < v (u.carry.hi == 0 && u.carry.lo < v)
	hi, rem := bits.Div64(carry, u.Hi, v)

	// can't panic because rem < v
	lo, r := bits.Div64(rem, u.Lo, v)

	return Uint128{Lo: lo, Hi: hi}, r, errors.None
}

// SubUnsafe returns u - v with u >= v
// must be called only when u >= v or the result will be incorrect
func SubUnsafe(u Uint128, v Uint128) Uint128 {
	lo, borrow := bits.Sub64(u.Lo, v.Lo, 0)
	hi, _ := bits.Sub64(u.Hi, v.Hi, borrow)
	return Uint128{Lo: lo, Hi: hi}
}
