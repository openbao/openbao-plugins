package ttlv

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
)

// JS Numbers are 64-bit floating point and can only represent 53-bits of precision,
// so any number values >= 2^52 must be represented as hex strings.
const (
	maxJsonInt int64 = 4503599627370496 // 2^52
	minJsonInt int64 = -maxJsonInt
)

var (
	maxJsonBigInt *big.Int = big.NewInt(maxJsonInt)
	minJsonBigInt *big.Int = big.NewInt(minJsonInt)
)

type jsonWriter struct {
	buf    *bytes.Buffer
	indent int
}

var _ writer = (*jsonWriter)(nil)

func newJSONWriter() *jsonWriter {
	return &jsonWriter{
		buf: new(bytes.Buffer),
	}
}

func (j *jsonWriter) writeIndent() {
	for range j.indent {
		j.buf.WriteString("    ")
	}
}

func (j *jsonWriter) startElem(ty Type, tag int) {
	j.writeIndent()
	j.buf.WriteByte('{')
	j.buf.WriteString(`"tag": "`)
	j.buf.WriteString(TagString(tag))
	if ty != TypeStructure {
		j.buf.WriteString(`", "type": "`)
		j.buf.WriteString(ty.String())
	}
	j.buf.WriteString(`", "value": `)
}

func (j *jsonWriter) endElem() {
	j.buf.WriteByte('}')
	if j.indent > 0 {
		j.buf.WriteString(",\n")
	}
}

func (j *jsonWriter) encodeAppend(ty Type, tag int, f func([]byte) []byte) {
	j.startElem(ty, tag)
	j.buf.Write(f(j.buf.AvailableBuffer()))
	j.endElem()
}

// Bytes implements writer.
func (j *jsonWriter) Bytes() []byte {
	return j.buf.Bytes()
}

// Clear implements writer.
func (j *jsonWriter) Clear() {
	j.buf.Reset()
}

// Integer implements writer.
func (j *jsonWriter) Integer(tag int, value int32) {
	j.encodeAppend(TypeInteger, tag, func(b []byte) []byte {
		return strconv.AppendInt(b, int64(value), 10)
	})
}

// LongInteger implements writer.
func (j *jsonWriter) LongInteger(tag int, value int64) {
	if value >= maxJsonInt || value <= minJsonInt {
		// Any values >= 2^52 must be represented as hex strings
		j.encodeAppend(TypeLongInteger, tag, func(b []byte) []byte {
			return fmt.Appendf(b, "\"0x%016x\"", uint64(value))
		})
		return
	}
	j.encodeAppend(TypeLongInteger, tag, func(b []byte) []byte {
		return strconv.AppendInt(b, value, 10)
	})
}

// BigInteger implements writer.
func (j *jsonWriter) BigInteger(tag int, value *big.Int) {
	if value.Cmp(maxJsonBigInt) >= 0 || value.Cmp(minJsonBigInt) <= 0 {
		// Any values >= 2^52 must be represented as hex strings
		bytes, padval, padlen := bigIntToBytes(value, 8)
		j.encodeAppend(TypeBigInteger, tag, func(b []byte) []byte {
			b = append(b, "\"0x"...)
			pad := [...]byte{padval}
			for range padlen {
				b = hex.AppendEncode(b, pad[:])
			}
			b = hex.AppendEncode(b, bytes)
			return append(b, '"')
		})
		return
	}
	j.encodeAppend(TypeBigInteger, tag, func(b []byte) []byte {
		return value.Append(b, 10)
	})
}

// Bitmask implements writer.
func (j *jsonWriter) Bitmask(bitmasktag, tag int, value int32) {
	if bitmasktag <= 0 {
		bitmasktag = tag
	}
	j.encodeAppend(TypeInteger, tag, func(b []byte) []byte {
		b = append(b, '"')
		b = appendBitmaskString(b, bitmasktag, value, "|")
		return append(b, '"')
	})
}

// Bool implements writer.
func (j *jsonWriter) Bool(tag int, value bool) {
	j.encodeAppend(TypeBoolean, tag, func(b []byte) []byte {
		return strconv.AppendBool(b, value)
	})
}

