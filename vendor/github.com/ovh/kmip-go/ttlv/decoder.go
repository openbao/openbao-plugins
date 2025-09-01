package ttlv

import (
	"encoding/xml"
	"fmt"
	"math/big"
	"reflect"
	"sync"
	"time"
)

type reader interface {
	Next() error
	Tag() int
	Type() Type
	Integer(tag int) (int32, error)
	LongInteger(tag int) (int64, error)
	BigInteger(tag int) (*big.Int, error)
	Enum(realtag, tag int) (uint32, error)
	Bool(tag int) (bool, error)
	Struct(tag int, f func(reader) error) error
	TextString(tag int) (string, error)
	ByteString(tag int) ([]byte, error)
	DateTime(tag int) (time.Time, error)
	Interval(tag int) (time.Duration, error)
	Bitmask(realtag, tag int) (int32, error)
}

// Decoder exposes methods to read TTLV tagged values to an internal buffer.
// It supports multiple formats like binary TTLV or xml TTLV.
type Decoder struct {
	*extension
	r reader
}

// NewTTLVDecoder create a new [Decoder] to decode values from
// the binary TTLV format.
func NewTTLVDecoder(bytes []byte) (Decoder, error) {
	r, err := newTTLVReader(bytes)
	if err != nil {
		return Decoder{}, err
	}
	return newDecoder(r), nil
}

// NewXMLDecoder create a new [Decoder] to decode values from
// the xml TTLV format.
func NewXMLDecoder(bytes []byte) (Decoder, error) {
	r, err := newXMLReader(bytes)
	if err != nil {
		return Decoder{}, err
	}
	return newDecoder(r), nil
}

// NewXMLDecoder create a new [Decoder] to decode xml TTLV values from
// the given [encoding/xml.Decoder].
func NewXMLFromDecoder(dec *xml.Decoder) (Decoder, error) {
	r, err := newXMLReaderFromDecoder(dec)
	if err != nil {
		return Decoder{}, err
	}
	return newDecoder(r), nil
}

// NewJSONDecoder create a new [Decoder] to decode values from
// the json TTLV format.
func NewJSONDecoder(bytes []byte) (Decoder, error) {
	r, err := newJSONReader(bytes)
	if err != nil {
		return Decoder{}, err
	}
	return newDecoder(r), nil
}

func newDecoder(r reader) Decoder {
	return Decoder{new(extension), r}
}

// Next advances to the next TTLV value.
func (dec *Decoder) Next() error {
	return dec.r.Next()
}

// Tag returns the tag of the current TTLV value being decoded.
// It returns 0 in case of EOF.
func (dec *Decoder) Tag() int {
	return dec.r.Tag()
}

// Tag returns the type of the current TTLV value being decoded.
// It returns 0 in case of EOF.
func (dec *Decoder) Type() Type {
	return dec.r.Type()
}

// Integer decodes an integer and advance to the next value.
func (dec *Decoder) Integer(tag int) (int32, error) {
	return dec.r.Integer(tag)
}

// LongInteger decodes a long integer and advance to the next value.
func (dec *Decoder) LongInteger(tag int) (int64, error) {
	return dec.r.LongInteger(tag)
}

// BigInteger decodes a big integer and advance to the next value.
func (dec *Decoder) BigInteger(tag int) (*big.Int, error) {
	return dec.r.BigInteger(tag)
}

// Enum reads an enum from the internal buffer.
// While `tag` is the tag to write with the value, which may differ from the enum's default tag,
// `realtag` can optionally be set to non-zero to identify the real default tag associated to the enum type.
// It's useful for deserializing the enum value from its text representation.
func (dec *Decoder) Enum(realtag, tag int) (uint32, error) {
	return dec.r.Enum(realtag, tag)
}

// Bool reads a boolean  to the internal buffer.
func (dec *Decoder) Bool(tag int) (bool, error) {
	return dec.r.Bool(tag)
}

