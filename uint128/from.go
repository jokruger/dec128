package uint128

import (
	"encoding/binary"
	"math/big"

	"github.com/jokruger/dec128/errors"
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
func FromBigInt(i *big.Int) (Uint128, errors.Error) {
	if i.Sign() < 0 {
		return Zero, errors.Negative
	}

	if i.BitLen() > 128 {
		return Zero, errors.Overflow
	}

	return Uint128{i.Uint64(), i.Rsh(i, 64).Uint64()}, errors.None
}

// FromString creates a new Uint128 from a string
func FromString(s string) (Uint128, errors.Error) {
	sz := len(s)

	if sz == 0 {
		return Zero, errors.None
	}

	if sz <= MaxSafeStrLen64 {
		// can be safely parsed as uint64
		var u uint64
		for i := range sz {
			if s[i] < '0' || s[i] > '9' {
				return Zero, errors.InvalidFormat
			}
			u = u*10 + uint64(s[i]-'0')
		}
		return FromUint64(u), errors.None
	}

	var u Uint128
	var err errors.Error
	for i := range sz {
		if s[i] < '0' || s[i] > '9' {
			return Zero, errors.InvalidFormat
		}

		u, err = u.Mul64(10)
		if err != errors.None {
			return Zero, err
		}

		u, err = u.Add64(uint64(s[i] - '0'))
		if err != errors.None {
			return Zero, err
		}
	}

	return u, errors.None
}
