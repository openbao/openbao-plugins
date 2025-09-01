package ttlv

import (
	"fmt"
	"math/big"
	"time"
)

// Enum is a generic TTLV enum value.
type Enum uint32

// Struct is a generic TTLV struct container.
type Struct []Value

func (v Struct) TagEncodeTTLV(e *Encoder, tag int) {
	e.Struct(tag, func(e *Encoder) {
		for _, f := range v {
			f.EncodeTTLV(e)
		}
	})
}

func (v *Struct) TagDecodeTTLV(d *Decoder, tag int) error {
	return d.Struct(tag, func(d *Decoder) error {
		for d.Tag() != 0 {
			field := Value{}
			if err := field.DecodeTTLV(d); err != nil {
				return err
			}
			*v = append(*v, field)
		}
		return nil
	})
}

// Value is a generic TTLV tagged value.
type Value struct {
	// The value's TTLV tag.
	Tag int
	// The TTLV value.
	Value any
}

func (v *Value) DecodeTTLV(d *Decoder) error {
	return v.TagDecodeTTLV(d, d.Tag())
}

func (v *Value) TagDecodeTTLV(d *Decoder, tag int) error {
	var err error
	ty := d.Type()
	switch ty {
	case TypeInteger:
		v.Value, err = d.Integer(tag)
	case TypeLongInteger:
		v.Value, err = d.LongInteger(tag)
	case TypeBigInteger:
		v.Value, err = d.BigInteger(tag)
	case TypeBoolean:
		v.Value, err = d.Bool(tag)
	case TypeByteString:
		v.Value, err = d.ByteString(tag)
	case TypeDateTime:
		v.Value, err = d.DateTime(tag)
	case TypeEnumeration:
		var enum uint32
		enum, err = d.Enum(0, tag)
		v.Value = Enum(enum)
	case TypeInterval:
		v.Value, err = d.Interval(tag)
	case TypeTextString:
		v.Value, err = d.TextString(tag)
	case TypeStructure:
		val := Struct{}
		err = val.TagDecodeTTLV(d, tag)
		v.Value = val
	default:
		return fmt.Errorf("Unsupported TTLV type %s", ty.String())
	}
	if err != nil {
		return err
	}
	v.Tag = tag
	return nil
}

func (v Value) EncodeTTLV(e *Encoder) {
	v.TagEncodeTTLV(e, v.Tag)
}

func (v Value) TagEncodeTTLV(e *Encoder, tag int) {
	switch val := v.Value.(type) {
	case int32:
		e.Integer(tag, val)
	case int64:
		e.LongInteger(tag, val)
	case *big.Int:
		e.BigInteger(tag, val)
	case bool:
		e.Bool(tag, val)
	case []byte:
		e.ByteString(tag, val)
	case time.Time:
		e.DateTime(tag, val)
	case Enum:
		e.Enum(0, tag, uint32(val))
	case time.Duration:
		e.Interval(tag, val)
	case string:
		e.TextString(tag, val)
	case Struct:
		val.TagEncodeTTLV(e, tag)
	default:
		panic(fmt.Sprintf("Unsupported type %T", val))
	}
}
