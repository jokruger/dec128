package dec128

import (
	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// The decomposer interface: the one decimal interchange contract that is neither a wire format nor a database type.
//
// Decompose and Compose are the shape agreed between the SQL driver authors - it is what the pgx and go-mssqldb drivers
// all speak. The four parts are a form byte, a sign, the coefficient as a big-endian base-2 integer and a base-ten
// exponent, and the value is
//
//	(neg) (form == finite) coefficient * 10 ^ exponent
//
// The other codecs here each answer a narrower question: binary.go is our own compact form, pg.go is PostgreSQL's
// NUMERIC on the wire, ieee.go is decimal128 and int128.go is a fixed-scale integer. This one is the general
// conversation between two decimal types that know nothing about each other, which is why it is worth having even
// though every one of those is cheaper.
//
// The interface is not imported: depending on github.com/golang-sql/decomposer to name it would be the package's first
// dependency, and this library has none. It is restated here for the compile-time assertion and nothing else.
type decomposer interface {
	Decompose(buf []byte) (form byte, negative bool, coefficient []byte, exponent int32)
	Compose(form byte, negative bool, coefficient []byte, exponent int32) error
}

var _ decomposer = (*Dec128)(nil)

// The three form bytes the interface defines. A Dec128 produces only formFinite and formNaN, and accepts only those
// two: there is no infinity in this type, and silently turning one into a NaN would lose the distinction a caller was
// relying on the interface to carry.
const (
	formFinite   byte = 0
	formInfinite byte = 1
	formNaN      byte = 2
)

// Decompose returns d as the four parts of the decomposer interface: the form, the sign, the coefficient as a
// big-endian base-2 integer and the exponent, which for a Dec128 is always -Scale and therefore never positive.
//
// A NaN yields form 2 with no coefficient, whatever reason the NaN carries; the interface has no room for the reason,
// so a caller who needs it should use ErrorDetails before converting. Zero yields a zero-length coefficient, which is
// what the interface specifies, and keeps its scale in the exponent so that 0.00 and 0 survive the round trip.
//
// If buf has room for the coefficient it is filled and returned, so a caller with a scratch buffer of 16 bytes - the
// widest a Dec128 coefficient can be - converts without allocating. The bytes are not written from the start of buf;
// that is the interface's convention, not an oversight.
//
// It reads no process-global configuration.
func (d Dec128) Decompose(buf []byte) (form byte, negative bool, coefficient []byte, exponent int32) {
	if d.state >= state.Error {
		return formNaN, false, nil, 0
	}

	negative = d.state == state.Neg
	exponent = -int32(d.scale)

	if d.coef.IsZero() {
		return formFinite, negative, nil, exponent
	}

	// The minimal big-endian encoding: the 16 bytes of the coefficient with the leading zeros dropped, which is what
	// big.Int.Bytes would produce and what the interface asks for.
	b := d.coef.BytesBigEndian()
	i := 0
	for i < len(b) && b[i] == 0 {
		i++
	}

	if n := len(b) - i; cap(buf) >= n {
		buf = buf[:n]
		copy(buf, b[i:])
		return formFinite, negative, buf, exponent
	}

	coefficient = make([]byte, len(b)-i)
	copy(coefficient, b[i:])

	return formFinite, negative, coefficient, exponent
}

// Compose sets d from the four parts of the decomposer interface, and returns an error when the value they describe is
// not one a Dec128 can hold: state.Overflow when it is too large, state.Inexact when it has digits below MaxScale that
// cannot be cancelled away.
//
// The value composed is coefficient * 10^exponent with the given sign. A positive exponent is applied to the
// coefficient, and an exponent below -MaxScale is answered by cancelling factors of ten out of the coefficient, so
// that 15000000 * 10^-25 composes to 1.5E-18 rather than being refused for the way it was written. What cannot be
// cancelled cannot be represented, and that is an error rather than a rounding: this is an interchange codec, and a
// codec that quietly drops digits is worse than one that refuses them. 15 * 10^-25 is therefore state.Inexact, not
// a zero and not a rounded 1E-19.
//
// Form 2 sets d to NaN(NaN) - the reason does not cross the interface - and returns nil, because a NaN is a value here
// and not a failure. Form 1 is an error: this type has no infinity, and composing one as a NaN would tell the caller
// their infinity arrived when it did not. Any other form byte is an error.
//
// On any error d is left unchanged.
//
// It reads no process-global configuration.
func (d *Dec128) Compose(form byte, negative bool, coefficient []byte, exponent int32) error {
	switch form {
	case formFinite:
	case formNaN:
		*d = Dec128{state: state.NaN}
		return nil
	case formInfinite:
		return state.InvalidFormat.Error()
	default:
		return state.InvalidFormat.Error()
	}

	if exponent > int32(MaxScale) {
		// Past this the multiplication overflows for every nonzero coefficient this type can hold.
		return state.Overflow.Error()
	}
	if exponent < -int32(MaxScale)-int32(uint128.MaxStrLen) {
		// Below this there are not enough digits in any coefficient to cancel the exponent back into range.
		return state.Inexact.Error()
	}

	// The coefficient is decoded straight into a uint128 rather than through a big.Int, so that Compose allocates
	// nothing like every other decode path here.
	i := 0
	for i < len(coefficient) && coefficient[i] == 0 {
		i++
	}
	sig := coefficient[i:]
	if len(sig) > 16 {
		return state.Overflow.Error()
	}
	var b [16]byte
	copy(b[16-len(sig):], sig)
	c := uint128.FromBytesBigEndian(b)

	if c.IsZero() {
		// A zero keeps whatever scale the exponent describes, when that is one a Dec128 has.
		if exponent > 0 || exponent < -int32(MaxScale) {
			*d = Dec128{}
			return nil
		}
		*d = Dec128{scale: uint8(-exponent)}
		return nil
	}

	// Bring the exponent into [-MaxScale, 0]: upward by multiplying the coefficient, downward by cancelling factors
	// of ten out of it. Cancelling is exact or it does not happen.
	if exponent > 0 {
		lo, hi := c.Mul64Carry(Pow10Uint64[exponent])
		if hi != 0 {
			return state.Overflow.Error()
		}
		c, exponent = lo, 0
	}
	for exponent < -int32(MaxScale) {
		k := uint8(min(-exponent-int32(MaxScale), int32(MaxScale)))
		q, r, st := c.QuoRemPow10(k)
		if st >= state.Error {
			return st.Error()
		}
		if r != 0 {
			return state.Inexact.Error()
		}
		c, exponent = q, exponent+int32(k)
	}

	st := state.Default
	if negative {
		st = state.Neg
	}

	*d = Dec128{coef: c, scale: uint8(-exponent), state: st}

	return nil
}
