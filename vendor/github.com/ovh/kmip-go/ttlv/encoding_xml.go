package ttlv

import (
	"bytes"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"
	"time"
)

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}

type xmlWriter struct {
	w   *xml.Encoder
	buf *bytes.Buffer
}

func newXMLWriter() *xmlWriter {
	buf := new(bytes.Buffer)
	enc := xml.NewEncoder(buf)
	enc.Indent("", "    ")
	return &xmlWriter{enc, buf}
}

func (enc *xmlWriter) Clear() {
	panicOnErr(enc.w.Close())
	enc.buf.Reset()
	enc.w = xml.NewEncoder(enc.buf)
	enc.w.Indent("", "    ")
}

func (enc *xmlWriter) Bytes() []byte {
	panicOnErr(enc.w.Flush())
	return enc.buf.Bytes()
}

func (enc *xmlWriter) startElement(ty Type, tag int) xml.StartElement {
	start := xml.StartElement{Name: xml.Name{Local: "TTLV"}}
	if tagName := getTagName(tag); tagName != "" {
		start.Name.Local = tagName
	} else {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "tag"}, Value: fmt.Sprintf("0x%06X", uint(tag))})
	}
	if ty != TypeStructure {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "type"}, Value: ty.String()})
	}
	return start
}

func (enc *xmlWriter) encode(ty Type, tag int, value string) {
	elem := enc.startElement(ty, tag)
	elem.Attr = append(elem.Attr, xml.Attr{Name: xml.Name{Local: "value"}, Value: value})
	panicOnErr(enc.w.EncodeToken(elem))

	// Below code is a hack to write self-closing tag. Golang's stdlib XML
	// does not support writing self-closing tag, but KMIP spec seems to require them.
	// So we just write the closing tag, then we modify the end of the output buffer
	// to transform it into a self-closing tag.
	panicOnErr(enc.w.Flush())
	l := enc.buf.Len()
	panicOnErr(enc.w.EncodeToken(elem.End()))
	panicOnErr(enc.w.Flush())
	enc.buf.Truncate(l - 1)
	if _, err := enc.buf.WriteString("/>"); err != nil {
		panic(err)
	}
}

func (enc *xmlWriter) Integer(tag int, value int32) {
	enc.encode(TypeInteger, tag, strconv.Itoa(int(value)))
}

func (enc *xmlWriter) LongInteger(tag int, value int64) {
	enc.encode(TypeLongInteger, tag, strconv.Itoa(int(value)))
}

func (enc *xmlWriter) BigInteger(tag int, value *big.Int) {
	bytes, pad, padLen := bigIntToBytes(value, 1)
	val := bytes
	if padLen > 0 {
		val = make([]byte, padLen, len(bytes)+padLen)
		if pad != 0 {
			for i := range val {
				val[i] = pad
			}
		}
		val = append(val, bytes...)
	}
	enc.encode(TypeBigInteger, tag, strings.ToUpper(hex.EncodeToString(val)))
}

func (enc *xmlWriter) Enum(enumtag, tag int, value uint32) {
	if enumtag <= 0 {
		enumtag = tag
	}
	strVal := enumName(enumtag, value)
	if strVal == "" {
		strVal = fmt.Sprintf("0x%08X", value)
	}
	enc.encode(TypeEnumeration, tag, strVal)
}

func (enc *xmlWriter) Bool(tag int, value bool) {
	enc.encode(TypeBoolean, tag, strconv.FormatBool(value))
}

func (enc *xmlWriter) Struct(tag int, f func(writer)) {
	start := enc.startElement(TypeStructure, tag)
	panicOnErr(enc.w.EncodeToken(start))
	f(enc)
	panicOnErr(enc.w.EncodeToken(start.End()))
	panicOnErr(enc.w.Flush())
}

func (enc *xmlWriter) TextString(tag int, str string) {
	enc.encode(TypeTextString, tag, str)
}

func (enc *xmlWriter) ByteString(tag int, str []byte) {
	enc.encode(TypeByteString, tag, strings.ToUpper(hex.EncodeToString(str)))
}

func (enc *xmlWriter) DateTime(tag int, date time.Time) {
	enc.encode(TypeDateTime, tag, date.Format(time.RFC3339))
}

func (enc *xmlWriter) Interval(tag int, interval time.Duration) {
	if interval < 0 {
		panic("interval cannot be negative")
	}
	enc.encode(TypeInterval, tag, strconv.Itoa(int(interval.Seconds())))
}

func (enc *xmlWriter) Bitmask(bitmasktag, tag int, value int32) {
	if bitmasktag <= 0 {
		bitmasktag = tag
	}
	enc.encode(TypeInteger, tag, bitmaskString(bitmasktag, value, " "))
}

type xmlReader struct {
	r    *xml.Decoder
	elem *xml.StartElement
}

func newXMLReaderFromDecoder(r *xml.Decoder) (*xmlReader, error) {
	dec := &xmlReader{
		r,
		nil,
	}
	return dec, dec.Next()
}

func newXMLReader(data []byte) (*xmlReader, error) {
	return newXMLReaderFromDecoder(xml.NewDecoder(bytes.NewReader(data)))
}