// ByteString implements writer.
func (j *jsonWriter) ByteString(tag int, str []byte) {
	j.encodeAppend(TypeByteString, tag, func(b []byte) []byte {
		b = append(b, '"')
		// TODO: Avoid intermediate string allocation
		b = append(b, strings.ToUpper(hex.EncodeToString(str))...)
		return append(b, '"')
	})
}

// DateTime implements writer.
func (j *jsonWriter) DateTime(tag int, date time.Time) {
	j.encodeAppend(TypeDateTime, tag, func(b []byte) []byte {
		b = append(b, '"')
		b = date.AppendFormat(b, time.RFC3339)
		return append(b, '"')
	})
}

// Enum implements writer.
func (j *jsonWriter) Enum(enumtag, tag int, value uint32) {
	if enumtag <= 0 {
		enumtag = tag
	}
	j.encodeAppend(TypeEnumeration, tag, func(b []byte) []byte {
		strVal := enumName(enumtag, value)
		if strVal == "" {
			return fmt.Appendf(b, "\"0x%08X\"", value)
		}
		return strconv.AppendQuote(b, strVal)
	})
}

// Interval implements writer.
func (j *jsonWriter) Interval(tag int, interval time.Duration) {
	if interval < 0 {
		panic("interval cannot be negative")
	}
	dur := int64(interval.Seconds())
	j.encodeAppend(TypeInterval, tag, func(b []byte) []byte {
		return strconv.AppendInt(b, dur, 10)
	})
}

// Struct implements writer.
func (j *jsonWriter) Struct(tag int, f func(writer)) {
	j.startElem(TypeStructure, tag)
	j.buf.WriteString("[\n")
	originalLen := j.buf.Len()
	enc := jsonWriter{
		buf:    j.buf,
		indent: j.indent + 1,
	}
	f(&enc)
	if j.buf.Len() > originalLen {
		j.buf.Truncate(j.buf.Len() - 2)
		j.buf.WriteByte('\n')
		j.writeIndent()
		j.buf.WriteByte(']')
	} else {
		j.buf.Truncate(j.buf.Len() - 1)
		j.buf.WriteString("]")
	}
	j.endElem()
}

// TextString implements writer.
func (j *jsonWriter) TextString(tag int, str string) {
	j.encodeAppend(TypeTextString, tag, func(b []byte) []byte {
		return strconv.AppendQuote(b, str)
	})
}

type jsonReader struct {
	value   []any
	current map[string]any
}

var _ reader = (*jsonReader)(nil)

func newJSONReader(data []byte) (*jsonReader, error) {
	r := &jsonReader{}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	r.value = append(r.value, v)
	return r, nil
}

// Next implements reader.
func (j *jsonReader) Next() error {
	if len(j.value) == 0 {
		return ErrEOF
	}
	j.value = j.value[1:]
	j.current = nil
	return nil
}

func (j *jsonReader) getMap() map[string]any {
	if len(j.value) == 0 {
		//TODO: Return error
		return nil
	}
	if j.current != nil {
		return j.current
	}
	j.current = j.value[0].(map[string]any)
	return j.current
}

// Type implements reader.
func (j *jsonReader) Type() Type {
	//TODO: Check that type is a string, and return error if not
	typ, _ := j.getMap()["type"].(string)
	if typ == "" {
		return TypeStructure
	}
	if ty, ok := typeFromName(typ); ok {
		return ty
	}
	//TODO: return error
	panic("Invalid type")
}

// Tag implements reader.
func (j *jsonReader) Tag() int {
	if len(j.value) == 0 {
		//TODO: Return error
		return 0
	}
	//TODO: Check that type is a string, and return error if not
	rawTag, _ := j.getMap()["tag"].(string)
	if rawTag == "" {
		// TODO: return error
		return 0
	}
	if strings.HasPrefix(rawTag, "0x") {
		parsedTag, err := strconv.ParseInt(rawTag[2:], 16, 32)
		if err != nil {
			// TODO: return error
			return 0
		}
		return int(parsedTag)
	}
	tg, err := getTagByName(rawTag)
	if err != nil {
		// TODO: return error
		return 0
	}
	return tg
}

