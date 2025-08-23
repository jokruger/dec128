package dec128

import (
	"encoding/binary"
	"io"

	"github.com/jokruger/dec128/state"
)

// BinarySize returns the number of bytes required to encode this instance of Dec128 in binary form.
func (d Dec128) BinarySize() int {
	sz := 1

	if d.state <= state.Error {
		if d.coef.Hi > 0 {
			sz += 8
		}
		if d.coef.Lo > 0 {
			sz += 8
		}
		if d.scale > 0 {
			sz++
		}
	}

	return sz
}

// EncodeBinary encodes the binary representation of Dec128 into buf. It returns an error if buf is too small, otherwise the number of bytes written into buf.
func (d Dec128) EncodeBinary(buf []byte) (int, error) {
	sz := len(buf)

	// Fast path for error state or zero coefficient
	if d.state >= state.Error || d.coef.IsZero() {
		if sz == 0 {
			return 0, io.ErrShortBuffer
		}
		buf[0] = byte(d.state)
		return 1, nil
	}

	flags := byte(d.state)
	pos := 1

	if d.coef.Hi > 0 {
		if pos+8 > sz {
			return pos, io.ErrShortBuffer
		}
		flags |= 0b10000000
		binary.LittleEndian.PutUint64(buf[pos:], d.coef.Hi)
		pos += 8
	}

	if d.coef.Lo > 0 {
		if pos+8 > sz {
			return pos, io.ErrShortBuffer
		}
		flags |= 0b01000000
		binary.LittleEndian.PutUint64(buf[pos:], d.coef.Lo)
		pos += 8
	}

	if d.scale > 0 {
		if pos+1 > sz {
			return pos, io.ErrShortBuffer
		}
		flags |= 0b00100000
		buf[pos] = d.scale
		pos++
	}

	// Write flag byte at the beginning
	buf[0] = flags

	return pos, nil
}

// DecodeBinary decodes binary representation of Dec128 from buf. It returns an error if buf is too small, otherwise the number of bytes consumed from buf.
func (d *Dec128) DecodeBinary(buf []byte) (int, error) {
	sz := len(buf)
	if sz == 0 {
		return 0, io.ErrShortBuffer
	}

	flags := buf[0]

	// Determine how many extra bytes to read.
	hiPresent := int((flags >> 7) & 1)  // 1 if coef.Hi is present, else 0
	loPresent := int((flags >> 6) & 1)  // 1 if coef.Lo is present, else 0
	expPresent := int((flags >> 5) & 1) // 1 if exponent is present, else 0
	extraLen := hiPresent*8 + loPresent*8 + expPresent

	if extraLen+1 > sz {
		return 1, io.ErrShortBuffer
	}

	// Parse the extra bytes.
	idx := 1
	var h, l uint64
	var e uint8

	if hiPresent > 0 {
		h = binary.LittleEndian.Uint64(buf[idx : idx+8])
		idx += 8
	}
	if loPresent > 0 {
		l = binary.LittleEndian.Uint64(buf[idx : idx+8])
		idx += 8
	}
	if expPresent > 0 {
		e = buf[idx]
		idx++
	}

	d.state = state.State(flags & 0b00011111)
	d.coef.Hi = h
	d.coef.Lo = l
	d.scale = e

	return idx, nil
}

// MarshalBinary implements the encoding.BinaryMarshaler interface. It encodes Dec128 into a binary form and returns the result.
func (d Dec128) MarshalBinary() ([]byte, error) {
	var buf [MaxBytes]byte
	n, err := d.EncodeBinary(buf[:])
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface.
func (d *Dec128) UnmarshalBinary(data []byte) error {
	n, err := d.DecodeBinary(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return io.ErrShortBuffer
	}
	return nil
}

// GobEncode implements the gob.GobEncoder interface for gob serialization.
func (d Dec128) GobEncode() ([]byte, error) {
	return d.MarshalBinary()
}

// GobDecode implements the gob.GobDecoder interface for gob serialization.
func (d *Dec128) GobDecode(data []byte) error {
	return d.UnmarshalBinary(data)
}

// AppendBinary appends the binary representation of Dec128 to the end of b (allocating a larger slice if necessary) and returns the updated slice.
func (d Dec128) AppendBinary(buf []byte) ([]byte, error) {
	var tmp [MaxBytes]byte
	n, err := d.EncodeBinary(tmp[:])
	if err != nil {
		return buf, err
	}
	return append(buf, tmp[:n]...), nil
}

// WriteBinary writes the binary representation of Dec128 to w.
func (d Dec128) WriteBinary(w io.Writer) error {
	var buf [MaxBytes]byte
	n, err := d.EncodeBinary(buf[:])
	if err != nil {
		return err
	}
	_, err = w.Write(buf[:n])
	return err
}

// ReadBinary reads the binary representation of Dec128 from r.
func (d *Dec128) ReadBinary(r io.Reader) error {
	// Use one fixed buffer of 18 bytes.
	var buf [18]byte

	// First, read the flag byte.
	if _, err := io.ReadFull(r, buf[:1]); err != nil {
		return err
	}
	flags := buf[0]

	// Determine how many extra bytes to read.
	hiPresent := int((flags >> 7) & 1)  // 1 if coef.Hi is present, else 0
	loPresent := int((flags >> 6) & 1)  // 1 if coef.Lo is present, else 0
	expPresent := int((flags >> 5) & 1) // 1 if exponent is present, else 0
	extraLen := hiPresent*8 + loPresent*8 + expPresent

	// Read the extra bytes directly into the remaining portion of buf.
	if extraLen > 0 {
		if _, err := io.ReadFull(r, buf[1:1+extraLen]); err != nil {
			return err
		}
	}

	// Parse the extra bytes.
	idx := 1
	var h, l uint64
	var e uint8

	if hiPresent > 0 {
		h = binary.LittleEndian.Uint64(buf[idx : idx+8])
		idx += 8
	}
	if loPresent > 0 {
		l = binary.LittleEndian.Uint64(buf[idx : idx+8])
		idx += 8
	}
	if expPresent > 0 {
		e = buf[idx]
		// idx++ is not needed since no further byte is used.
	}

	d.state = state.State(flags & 0b00011111)
	d.coef.Hi = h
	d.coef.Lo = l
	d.scale = e

	return nil
}
