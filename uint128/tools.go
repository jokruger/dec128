package uint128

import (
	"encoding/binary"
	"math/bits"
)

func (self Uint128) PutBytes(bs []byte) error {
	if len(bs) < 16 {
		return ErrNotEnough
	}
	binary.LittleEndian.PutUint64(bs[:8], self.Lo)
	binary.LittleEndian.PutUint64(bs[8:], self.Hi)
	return nil
}

func (self Uint128) PutBytesBigEndian(bs []byte) error {
	if len(bs) < 16 {
		return ErrNotEnough
	}
	binary.BigEndian.PutUint64(bs[:8], self.Hi)
	binary.BigEndian.PutUint64(bs[8:], self.Lo)
	return nil
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
