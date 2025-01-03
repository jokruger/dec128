package uint128

import (
	"encoding/binary"
	"math/bits"

	"github.com/jokruger/dec128/errors"
)

func (self Uint128) PutBytes(bs []byte) errors.Error {
	if len(bs) < 16 {
		return errors.NotEnoughBytes
	}

	binary.LittleEndian.PutUint64(bs[:8], self.Lo)
	binary.LittleEndian.PutUint64(bs[8:], self.Hi)

	return errors.None
}

func (self Uint128) PutBytesBigEndian(bs []byte) errors.Error {
	if len(bs) < 16 {
		return errors.NotEnoughBytes
	}

	binary.BigEndian.PutUint64(bs[:8], self.Hi)
	binary.BigEndian.PutUint64(bs[8:], self.Lo)

	return errors.None
}

func (self Uint128) AppendBytes(bs []byte) []byte {
	bs = binary.LittleEndian.AppendUint64(bs, self.Lo)
	bs = binary.LittleEndian.AppendUint64(bs, self.Hi)
	return bs
}

func (self Uint128) AppendBytesBigEndian(bs []byte) []byte {
	bs = binary.BigEndian.AppendUint64(bs, self.Hi)
	bs = binary.BigEndian.AppendUint64(bs, self.Lo)
	return bs
}

func (self Uint128) ReverseBytes() Uint128 {
	return Uint128{bits.ReverseBytes64(self.Hi), bits.ReverseBytes64(self.Lo)}
}
