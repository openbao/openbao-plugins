package ttlv

import (
	"reflect"
	"strconv"
	"strings"
)

type fieldInfo struct {
	tag        string
	omitempty  bool
	vrange     *versionRange
	setVersion bool
}

func getFieldInfo(fldT reflect.StructField) fieldInfo {
	tagVal, _ := fldT.Tag.Lookup("ttlv")
	return parseFieldInfo(tagVal)
}

func parseFieldInfo(s string) fieldInfo {
	parts := strings.Split(s, ",")
	ann := fieldInfo{tag: parts[0]}

	for _, part := range parts[1:] {
		if part == "omitempty" {
			ann.omitempty = true
			continue
		}
		if part == "set-version" {
			ann.setVersion = true
			continue
		}
		parts := strings.Split(part, "=")
		if len(parts) != 2 {
			panic("invalid sub-tag " + part)
		}
		if parts[0] == "version" {
			vrange, err := parseVersionRange(parts[1])
			if err != nil {
				panic("Invalid sub-tag version range: " + err.Error())
			}
			ann.vrange = &vrange
			continue
		}
		panic("invalid sub-tag " + part)
	}

	return ann
}

func getFieldTag(fldT reflect.StructField, tagVal string) int {
	if tagVal == "" {
		// if fldT.Type.Implements(reflect.TypeFor[Encodable]()) {
		// 	// FIXME: How to pass a custom tag if any ?
		// 	fieldsEncode = append(fieldsEncode, func(e *Encoder, v reflect.Value) {
		// 		if encodable := v.Field(i).Interface(); encodable != nil {
		// 			encodable.(Encodable).EncodeTTLV(e)
		// 		}
		// 	})
		// 	continue
		// }
		if tg, err := getTagByName(fldT.Name); err == nil {
			// Check if we already know a tag with the same name as the field
			return tg
		} else if tg, err := getTagForType(fldT.Type); err == nil {
			// if not check if we know the default tag for this type (either explicitly registered, or fallback to type name)
			return tg
		}
		return 0
	}

	if strings.HasPrefix(tagVal, "0x") {
		n, err := strconv.ParseInt(tagVal[2:], 16, 64)
		if err != nil {
			panic(err)
		}
		return int(n)
	}

	numTag, err := getTagByName(tagVal)
	if err != nil {
		panic(err)
	}
	return numTag
}
