package dec128

import (
	"encoding/binary"
	"io"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/gobu/bytes"
)

// WriteBinary writes the binary representation of Dec128 to w.
func (self Dec128) WriteBinary(w io.Writer) error {
	// Pre-allocate a fixed buffer on stack (max 18 bytes)
	var buf [18]byte

	// Fast path for error state or zero coefficient
	if self.state >= state.Error || self.coef.IsZero() {
		buf[0] = byte(self.state)
		_, err := w.Write(buf[:1])
		return err
	}

	// Cache fields locally to reduce repeated lookups.
	flags := byte(self.state)
	pos := 1
	h, l, e := self.coef.Hi, self.coef.Lo, self.exp

	if h != 0 {
		flags |= 0x80 // 0b10000000
		binary.LittleEndian.PutUint64(buf[pos:], h)
		pos += 8
	}
	if l != 0 {
		flags |= 0x40 // 0b01000000
		binary.LittleEndian.PutUint64(buf[pos:], l)
		pos += 8
	}
	if e != 0 {
		flags |= 0x20 // 0b00100000
		buf[pos] = e
		pos++
	}

	// Write flag byte at the beginning
	buf[0] = flags

	// Single write call for the complete buffer
	_, err := w.Write(buf[:pos])
	return err
}

// ReadBinary reads the binary representation of Dec128 from r.
func (self *Dec128) ReadBinary(r io.Reader) error {
	// Read the flag byte.
	var s [1]byte
	if _, err := io.ReadFull(r, s[:]); err != nil {
		return err
	}

	// Determine how many extra bytes to read based on flag bits.
	extraLen := 0
	if s[0]&0b10000000 != 0 {
		extraLen += 8
	}
	if s[0]&0b01000000 != 0 {
		extraLen += 8
	}
	if s[0]&0b00100000 != 0 {
		extraLen += 1
	}

	// Pre-allocate a fixed-size extra buffer (maximum 8+8+1=17 bytes).
	var extra [17]byte
	if extraLen > 0 {
		if _, err := io.ReadFull(r, extra[:extraLen]); err != nil {
			return err
		}
	}

	// Parse the extra bytes.
	idx := 0
	var h, l uint64
	var e uint8
	if s[0]&0b10000000 != 0 {
		h = binary.LittleEndian.Uint64(extra[idx : idx+8])
		idx += 8
	}
	if s[0]&0b01000000 != 0 {
		l = binary.LittleEndian.Uint64(extra[idx : idx+8])
		idx += 8
	}
	if s[0]&0b00100000 != 0 {
		e = extra[idx]
		// idx++ // (not needed since there's no further use)
	}

	self.state = state.State(s[0] & 0b00011111)
	self.coef.Hi = h
	self.coef.Lo = l
	self.exp = e

	return nil
}

// EncodeBinary encodes the binary representation of Dec128 into buf. It returns an error if buf is too small, otherwise the number of bytes written into buf.
func (self Dec128) EncodeBinary(buf []byte) (int, error) {
	b := bytes.MakeWriteBuffer(buf, 0, false)
	err := self.WriteBinary(&b)
	return b.Pos(), err
}

// DecodeBinary decodes binary representation of Dec128 from buf. It returns an error if buf is too small, otherwise the number of bytes consumed from buf.
func (self *Dec128) DecodeBinary(buf []byte) (int, error) {
	b := bytes.MakeReadBuffer(buf, 0, false)
	err := self.ReadBinary(&b)
	return b.Pos(), err
}

// AppendBinary appends the binary representation of Dec128 to the end of b (allocating a larger slice if necessary) and returns the updated slice.
func (self Dec128) AppendBinary(buf []byte) ([]byte, error) {
	b := bytes.MakeWriteBuffer(buf, len(buf), true)
	err := self.WriteBinary(&b)
	return b.Bytes(), err
}

// MarshalBinary implements the encoding.BinaryMarshaler interface. It encodes Dec128 into a binary form and returns the result.
func (self Dec128) MarshalBinary() (data []byte, err error) {
	return self.AppendBinary(nil)
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface.
func (self *Dec128) UnmarshalBinary(data []byte) error {
	n, err := self.DecodeBinary(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return io.ErrShortBuffer
	}
	return nil
}

// GobEncode implements the gob.GobEncoder interface for gob serialization.
func (self Dec128) GobEncode() ([]byte, error) {
	return self.MarshalBinary()
}

// GobDecode implements the gob.GobDecoder interface for gob serialization.
func (self *Dec128) GobDecode(data []byte) error {
	return self.UnmarshalBinary(data)
}