func (j *jsonReader) getValue() any {
	if len(j.value) == 0 {
		//TODO: Return error
		return nil
	}
	return j.getMap()["value"]
}

func (j *jsonReader) assertType(ty Type, tag int) error {
	if len(j.value) == 0 {
		return ErrEOF
	}
	if j.Tag() != tag {
		//TODO: Add details
		return Errorf("Unexpected TTLV tag. Got %q but expected %s", j.Tag(), TagString(tag))
	}
	if j.Type() != ty {
		//TODO: Add details
		return Errorf("Invalid TTLV type for tag %s. Got %s but expected %s", TagString(tag), j.Type(), ty)
	}
	return nil
}

// Integer implements reader.
func (j *jsonReader) Integer(tag int) (int32, error) {
	if err := j.assertType(TypeInteger, tag); err != nil {
		return 0, err
	}
	switch val := j.getValue().(type) {
	case json.Number:
		n, err := val.Int64()
		if err != nil {
			return 0, err
		}
		// Check integer bounds
		if n > math.MaxInt32 {
			return 0, Errorf("integer is out of bound")
		}
		return int32(n), j.Next()
	case string:
		parsed, err := parseInt(val, 32)
		if err != nil {
			return 0, err
		}
		// Check integer bounds
		if parsed > math.MaxInt32 {
			return 0, Errorf("integer is out of bound")
		}
		return int32(parsed), j.Next()
	default:
		return 0, Errorf("Invalid integer  value %q", val)
	}
}

// LongInteger implements reader.
func (j *jsonReader) LongInteger(tag int) (int64, error) {
	if err := j.assertType(TypeLongInteger, tag); err != nil {
		return 0, err
	}
	switch val := j.getValue().(type) {
	case json.Number:
		n, err := val.Int64()
		if err != nil {
			return 0, err
		}
		return n, j.Next()
	case string:
		parsed, err := parseInt(val, 64)
		if err != nil {
			return 0, err
		}
		return parsed, j.Next()
	default:
		return 0, Errorf("invalid long integer value %q", val)
	}
}

// BigInteger implements reader.
func (j *jsonReader) BigInteger(tag int) (*big.Int, error) {
	if err := j.assertType(TypeBigInteger, tag); err != nil {
		return nil, err
	}
	switch val := j.getValue().(type) {
	case json.Number:
		n, err := val.Int64()
		if err != nil {
			return nil, err
		}
		return big.NewInt(n), j.Next()
	case string:
		if !strings.HasPrefix(val, "0x") {
			return nil, Errorf("invalid big integer value %q", val)
		}
		bytes, err := hex.DecodeString(val[2:])
		if err != nil {
			return nil, err
		}
		return bytesToBigInt(bytes), j.Next()
	default:
		return nil, Errorf("invalid big integer value %q", val)
	}
}

// Enum implements reader.
func (j *jsonReader) Enum(realtag, tag int) (uint32, error) {
	if err := j.assertType(TypeEnumeration, tag); err != nil {
		return 0, err
	}
	if realtag <= 0 {
		realtag = tag
	}
	switch val := j.getValue().(type) {
	case json.Number:
		n, err := val.Int64()
		if err != nil {
			return 0, err
		}
		// Check integer bounds
		if n > math.MaxUint32 {
			return 0, Errorf("integer is out of bound")
		}
		return uint32(n), j.Next()
	case string:
		var parsed uint64
		var err error
		if strings.HasPrefix(val, "0x") {
			parsed, err = strconv.ParseUint(val[2:], 16, 32)
		} else {
			parsed, err = strconv.ParseUint(val, 10, 32)
			if err != nil {
				var p uint32
				p, err = enumByName(realtag, val)
				parsed = uint64(p)
			}
		}
		if err != nil {
			return 0, err
		}
		return uint32(parsed), j.Next()
	default:
		return 0, Errorf("invalid enum value: %q", val)
	}
}

// Bool implements reader.
func (j *jsonReader) Bool(tag int) (bool, error) {
	if err := j.assertType(TypeBoolean, tag); err != nil {
		return false, err
	}
	switch b := j.getValue().(type) {
	case bool:
		return b, j.Next()
	case string:
		parsed, err := parseInt(b, 64)
		if err != nil {
			return false, err
		}
		return parsed != 0, j.Next()
	default:
		return false, Errorf("invalid boolean value")
	}
}