func (dec *xmlReader) Next() error {
	if ty := dec.Type(); ty != Type(0) && ty != TypeStructure {
		if err := dec.r.Skip(); err != nil {
			return err
		}
	}
	for {
		tok, err := dec.r.Token()
		if err != nil {
			if err == io.EOF && dec.elem != nil {
				dec.elem = nil
				return nil
			}
			return err
		}
		switch elem := tok.(type) {
		case xml.StartElement:
			dec.elem = &elem
			return nil
		case xml.EndElement:
			dec.elem = nil
			return nil
		}
	}
}

func (dec *xmlReader) value() string {
	for _, attr := range dec.elem.Attr {
		if attr.Name.Local == "value" {
			return attr.Value
		}
	}
	return ""
}

func (dec *xmlReader) rawTag() string {
	if dec.elem == nil {
		return ""
	}
	name := dec.elem.Name.Local
	if name != "TTLV" {
		return name
	}
	for _, attr := range dec.elem.Attr {
		if attr.Name.Local == "tag" {
			return attr.Value
		}
	}
	return ""
}

func (dec *xmlReader) Tag() int {
	rawTag := dec.rawTag()
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
		return 0
	}
	return tg
}

func (dec *xmlReader) Type() Type {
	if dec.elem == nil {
		return 0
	}
	for _, attr := range dec.elem.Attr {
		if attr.Name.Local == "type" {
			if ty, ok := typeFromName(attr.Value); ok {
				return ty
			}
			//TODO: return error
			panic("Invalid type")
		}
	}
	return TypeStructure
}

func (dec *xmlReader) assertType(ty Type, tag int) error {
	if dec.elem == nil {
		return ErrEOF
	}
	if dec.Tag() != tag {
		//TODO: Add details
		return Errorf("Unexpected TTLV tag. Got %q but expected %s", dec.rawTag(), TagString(tag))
	}
	if dec.Type() != ty {
		//TODO: Add details
		return Errorf("Invalid TTLV type for tag %s. Got %s but expected %s", TagString(tag), dec.Type(), ty)
	}
	return nil
}

func (dec *xmlReader) Integer(tag int) (int32, error) {
	if err := dec.assertType(TypeInteger, tag); err != nil {
		return 0, err
	}
	parsed, err := parseInt(dec.value(), 32)
	if err != nil {
		return 0, err
	}
	return int32(parsed), dec.Next()
}

func (dec *xmlReader) LongInteger(tag int) (int64, error) {
	if err := dec.assertType(TypeLongInteger, tag); err != nil {
		return 0, err
	}
	parsed, err := parseInt(dec.value(), 64)
	if err != nil {
		return 0, err
	}
	return parsed, dec.Next()
}

func (dec *xmlReader) BigInteger(tag int) (*big.Int, error) {
	if err := dec.assertType(TypeBigInteger, tag); err != nil {
		return nil, err
	}
	bytes, err := hex.DecodeString(dec.value())
	if err != nil {
		return nil, err
	}
	return bytesToBigInt(bytes), dec.Next()
}

func (dec *xmlReader) Enum(realtag, tag int) (uint32, error) {
	if err := dec.assertType(TypeEnumeration, tag); err != nil {
		return 0, err
	}
	if realtag <= 0 {
		realtag = tag
	}
	val := dec.value()
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
	return uint32(parsed), dec.Next()
}

func (dec *xmlReader) Bool(tag int) (bool, error) {
	if err := dec.assertType(TypeBoolean, tag); err != nil {
		return false, err
	}
	b, err := strconv.ParseBool(dec.value())
	if err != nil {
		return b, err
	}
	return b, dec.Next()
}

func (dec *xmlReader) Struct(tag int, f func(reader) error) error {
	if err := dec.assertType(TypeStructure, tag); err != nil {
		return err
	}
	subDec := xmlReader{dec.r, nil}
	if err := subDec.Next(); err != nil {
		return err
	}
	if err := f(&subDec); err != nil {
		return err
	}
	for subDec.elem != nil {
		if err := subDec.Next(); err != nil {
			return err
		}
	}
	return dec.Next()
}

func (dec *xmlReader) TextString(tag int) (string, error) {
	if err := dec.assertType(TypeTextString, tag); err != nil {
		return "", err
	}
	return dec.value(), dec.Next()
}

func (dec *xmlReader) ByteString(tag int) ([]byte, error) {
	if err := dec.assertType(TypeByteString, tag); err != nil {
		return nil, err
	}
	bytes, err := hex.DecodeString(dec.value())
	if err != nil {
		return nil, err
	}
	return bytes, dec.Next()
}

func (dec *xmlReader) DateTime(tag int) (time.Time, error) {
	if err := dec.assertType(TypeDateTime, tag); err != nil {
		return time.Time{}, err
	}
	dt, err := time.Parse(time.RFC3339, dec.value())
	if err != nil {
		return time.Time{}, err
	}
	return dt.Local(), dec.Next()
}

func (dec *xmlReader) Interval(tag int) (time.Duration, error) {
	if err := dec.assertType(TypeInterval, tag); err != nil {
		return 0, err
	}
	parsed, err := parseUint(dec.value(), 32)
	if err != nil {
		return 0, err
	}
	return time.Duration(parsed) * time.Second, dec.Next()
}

func (dec *xmlReader) Bitmask(realtag, tag int) (int32, error) {
	if err := dec.assertType(TypeInteger, tag); err != nil {
		return 0, err
	}
	if realtag <= 0 {
		realtag = tag
	}
	strVal := dec.value()
	parts := strings.Fields(strVal)
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
	return result, dec.Next()
}