// Struct reads a structure to the internal buffer.
// It calls the provided callback `f` with a Decoder to use for reading
// struct's fields.
func (dec *Decoder) Struct(tag int, f func(*Decoder) error) error {
	return dec.r.Struct(tag, func(r reader) error {
		return f(&Decoder{dec.extension, r})
	})
}

// TextString reads a string and advance to the next value.
func (dec *Decoder) TextString(tag int) (string, error) {
	return dec.r.TextString(tag)
}

// ByteString reads a byte array and advance to the next value.
func (dec *Decoder) ByteString(tag int) ([]byte, error) {
	return dec.r.ByteString(tag)
}

// DateTime reads a date-time and advance to the next value.
func (dec *Decoder) DateTime(tag int) (time.Time, error) {
	return dec.r.DateTime(tag)
}

// DateTime reads a duration and advance to the next value.
func (dec *Decoder) Interval(tag int) (time.Duration, error) {
	return dec.r.Interval(tag)
}

// Bitmaks reads a bitmask value from the internal buffer.
// While `tag` is the tag to write with the value, which may differ from the bitmask's default tag,
// `realtag` can optionally be set to non-zero to identify the real default tag associated to the bitmask type.
// It's useful for deserializing the bitmask value from its text representation.
func (dec *Decoder) Bitmask(realtag, tag int) (int32, error) {
	return dec.r.Bitmask(realtag, tag)
}

// TagAny decodes `value` by deserializing it from the buffer with the given tag instead of value's type default one.
// It panics if value's type cannot be encoded or if it's not a pointer.
func (dec *Decoder) TagAny(tag int, value any) (err error) {
	switch v := value.(type) {
	case *byte:
		var x int32
		x, err = dec.Integer(tag)
		*v = byte(x)
	case *int8:
		var x int32
		x, err = dec.Integer(tag)
		*v = int8(x)
	case *int16:
		var x int32
		x, err = dec.Integer(tag)
		*v = int16(x)
	case *int32:
		*v, err = dec.Integer(tag)
	case *int64:
		*v, err = dec.LongInteger(tag)
	case *bool:
		*v, err = dec.Bool(tag)
	case *string:
		*v, err = dec.TextString(tag)
	case *[]byte:
		*v, err = dec.ByteString(tag)
	case *time.Duration:
		*v, err = dec.Interval(tag)
	case *time.Time:
		*v, err = dec.DateTime(tag)
	case **big.Int:
		*v, err = dec.BigInteger(tag)
	case TagDecodable:
		return v.TagDecodeTTLV(dec, tag)
	default:
		return dec.decodeValue(tag, reflect.ValueOf(value))
	}
	return err
}

// Any decodes `value` by deserializing it from the buffer using value's type default tag.
// It panics if no tag can be found for `value` or if value does not implement [Encodable].
// `value` must be a pointer.
func (dec *Decoder) Any(value any) error {
	switch v := value.(type) {
	case Decodable:
		return v.DecodeTTLV(dec)
	default:
		tag, err := getTagForValue(reflect.ValueOf(value))
		if err != nil {
			panic(err)
		}
		return dec.TagAny(tag, value)
	}
}

// Opt optionally decodes a value if the current tag match the one given in parameter.
// If not, the deserialization is skipped.
//
// gotcha: When skipped, the passed value is not zeroed.
func (dec *Decoder) Opt(tag int, value any) error {
	if dec.Tag() == tag {
		return dec.TagAny(tag, value)
	}
	return nil
}

func (d *Decoder) decodeValue(tag int, value reflect.Value) error {
	if value.Kind() != reflect.Pointer {
		panic(fmt.Errorf("value must be a pointer, but got %q (tag: %s)", value.Kind(), TagString(tag)))
	}
	value = value.Elem()
	f := decodeFuncFor(value.Type())
	return f(d, tag, value)
}

var decodeFuncsCache = new(sync.Map)

func decodeFuncFor(ty reflect.Type) func(d *Decoder, tag int, value reflect.Value) error {
	if f, ok := decodeFuncsCache.Load(ty); ok {
		return f.(func(d *Decoder, tag int, value reflect.Value) error)
	}
	f := decodeFunc(ty)
	decodeFuncsCache.Store(ty, f)
	return f
}

