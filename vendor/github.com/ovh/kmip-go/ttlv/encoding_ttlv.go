package ttlv

import (
	"encoding/binary"
	"math/big"
	"slices"
	"time"
)

type ttlvWriter struct {
	buf []byte
}

func newTTLVWriter() *ttlvWriter {
	return new(ttlvWriter)
}

func (enc *ttlvWriter) Bytes() []byte {
	return enc.buf
}

func (enc *ttlvWriter) Clear() {
	enc.buf = enc.buf[:0]
}

func (enc *ttlvWriter) writeByte(b byte) {
	enc.buf = append(enc.buf, b)
}

func (enc *ttlvWriter) pad(n int, v byte) {
	for range n {
		enc.writeByte(v)
	}
}

func (enc *ttlvWriter) writeTag(tag int) {
	enc.buf = append(enc.buf, byte(tag>>16), byte(tag>>8), byte(tag))
}

func (enc *ttlvWriter) writeType(typ Type) {
	enc.writeByte(byte(typ))
}

func (enc *ttlvWriter) writeLength(length int) {
	enc.buf = binary.BigEndian.AppendUint32(enc.buf, uint32(length))
}

func (enc *ttlvWriter) encodeAppend(tag int, typ Type, length int, f func([]byte) []byte) {
	enc.writeTag(tag)
	enc.writeType(typ)
	enc.writeLength(length)
	enc.buf = f(enc.buf)
}

func (enc *ttlvWriter) encodeAppendRightPadded(tag int, typ Type, length, padLen int, padVal byte, f func([]byte) []byte) {
	enc.encodeAppend(tag, typ, length, f)
	enc.pad(padLen, padVal)
}

func (enc *ttlvWriter) encodeAppendLeftPadded(tag int, typ Type, length, padLen int, padVal byte, f func([]byte) []byte) {
	enc.encodeAppend(tag, typ, length, func(b []byte) []byte {
		for range padLen {
			b = append(b, padVal)
		}
		return f(b)
	})
}

func (enc *ttlvWriter) Integer(tag int, value int32) {
	enc.encodeAppend(tag, TypeInteger, 4, func(b []byte) []byte {
		b = binary.BigEndian.AppendUint32(b, uint32(value))
		return append(b, 0, 0, 0, 0)
	})
}

func (enc *ttlvWriter) LongInteger(tag int, value int64) {
	enc.encodeAppend(tag, TypeLongInteger, 8, func(b []byte) []byte {
		return binary.BigEndian.AppendUint64(b, uint64(value))
	})
}

func (enc *ttlvWriter) BigInteger(tag int, value *big.Int) {
	//TODO: Optimize by appending the bytes directly into the buffer
	bytes, padVal, padLen := bigIntToBytes(value, 8)
	enc.encodeAppendLeftPadded(tag, TypeBigInteger, len(bytes)+padLen, padLen, padVal, func(b []byte) []byte {
		return append(b, bytes...)
	})
}

func (enc *ttlvWriter) Enum(enumtag, tag int, value uint32) {
	enc.encodeAppend(tag, TypeEnumeration, 4, func(b []byte) []byte {
		b = binary.BigEndian.AppendUint32(b, value)
		return append(b, 0, 0, 0, 0)
	})
}

func (enc *ttlvWriter) Bool(tag int, value bool) {
	enc.encodeAppend(tag, TypeBoolean, 8, func(b []byte) []byte {
		v := byte(0)
		if value {
			v = 1
		}
		return append(b, 0, 0, 0, 0, 0, 0, 0, v)
	})
}

func (enc *ttlvWriter) Struct(tag int, f func(writer)) {
	enc.writeTag(tag)
	enc.writeType(TypeStructure)
	off := len(enc.buf)
	// Placeholder for the size, will be updated once the struct is written
	// and its size is known.
	enc.writeLength(0)
	f(enc)
	length := len(enc.buf) - off - 4
	binary.BigEndian.AppendUint32(enc.buf[:off], uint32(length))
}

func (enc *ttlvWriter) TextString(tag int, str string) {
	enc.encodeAppendRightPadded(tag, TypeTextString, len(str), padForLen(len(str), 8), 0, func(b []byte) []byte {
		return append(b, str...)
	})
}

func (enc *ttlvWriter) ByteString(tag int, str []byte) {
	enc.encodeAppendRightPadded(tag, TypeByteString, len(str), padForLen(len(str), 8), 0, func(b []byte) []byte {
		return append(b, str...)
	})
}

func (enc *ttlvWriter) DateTime(tag int, date time.Time) {
	enc.encodeAppend(tag, TypeDateTime, 8, func(b []byte) []byte {
		return binary.BigEndian.AppendUint64(b, uint64(date.Unix()))
	})
}

func (enc *ttlvWriter) Interval(tag int, interval time.Duration) {
	if interval < 0 {
		panic("interval cannot be negative")
	}
	enc.encodeAppend(tag, TypeInterval, 4, func(b []byte) []byte {
		b = binary.BigEndian.AppendUint32(b, uint32(interval.Seconds()))
		return append(b, 0, 0, 0, 0)
	})
}

func (enc *ttlvWriter) Bitmask(bitmasktag, tag int, value int32) {
	enc.Integer(tag, value)
}

