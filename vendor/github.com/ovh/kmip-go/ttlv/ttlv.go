// Package ttlv implements the TTLV serialization and deserialization as defined in the Oasis [KMIP 1.4 specification, section 9.1].
// It also supports other encoding format like
//   - XML as defined in [KMIP 1.4 Profiles, section 5.4]
//   - JSON as defined in [KMIP 1.4 Profiles, section 5.5]
//   - Non standard "human friendly" textual format
//
// This package is the low level part of KMIP protocol serialization and should mostly never be used directly except to extend the kmip protocol implementation
// or to perform testing or optimizing stuff.
//
// [KMIP 1.4 Profiles, section 5.4]: http://docs.oasis-open.org/kmip/profiles/v1.4/os/kmip-profiles-v1.4-os.html#_Toc491431437
// [KMIP 1.4 specification, section 9.1]: http://docs.oasis-open.org/kmip/spec/v1.4/os/kmip-spec-v1.4-os.html#_Toc490660911
// [KMIP 1.4 Profiles, section 5.5]: http://docs.oasis-open.org/kmip/profiles/v1.4/os/kmip-profiles-v1.4-os.html#_Toc491431461
package ttlv

import (
	"errors"
	"fmt"
)

// ErrEncoding is the error type returned when decoding data fails.
type ErrEncoding struct {
	cause error
}

// ErrEOF is an ErrEncoding instance returned whensome data are missing while decoding.
var ErrEOF = Errorf("unexpected end of data")

// Errorf creates a new ErrEncoding.
func Errorf(format string, args ...any) error {
	return ErrEncoding{
		cause: fmt.Errorf(format, args...),
	}
}

func (err ErrEncoding) Error() string {
	return fmt.Sprintf("encoding error: %s", err.cause)
}

func (err ErrEncoding) Unwrap() error {
	return err.cause
}

// IsErrEncoding returns true if err is of type or wraps
// and error of type ErrEncoding.
func IsErrEncoding(err error) bool {
	var e ErrEncoding
	return errors.As(err, &e)
}

// MarshalTTLV serializes `data` into a binary TTLV encoded byte array.
func MarshalTTLV(data any) []byte {
	enc := NewTTLVEncoder()
	enc.Any(data)
	return enc.Bytes()
}

// MarshalXML serializes `data` into an xml TTLV encoded byte string.
func MarshalXML(data any) []byte {
	enc := NewXMLEncoder()
	enc.Any(data)
	return enc.Bytes()
}

// MarshalJSON serializes `data` into a json TTLV encoded byte string.
func MarshalJSON(data any) []byte {
	enc := NewJSONEncoder()
	enc.Any(data)
	return enc.Bytes()
}

// MarshalText serializes `data` into a textual and human-friendly form
// of TTLV. Useful mainly for debugging.
func MarshalText(data any) []byte {
	enc := NewTextEncoder()
	enc.Any(data)
	return enc.Bytes()
}

// UnmarshalTTLV deserializes the binary TTLV byte string into the object pointed by `ptr`.
//
// `ptr` must be a pointer.
func UnmarshalTTLV(data []byte, ptr any) error {
	dec, err := NewTTLVDecoder(data)
	if err != nil {
		return err
	}
	return dec.Any(ptr)
}

// UnmarshalXML deserializes the xml TTLV byte string into the object pointed by `ptr`.
//
// `ptr` must be a pointer.
func UnmarshalXML(data []byte, ptr any) error {
	dec, err := NewXMLDecoder(data)
	if err != nil {
		return err
	}
	return dec.Any(ptr)
}

// UnmarshalJSON deserializes the json TTLV byte string into the object pointed by `ptr`.
//
// `ptr` must be a pointer.
func UnmarshalJSON(data []byte, ptr any) error {
	dec, err := NewJSONDecoder(data)
	if err != nil {
		return err
	}
	return dec.Any(ptr)
}

// TagEncodable is implemented by types implementing a custom serialization
// logic, instead of relying on reflection.
type TagEncodable interface {
	TagEncodeTTLV(e *Encoder, tag int)
}

// TODO: Do we still need that ?
type Encodable interface {
	EncodeTTLV(e *Encoder)
}

// TagEncodable is implemented by types implementing a custom deserialization
// logic, instead of relying on reflection.
type TagDecodable interface {
	TagDecodeTTLV(d *Decoder, tag int) error
}

// TODO: Do we still need that ?
type Decodable interface {
	DecodeTTLV(d *Decoder) error
}
