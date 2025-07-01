package uint128

import (
	"encoding/binary"
	"math/big"

	"github.com/jokruger/dec128/state"
)

// FromUint64 creates a new Uint128 from a uint64
func FromUint64(u uint64) Uint128 {
	return Uint128{u, 0}
}

// FromBytes creates a new Uint128 from a [16]byte
func FromBytes(bs [16]byte) Uint128 {
	return Uint128{binary.LittleEndian.Uint64(bs[:8]), binary.LittleEndian.Uint64(bs[8:])}
}

// FromBytesBigEndian creates a new Uint128 from a [16]byte in big endian
func FromBytesBigEndian(b [16]byte) Uint128 {
	return Uint128{binary.BigEndian.Uint64(b[8:]), binary.BigEndian.Uint64(b[:8])}
}

// FromBigInt creates a new Uint128 from a *big.Int
func FromBigInt(i *big.Int) (Uint128, state.State) {
	if i.Sign() < 0 {
		return Zero, state.NegativeInUnsignedOp
	}

	if i.BitLen() > 128 {
		return Zero, state.Overflow
	}

	return Uint128{i.Uint64(), i.Rsh(i, 64).Uint64()}, state.OK
}

// FromString creates a new Uint128 from a string
func FromString[S string | []byte](s S) (Uint128, state.State) {
	sz := len(s)

	if sz == 0 {
		return Zero, state.OK
	}

	if sz <= MaxSafeStrLen64 {
		// can be safely parsed as uint64
		var u uint64
		for i := range sz {
			c := s[i]
			if c < '0' || c > '9' {
				return Zero, state.InvalidFormat
			}
			u = u*10 + uint64(c-'0')
		}
		return Uint128{u, 0}, state.OK
	}

	var u Uint128
	var e state.State
	for i := range sz {
		c := s[i]
		if c < '0' || c > '9' {
			return Zero, state.InvalidFormat
		}

		u, e = u.Mul64(10)
		if e >= state.Error {
			return Zero, e
		}

		u, e = u.Add64(uint64(c - '0'))
		if e >= state.Error {
			return Zero, e
		}
	}

	return u, state.OK
}