func decodeFunc(ty reflect.Type) func(d *Decoder, tag int, value reflect.Value) error {
	if ty.Kind() != reflect.Interface && ty.Implements(reflect.TypeFor[TagDecodable]()) {
		return func(d *Decoder, tag int, v reflect.Value) error {
			if v.Kind() == reflect.Pointer {
				if d.Tag() != tag {
					v.SetZero()
					return nil
				}
				if v.IsNil() {
					v.Set(reflect.New(ty.Elem()))
				}
			}
			return v.Interface().(TagDecodable).TagDecodeTTLV(d, tag)
		}
	}
	if reflect.PointerTo(ty).Implements(reflect.TypeFor[TagDecodable]()) {
		return func(d *Decoder, tag int, v reflect.Value) error {
			if v.Kind() == reflect.Interface && v.IsNil() {
				v.SetZero()
				return nil
			}
			if !v.CanAddr() {
				panic(ty.Name() + " Implements ttlv.Encodable but its value cannot be addressed")
			}
			return v.Addr().Interface().(TagDecodable).TagDecodeTTLV(d, tag)
		}
	}

	if isEnum(ty) {
		return func(d *Decoder, tag int, value reflect.Value) error {
			enumtag, _ := getTagForType(ty)
			v, err := d.Enum(enumtag, tag)
			if err != nil {
				return err
			}
			value.SetUint(uint64(v))
			return nil
		}
	}

	if isBitmask(ty) {
		return func(d *Decoder, tag int, value reflect.Value) error {
			bitmasktag, _ := getTagForType(ty)
			v, err := d.Bitmask(bitmasktag, tag)
			if err != nil {
				return err
			}
			value.SetInt(int64(v))
			return nil
		}
	}

	switch ty {
	case reflect.TypeFor[time.Duration]():
		return func(d *Decoder, tag int, value reflect.Value) error {
			v, err := d.Interval(tag)
			if err != nil {
				return err
			}
			value.SetInt(int64(v))
			return nil
		}
	case reflect.TypeFor[time.Time]():
		return func(d *Decoder, tag int, value reflect.Value) error {
			v, err := d.DateTime(tag)
			if err != nil {
				return err
			}
			value.Set(reflect.ValueOf(v))
			return nil
		}
	case reflect.TypeFor[big.Int]():
		return func(d *Decoder, tag int, value reflect.Value) error {
			bi, err := d.BigInteger(tag)
			if err != nil {
				return err
			}
			value.Set(reflect.ValueOf(*bi))
			return nil
		}
	}

	switch ty.Kind() {
	case reflect.Pointer:
		for ty.Kind() == reflect.Pointer {
			ty = ty.Elem()
		}
		f := decodeFuncFor(ty)
		return func(d *Decoder, tag int, value reflect.Value) error {
			if d.Tag() != tag {
				value.SetZero()
				return nil
			}
			for value.Kind() == reflect.Pointer {
				if value.IsNil() {
					value.Set(reflect.New(value.Type().Elem()))
				}
				value = value.Elem()
			}
			return f(d, tag, value)
		}

	case reflect.Uint8, reflect.Uint16:
		return func(d *Decoder, tag int, value reflect.Value) error {
			v, err := d.Integer(tag)
			if err != nil {
				return err
			}
			value.SetUint(uint64(v))
			return nil
		}
	case reflect.Uint32, reflect.Uint64:
		return func(d *Decoder, tag int, value reflect.Value) error {
			v, err := d.LongInteger(tag)
			if err != nil {
				return err
			}
			value.SetUint(uint64(v))
			return nil
		}
	case reflect.Int8, reflect.Int16, reflect.Int32:
		return func(d *Decoder, tag int, value reflect.Value) error {
			v, err := d.Integer(tag)
			if err != nil {
				return err
			}
			value.SetInt(int64(v))
			return nil
		}
	case reflect.Int64:
		return func(d *Decoder, tag int, value reflect.Value) error {
			v, err := d.LongInteger(tag)
			if err != nil {
				return err
			}
			value.SetInt(v)
			return nil
		}
	case reflect.Bool:
		return func(d *Decoder, tag int, value reflect.Value) error {
			v, err := d.Bool(tag)
			if err != nil {
				return err
			}
			value.SetBool(v)
			return nil
		}
	case reflect.String:
		return func(d *Decoder, tag int, value reflect.Value) error {
			v, err := d.TextString(tag)
			if err != nil {
				return err
			}
			value.SetString(v)
			return nil
		}
	case reflect.Slice:
		if ty.Elem().Kind() == reflect.Uint8 {
			return func(d *Decoder, tag int, value reflect.Value) error {
				v, err := d.ByteString(tag)
				if err != nil {
					return err
				}
				value.SetBytes(v)
				return nil
			}
		}
		// 	fallthrough
		// case reflect.Array:
		elemTy := ty.Elem()
		ff := decodeFuncFor(reflect.PointerTo(elemTy))
		return func(d *Decoder, tag int, value reflect.Value) error {
			for d.Tag() == tag {
				elem := reflect.New(elemTy)
				if err := ff(d, tag, elem); err != nil {
					return err
				}
				value.Set(reflect.Append(value, elem.Elem()))
			}
			return nil
		}
	case reflect.Struct:
		return structDecodeFunc(ty)
	case reflect.Interface:
		return func(d *Decoder, tag int, value reflect.Value) error {
			return d.decodeValue(tag, value.Elem())
		}
	default:
		panic("Unsupported type: " + ty.String())
	}
}

