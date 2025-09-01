package ttlv

import (
	"fmt"
	"iter"
	"reflect"
)

var (
	tagByName = map[string]int{}
	tagByType = map[reflect.Type]int{}
	tagNames  = map[int]string{}

	enums       = map[reflect.Type]int{}
	enumNames   = map[int]map[uint32]string{}
	enumsByName = map[int]map[string]uint32{}

	bitmasks      = map[reflect.Type]int{}
	bitmaskNames  = map[int][]string{}
	bitmaskByName = map[int]map[string]int32{}
)

func getTagByName(name string) (int, error) {
	if tag, ok := tagByName[name]; ok {
		return tag, nil
	}
	return 0, fmt.Errorf("Unknown tag %q", name)
}

func getTagName(tag int) string {
	n := tagNames[tag]
	return n
}

func TagString(tag int) string {
	if name, ok := tagNames[tag]; ok {
		return name
	}
	return fmt.Sprintf("0x%06X", uint(tag))
}

func getTagForType(ty reflect.Type) (int, error) {
	for ty.Kind() == reflect.Pointer {
		ty = ty.Elem()
	}
	if ty.Kind() == reflect.Array || ty.Kind() == reflect.Slice {
		return getTagForType(ty.Elem())
	}
	if tag, ok := tagByType[ty]; ok {
		return tag, nil
	}
	if tag, ok := tagByName[ty.Name()]; ok {
		return tag, nil
	}
	return 0, fmt.Errorf("No default tag found for type %q", ty.Name())
}

func getTagForValue(val reflect.Value) (int, error) {
	tag, err := getTagForType(val.Type())
	if err != nil {
		for val.Kind() == reflect.Pointer {
			val = val.Elem()
		}
		if val.Kind() == reflect.Interface {
			tag, err = getTagForType(val.Elem().Type())
		}
	}
	return tag, err
}

// RegisterTag registers a new named tag and save the mapping between name and interger value for later usage
// by encoder and decoders.
// An optional list of types can be provided to also register a mapping between the types and the default tag used
// for serializing/deserializing them.
func RegisterTag(name string, value int, ty ...reflect.Type) {
	tagByName[name] = value
	tagNames[value] = name
	for _, t := range ty {
		tagByType[t] = value
	}
}

// RegisterEnum registers an enum tag with its tag and all its string values.
func RegisterEnum[T ~uint32](tag int, names map[T]string) {
	//TODO: Merge new registration with previous registrations. Panic on overwrite
	ty := reflect.TypeFor[T]()
	enums[ty] = tag
	tagByType[ty] = tag

	if names == nil {
		return
	}

	if enumNames[tag] == nil {
		enumNames[tag] = make(map[uint32]string, len(names))
	}
	if enumsByName[tag] == nil {
		enumsByName[tag] = make(map[string]uint32, len(names))
	}
	for enum, name := range names {
		enumNames[tag][uint32(enum)] = name
		enumsByName[tag][name] = uint32(enum)
	}
}

// EnumValues returns an iterator over registered enum values and names
// for the given type T. If T is not a known enum, then the iterator will
// yield an empty result set.
func EnumValues[T ~uint32]() iter.Seq2[T, string] {
	tag, ok := enums[reflect.TypeFor[T]()]
	if !ok {
		return func(yield func(T, string) bool) {}
	}
	return func(yield func(T, string) bool) {
		for id, name := range enumNames[tag] {
			if !yield(T(id), name) {
				return
			}
		}
	}
}

func isEnum(ty reflect.Type) bool {
	_, ok := enums[ty]
	return ok
}

func enumName(tag int, value uint32) string {
	if reg := enumNames[tag]; reg != nil {
		n := reg[value]
		return n
	}
	return ""
}

// EnumStr returns the string representation of an enum. If it's known,
// the string is the normalized name, otherwise it's the 0x prefixed hex value.
func EnumStr[T ~uint32](value T) string {
	if tag := enums[reflect.TypeFor[T]()]; tag != 0 {
		name := enumName(tag, uint32(value))
		return name
	}
	return fmt.Sprintf("0x%08X", uint32(value))
}

func enumByName(tag int, name string) (uint32, error) {
	if reg := enumsByName[tag]; reg != nil {
		n := reg[name]
		return n, nil
	}
	return 0, fmt.Errorf("Unknown enum value %q", name)
}

// RegisterBitmask registers a bitmask types with it's tag and string values.
func RegisterBitmask[T ~int32](tag int, names ...string) {
	ty := reflect.TypeFor[T]()
	tagByType[ty] = tag
	bitmasks[reflect.TypeFor[T]()] = tag
	bitmaskNames[tag] = names
	bitmaskByName[tag] = make(map[string]int32, len(names))
	for i, name := range names {
		bitmaskByName[tag][name] = 1 << i
	}
}

// BitmaskStr returns the string representation of a bitmask value, consisting of
// a concatenation of all the flags values separated by `sep`. If it's known,
// the flag string value is the normalized name, otherwise it's the 0x prefixed hex value.
func BitmaskStr[T ~int32](value T, sep string) string {
	return bitmaskString(bitmasks[reflect.TypeFor[T]()], value, sep)
}

func bitmaskString[T ~int32](tag int, value T, sep string) string {
	return string(appendBitmaskString([]byte{}, tag, value, sep))
}

func appendBitmaskString[T ~int32](dst []byte, tag int, value T, sep string) []byte {
	if value == 0 {
		return dst
	}
	bsep := []byte(sep)
	mapper := bitmaskNames[tag]
	wrote := false
	for i := 0; i < 32; i++ {
		v := int32(value & (1 << i))
		if v == 0 {
			continue
		}
		if wrote {
			dst = append(dst, bsep...)
		}
		wrote = true
		if i < len(mapper) {
			dst = append(dst, []byte(mapper[i])...)
			continue
		}
		// Handle case where it's not registered
		// by writing 0x prefixed hex value
		dst = fmt.Appendf(dst, "0x%08X", uint32(v))
	}
	return dst
}

func bitmaskByStr(tag int, name string) (int32, error) {
	if reg := bitmaskByName[tag]; reg != nil {
		n := reg[name]
		return n, nil
	}
	return 0, fmt.Errorf("Unknown bitmask value %q", name)
}

func isBitmask(ty reflect.Type) bool {
	_, ok := bitmasks[ty]
	return ok
}
