package dec128

import (
	"fmt"
	"io"
)

// Printing a Dec128 with the fmt verbs a money value is printed with.
//
// Without this, fmt reaches String through the Stringer interface and %v and %s work, but %.2f - the most natural
// way to write an amount into a line of text - prints %!f(dec128.Dec128=1.5) instead, because the verb does not
// apply to a value that is not a float. Implementing Formatter makes the verb mean what it says.

var _ fmt.Formatter = Dec128{}

// Format implements fmt.Formatter, so that the verbs an amount is printed with do what they say:
//
//	%v, %s   String: the value with trailing zeros removed
//	%f, %F   StringFixed: the value at its own scale. With a precision, %.2f, the value at exactly that many
//	         places, rounded half to even - the rounding Go's own %f applies to a float64. A precision above
//	         MaxScale is treated as MaxScale.
//	%e, %E   StringSci: scientific notation, with E for an upper-case exponent marker
//	%d       the integer part, truncated toward zero, for the currencies with no minor unit
//	%#v      an expression that reconstructs the value, in place of the unexported fields
//
// The width, '-' (left justify), '+' (always sign) and '0' (pad with zeros after the sign) flags are honoured for
// every verb. A NaN prints as NaN under all of them, ignores the precision and the sign flag, and is padded with
// spaces even under '0', as a float NaN is. Any other verb prints %!x(dec128.Dec128=1.5), which is what fmt does
// with a verb a type does not accept.
//
// Note that %#v reads "dec128.FromString(\"1.50\")", which rebuilds the value but, for a NaN, not the reason it
// carries; ErrorDetails is where that lives. The process-global SetTrimOutput is read by String and StringFixed and
// so applies here too.
func (d Dec128) Format(f fmt.State, verb rune) {
	var buf [MaxSciStrLen]byte
	var body []byte

	switch verb {
	case 'v':
		if f.Flag('#') {
			_, _ = io.WriteString(f, "dec128.FromString(\"")
			_, _ = f.Write(d.StringFixedToBuf(buf[:]))
			_, _ = io.WriteString(f, "\")")
			return
		}
		body = d.StringToBuf(buf[:])
	case 's':
		body = d.StringToBuf(buf[:])
	case 'f', 'F':
		v := d
		if p, ok := f.Precision(); ok {
			v = d.RescaleRound(uint8(min(max(p, 0), int(MaxScale))), ROUND_BANK)
		}
		body = v.StringFixedToBuf(buf[:])
	case 'e', 'E':
		body = d.StringSciToBuf(buf[:])
		if verb == 'E' {
			for i := range body {
				if body[i] == 'e' {
					body[i] = 'E'
				}
			}
		}
	case 'd':
		body = d.Trunc(0).StringToBuf(buf[:])
	default:
		_, _ = io.WriteString(f, "%!"+string(verb)+"(dec128.Dec128=")
		_, _ = f.Write(d.StringToBuf(buf[:]))
		_, _ = io.WriteString(f, ")")
		return
	}

	// The sign is written separately from the digits, because '0' pads between the two.
	nan := d.IsNaN()
	sign := byte(0)
	if !nan {
		switch {
		case len(body) > 0 && body[0] == '-':
			sign, body = '-', body[1:]
		case f.Flag('+'):
			sign = '+'
		}
	}

	n := len(body)
	if sign != 0 {
		n++
	}
	pad := 0
	if w, ok := f.Width(); ok && w > n {
		pad = w - n
	}
	left := f.Flag('-')
	zero := f.Flag('0') && !left && !nan

	if !left && !zero {
		writeRepeat(f, ' ', pad)
	}
	if sign != 0 {
		_, _ = f.Write([]byte{sign})
	}
	if zero {
		writeRepeat(f, '0', pad)
	}
	_, _ = f.Write(body)
	if left {
		writeRepeat(f, ' ', pad)
	}
}

// writeRepeat writes n copies of c, a chunk at a time so that a wide field costs one call per chunk rather than one
// per byte.
func writeRepeat(w io.Writer, c byte, n int) {
	if n <= 0 {
		return
	}
	var chunk [32]byte
	for i := range chunk {
		chunk[i] = c
	}
	for n > 0 {
		k := min(n, len(chunk))
		_, _ = w.Write(chunk[:k])
		n -= k
	}
}