func structDecodeFunc(ty reflect.Type) func(d *Decoder, tag int, value reflect.Value) error {
	fieldsDecode := []func(d *Decoder, value reflect.Value) error{}

	for i := 0; i < ty.NumField(); i++ {
		fldT := ty.Field(i)
		if !fldT.IsExported() {
			continue
		}

		info := getFieldInfo(fldT)
		if info.tag == "-" {
			continue
		}

		numTag := getFieldTag(fldT, info.tag)

		if numTag == 0 {
			if fldT.Type.Kind() == reflect.Interface {
				fieldsDecode = append(fieldsDecode, func(d *Decoder, value reflect.Value) error {
					value = value.Field(i)
					tag, err := getTagForType(value.Elem().Type())
					if err != nil {
						return err
					}
					return d.decodeValue(tag, value)
				})
				continue
			}
			panic(fmt.Sprintf("Missing tag for field %s of type %s", fldT.Name, ty.Name()))
		}
		ffunc := decodeFuncFor(fldT.Type)
		if info.omitempty {
			ff := ffunc
			ffunc = func(d *Decoder, i int, v reflect.Value) error {
				if d.Tag() != i {
					v.SetZero()
					return nil
				}
				return ff(d, i, v)
			}
		}
		if info.vrange != nil {
			ff := ffunc
			ffunc = func(d *Decoder, i int, v reflect.Value) error {
				// If the field is not for current version, consider it optional
				// (but still accept and decode it if it's present)
				if !d.versionIn(*info.vrange) && d.Tag() != i {
					v.SetZero()
					return nil
				}
				return ff(d, i, v)
			}
		}
		if info.setVersion {
			// Check that field type implements Version interface (major / minor)
			if !fldT.Type.Implements(reflect.TypeFor[Version]()) {
				panic(fmt.Sprintf("Type %s does not implement ttlv.Version", fldT.Type.String()))
			}
			ff := ffunc
			ffunc = func(d *Decoder, i int, v reflect.Value) error {
				if err := ff(d, i, v); err != nil {
					return err
				}
				d.setVersion(v.Interface().(Version))
				return nil
			}
		}
		fieldsDecode = append(fieldsDecode, func(d *Decoder, value reflect.Value) error {
			return ffunc(d, numTag, value.Field(i))
		})
	}

	return func(d *Decoder, tag int, value reflect.Value) error {
		return d.Struct(tag, func(d *Decoder) error {
			for _, fd := range fieldsDecode {
				if err := fd(d, value); err != nil {
					return err
				}
			}
			return nil
		})
	}
}
