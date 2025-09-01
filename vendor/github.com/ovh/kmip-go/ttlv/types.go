package ttlv

import "fmt"

// Type is a TTLV encoding type as defined in the [KMIP 1.4 specification, section 9.1.1.2]
// for TTLV encoding.
//
// [KMIP 1.4 specification, section 9.1.1.2]: http://docs.oasis-open.org/kmip/spec/v1.4/os/kmip-spec-v1.4-os.html#_Toc490660914
type Type uint8

const (
	TypeStructure Type = 0x01 + iota
	TypeInteger
	TypeLongInteger
	TypeBigInteger
	TypeEnumeration
	TypeBoolean
	TypeTextString
	TypeByteString
	TypeDateTime
	TypeInterval
)

func (ty Type) String() string {
	if n, ok := typesName[ty]; ok {
		return n
	}
	return fmt.Sprintf("Unknown(%02X)", uint8(ty))
}

// typeFromName returns the type for the given normalized name string.
// It returns (0, false) if the name is not valid, otherwise it
// returns the type and true.
//
// The function is case sensitive, and the type mus mathc the camel case as
// defined in the standard.
func typeFromName(name string) (Type, bool) {
	ty, ok := nameTypes[name]
	return ty, ok
}

var (
	typesName = map[Type]string{
		TypeStructure:   "Structure",
		TypeInteger:     "Integer",
		TypeLongInteger: "LongInteger",
		TypeBigInteger:  "BigInteger",
		TypeEnumeration: "Enumeration",
		TypeBoolean:     "Boolean",
		TypeTextString:  "TextString",
		TypeByteString:  "ByteString",
		TypeDateTime:    "DateTime",
		TypeInterval:    "Interval",
	}
	nameTypes = revMap(typesName)
)