// TextString implements reader.
func (j *jsonReader) TextString(tag int) (string, error) {
	if err := j.assertType(TypeTextString, tag); err != nil {
		return "", err
	}
	if s, ok := j.getValue().(string); ok {
		return s, j.Next()
	}
	return "", Errorf("invalid text string value")
}

// ByteString implements reader.
func (j *jsonReader) ByteString(tag int) ([]byte, error) {
	if err := j.assertType(TypeByteString, tag); err != nil {
		return nil, err
	}
	if s, ok := j.getValue().(string); ok {
		bytes, err := hex.DecodeString(s)
		if err != nil {
			return nil, err
		}
		return bytes, j.Next()
	}
	return nil, Errorf("invalid byte string value")
}

// DateTime implements reader.
func (j *jsonReader) DateTime(tag int) (time.Time, error) {
	if err := j.assertType(TypeDateTime, tag); err != nil {
		return time.Time{}, err
	}
	switch val := j.getValue().(type) {
	case string:
		if strings.HasPrefix(val, "0x") {
			parsed, err := strconv.ParseUint(val[2:], 10, 64)
			if err != nil {
				return time.Time{}, err
			}
			epoch := int64(parsed)
			if epoch < 0 {
				return time.Time{}, Errorf("date-time cannot be negative")
			}
			return time.Unix(epoch, 0).UTC(), j.Next()
		}
		t, err := time.Parse(time.RFC3339, val)
		if err != nil {
			return t, err
		}
		return t.Local(), j.Next()
	default:
		return time.Time{}, Errorf("invalid date-time value: %q", val)
	}
}

// Interval implements reader.
func (j *jsonReader) Interval(tag int) (time.Duration, error) {
	if err := j.assertType(TypeInterval, tag); err != nil {
		return 0, err
	}
	switch val := j.getValue().(type) {
	case json.Number:
		n, err := val.Int64()
		if err != nil {
			return 0, err
		}
		return time.Duration(n) * time.Second, j.Next()
	case string:
		parsed, err := parseUint(val, 32)
		if err != nil {
			return 0, err
		}
		// Check integer bounds
		if parsed > math.MaxInt64 {
			return 0, Errorf("integer is out of bound")
		}
		return time.Duration(parsed), j.Next()
	default:
		return 0, Errorf("Invalid interval value %q", val)
	}
}

// Struct implements reader.
func (j *jsonReader) Struct(tag int, f func(reader) error) error {
	if err := j.assertType(TypeStructure, tag); err != nil {
		return err
	}
	st, ok := j.getValue().([]any)
	if !ok {
		return Errorf("Invalid structure data layout")
	}
	if err := f(&jsonReader{value: st}); err != nil {
		return err
	}
	return j.Next()
}

// Bitmask implements reader.
func (j *jsonReader) Bitmask(realtag, tag int) (int32, error) {
	if err := j.assertType(TypeInteger, tag); err != nil {
		return 0, err
	}
	if realtag <= 0 {
		realtag = tag
	}
	switch val := j.getValue().(type) {
	case json.Number:
		n, err := val.Int64()
		if err != nil {
			return 0, err
		}
		// Check integer bounds
		if n > math.MaxInt32 {
			return 0, Errorf("integer is out of bound")
		}
		return int32(n), j.Next()
	case string:
		parts := strings.Split(val, "|")
		result := int32(0)
		for _, part := range parts {
			part = strings.TrimSpace(part)
			var parsed int64
			var err error
			if strings.HasPrefix(part, "0x") {
				parsed, err = strconv.ParseInt(part[2:], 16, 32)
			} else {
				parsed, err = strconv.ParseInt(part, 10, 32)
				if err != nil {
					// Look for the name
					var p int32
					p, err = bitmaskByStr(realtag, part)
					parsed = int64(p)
				}
			}
			if err != nil {
				return 0, err
			}
			result |= int32(parsed)
		}
		return result, j.Next()
	default:
		return 0, Errorf("Invalid bitmask value %q", val)
	}
}