type ttlvReader struct {
	buf []byte
}

func newTTLVReader(buf []byte) (*ttlvReader, error) {
	dec := &ttlvReader{buf: buf}
	return dec, dec.validate()
}

func (dec *ttlvReader) Next() error {
	dec.buf = dec.buf[8+dec.paddedLen():]
	return dec.validate()
}

func (dec *ttlvReader) paddedLen() int {
	l := dec.len()
	return l + padForLen(l, 8)
}

func (dec *ttlvReader) validate() error {
	if len(dec.buf) == 0 {
		// panic("EOF")
		return nil
	}
	if len(dec.buf) < 8 {
		return Errorf("TTLV header too short")
	}
	if len(dec.buf[8:]) < dec.paddedLen() {
		return Errorf("TTLV value too short. Got %d bytes, expected %d", len(dec.buf[8:]), dec.paddedLen())
	}
	if ty := dec.Type(); ty > TypeInterval || ty == 0 {
		return Errorf("invalid TTLV type %s", ty)
	}
	// if th := (dec.Tag() >> 16) & 0xFF; th != 0x42 && th != 0x54 {
	// 	return Errorf("invalid TTLV tag %X", dec.Tag())
	// }
	return nil
}

func (dec *ttlvReader) Tag() int {
	if len(dec.buf) == 0 {
		return 0
	}
	bytes := [4]byte{0, dec.buf[0], dec.buf[1], dec.buf[2]}
	return int(binary.BigEndian.Uint32(bytes[:]))
}

func (dec *ttlvReader) Type() Type {
	if len(dec.buf) == 0 {
		return Type(0)
	}
	return Type(dec.buf[3])
}

func (dec *ttlvReader) len() int {
	if len(dec.buf) == 0 {
		return 0
	}
	return int(binary.BigEndian.Uint32(dec.buf[4:8]))
}

func (dec *ttlvReader) value() []byte {
	if len(dec.buf) == 0 {
		return nil
	}
	return dec.buf[8 : 8+dec.len()]
}

func (dec *ttlvReader) assertType(ty Type, tag int) error {
	if len(dec.buf) == 0 {
		return ErrEOF
	}
	if dec.Tag() != tag {
		return Errorf("Unexpected TTLV tag. Got %s but expected %s", TagString(dec.Tag()), TagString(tag))
	}
	if dec.Type() != ty {
		return Errorf("Invalid TTLV type for tag %s. Got %s but expected %s", TagString(tag), dec.Type(), ty)
	}
	return nil
}

func (dec *ttlvReader) Integer(tag int) (int32, error) {
	if err := dec.assertType(TypeInteger, tag); err != nil {
		return 0, err
	}
	v := int32(binary.BigEndian.Uint32(dec.value()))
	return v, dec.Next()
}

func (dec *ttlvReader) LongInteger(tag int) (int64, error) {
	if err := dec.assertType(TypeLongInteger, tag); err != nil {
		return 0, err
	}
	v := int64(binary.BigEndian.Uint64(dec.value()))
	return v, dec.Next()
}

func (dec *ttlvReader) BigInteger(tag int) (*big.Int, error) {
	v := dec.value()
	return bytesToBigInt(v), dec.Next()
}

func (dec *ttlvReader) Enum(realtag, tag int) (uint32, error) {
	if err := dec.assertType(TypeEnumeration, tag); err != nil {
		return 0, err
	}
	v := binary.BigEndian.Uint32(dec.value())
	return v, dec.Next()
}

func (dec *ttlvReader) Bool(tag int) (bool, error) {
	if err := dec.assertType(TypeBoolean, tag); err != nil {
		return false, err
	}
	v := dec.value()[7] != 0
	return v, dec.Next()
}

func (dec *ttlvReader) Struct(tag int, f func(reader) error) error {
	if err := dec.assertType(TypeStructure, tag); err != nil {
		return err
	}
	if err := f(&ttlvReader{buf: dec.value()}); err != nil {
		return err
	}
	return dec.Next()
}

func (dec *ttlvReader) TextString(tag int) (string, error) {
	if err := dec.assertType(TypeTextString, tag); err != nil {
		return "", err
	}
	// Casting to string copies the slice into a ne immutable one
	v := string(dec.value())
	return v, dec.Next()
}

func (dec *ttlvReader) ByteString(tag int) ([]byte, error) {
	if err := dec.assertType(TypeByteString, tag); err != nil {
		return nil, err
	}
	// Copy bytes
	v := slices.Clone(dec.value())
	return v, dec.Next()
}

func (dec *ttlvReader) DateTime(tag int) (time.Time, error) {
	if err := dec.assertType(TypeDateTime, tag); err != nil {
		return time.Time{}, err
	}
	v := time.Unix(int64(binary.BigEndian.Uint64(dec.value())), 0)
	return v, dec.Next()
}

func (dec *ttlvReader) Interval(tag int) (time.Duration, error) {
	if err := dec.assertType(TypeInterval, tag); err != nil {
		return 0, err
	}
	v := time.Duration(binary.BigEndian.Uint32(dec.value())) * time.Second
	return v, dec.Next()
}

func (dec *ttlvReader) Bitmask(realtag, tag int) (int32, error) {
	return dec.Integer(tag)
}
