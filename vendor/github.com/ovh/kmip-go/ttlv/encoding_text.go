package ttlv

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"
)

type textWriter struct {
	buf    *bytes.Buffer
	indent int
}

var _ writer = (*textWriter)(nil)

func newTextWriter() *textWriter {
	return &textWriter{
		buf: new(bytes.Buffer),
	}
}

func (j *textWriter) writeIndent() {
	for range j.indent {
		j.buf.WriteString("    ")
	}
}

func (j *textWriter) startElem(ty Type, tag int) {
	if j.buf.Len() > 0 {
		j.buf.WriteByte('\n')
	}
	j.writeIndent()
	j.buf.WriteString(TagString(tag))
	j.buf.WriteString(" (")
	j.buf.WriteString(ty.String())
	j.buf.WriteString(`): `)
}

func (j *textWriter) encodeAppend(ty Type, tag int, f func([]byte) []byte) {
	j.startElem(ty, tag)
	j.buf.Write(f(j.buf.AvailableBuffer()))
}

// Bytes implements writer.
func (j *textWriter) Bytes() []byte {
	return j.buf.Bytes()
}

// Clear implements writer.
func (j *textWriter) Clear() {
	j.buf.Reset()
}

// Integer implements writer.
func (j *textWriter) Integer(tag int, value int32) {
	j.encodeAppend(TypeInteger, tag, func(b []byte) []byte {
		return strconv.AppendInt(b, int64(value), 10)
	})
}

// LongInteger implements writer.
func (j *textWriter) LongInteger(tag int, value int64) {
	j.encodeAppend(TypeLongInteger, tag, func(b []byte) []byte {
		return strconv.AppendInt(b, value, 10)
	})
}

// BigInteger implements writer.
func (j *textWriter) BigInteger(tag int, value *big.Int) {
	j.encodeAppend(TypeBigInteger, tag, func(b []byte) []byte {
		return value.Append(b, 10)
	})
}

// Bitmask implements writer.
func (j *textWriter) Bitmask(bitmasktag, tag int, value int32) {
	if bitmasktag <= 0 {
		bitmasktag = tag
	}
	j.encodeAppend(TypeInteger, tag, func(b []byte) []byte {
		return appendBitmaskString(b, bitmasktag, value, " | ")
	})
}

// Bool implements writer.
func (j *textWriter) Bool(tag int, value bool) {
	j.encodeAppend(TypeBoolean, tag, func(b []byte) []byte {
		return strconv.AppendBool(b, value)
	})
}

// ByteString implements writer.
func (j *textWriter) ByteString(tag int, str []byte) {
	j.encodeAppend(TypeByteString, tag, func(b []byte) []byte {
		// TODO: Avoid intermediate string allocation
		return append(b, strings.ToUpper(hex.EncodeToString(str))...)
	})
}

// DateTime implements writer.
func (j *textWriter) DateTime(tag int, date time.Time) {
	j.encodeAppend(TypeDateTime, tag, func(b []byte) []byte {
		return date.AppendFormat(b, time.RFC3339)
	})
}

// Enum implements writer.
func (j *textWriter) Enum(enumtag, tag int, value uint32) {
	if enumtag <= 0 {
		enumtag = tag
	}
	j.encodeAppend(TypeEnumeration, tag, func(b []byte) []byte {
		strVal := enumName(enumtag, value)
		if strVal == "" {
			return fmt.Appendf(b, "0x%08X", value)
		}
		return append(b, strVal...)
	})
}

// Interval implements writer.
func (j *textWriter) Interval(tag int, interval time.Duration) {
	if interval < 0 {
		panic("interval cannot be negative")
	}
	j.encodeAppend(TypeInterval, tag, func(b []byte) []byte {
		return append(b, interval.String()...)
	})
}

// Struct implements writer.
func (j *textWriter) Struct(tag int, f func(writer)) {
	j.startElem(TypeStructure, tag)
	oLen := j.buf.Len()
	enc := textWriter{
		buf:    j.buf,
		indent: j.indent + 1,
	}
	f(&enc)
	if oLen == j.buf.Len() {
		j.buf.WriteString("\n")
		enc.writeIndent()
		j.buf.WriteString("... empty ...")
	}
}

// TextString implements writer.
func (j *textWriter) TextString(tag int, str string) {
	j.encodeAppend(TypeTextString, tag, func(b []byte) []byte {
		return append(b, str...)
	})
}
