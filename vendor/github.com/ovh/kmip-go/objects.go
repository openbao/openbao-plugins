package kmip

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ovh/kmip-go/ttlv"
)

var objectTypes = map[ObjectType]reflect.Type{
	ObjectTypeSecretData:   reflect.TypeFor[SecretData](),
	ObjectTypeCertificate:  reflect.TypeFor[Certificate](),
	ObjectTypeSymmetricKey: reflect.TypeFor[SymmetricKey](),
	ObjectTypePublicKey:    reflect.TypeFor[PublicKey](),
	ObjectTypePrivateKey:   reflect.TypeFor[PrivateKey](),
	ObjectTypeSplitKey:     reflect.TypeFor[SplitKey](),
	ObjectTypeOpaqueObject: reflect.TypeFor[OpaqueObject](),
	ObjectTypeTemplate:     reflect.TypeFor[Template](),
	ObjectTypePGPKey:       reflect.TypeFor[PGPKey](),
}

// TODO: Make it private.
func NewObjectForType(objType ObjectType) (Object, error) {
	ty, ok := objectTypes[objType]
	if !ok {
		return nil, fmt.Errorf("Invalid object type %X", objType)
	}
	return reflect.New(ty).Interface().(Object), nil
}

type Object interface {
	ObjectType() ObjectType
}

type SecretData struct {
	SecretDataType SecretDataType
	KeyBlock       KeyBlock
}

func (sd *SecretData) ObjectType() ObjectType {
	return ObjectTypeSecretData
}

func (sd *SecretData) Data() ([]byte, error) {
	switch sd.KeyBlock.KeyFormatType {
	case KeyFormatTypeRaw, KeyFormatTypeOpaque:
		return sd.KeyBlock.GetBytes()
	default:
		return nil, fmt.Errorf("Unsupported key format type %s", ttlv.EnumStr(sd.KeyBlock.KeyFormatType))
	}
}

type Certificate struct {
	CertificateType  CertificateType
	CertificateValue []byte
}

func (sd *Certificate) ObjectType() ObjectType {
	return ObjectTypeCertificate
}

func (sd *Certificate) X509Certificate() (*x509.Certificate, error) {
	if sd.CertificateType != CertificateTypeX_509 {
		return nil, fmt.Errorf("Unsupported certificate type. Got %s but want %s", ttlv.EnumStr(sd.CertificateType), ttlv.EnumStr(CertificateTypeX_509))
	}
	return x509.ParseCertificate(sd.CertificateValue)
}

// PemCertificate returns the PEM encoded value of an x509 certificate. It returns an error
// if the kmip object is not a certificate of type X509, or if the certificate data is invalid.
func (sd *Certificate) PemCertificate() (string, error) {
	cert, err := sd.X509Certificate()
	if err != nil {
		return "", err
	}
	block := pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}
	return string(pem.EncodeToMemory(&block)), nil
}

type SymmetricKey struct {
	KeyBlock KeyBlock
}

func (sd *SymmetricKey) ObjectType() ObjectType {
	return ObjectTypeSymmetricKey
}

func (sd *SymmetricKey) KeyMaterial() ([]byte, error) {
	switch sd.KeyBlock.KeyFormatType {
	case KeyFormatTypeRaw:
		return sd.KeyBlock.GetBytes()
	case KeyFormatTypeTransparentSymmetricKey:
		mat, err := sd.KeyBlock.GetMaterial()
		if err != nil {
			return nil, err
		}
		if mat.TransparentSymmetricKey == nil {
			return nil, errors.New("Empty key material")
		}
		return mat.TransparentSymmetricKey.Key, nil
	default:
		return nil, fmt.Errorf("Unsupported key format type %s", ttlv.EnumStr(sd.KeyBlock.KeyFormatType))
	}
}

type PublicKey struct {
	KeyBlock KeyBlock
}

func (sd *PublicKey) ObjectType() ObjectType {
	return ObjectTypePublicKey
}

func (key *PublicKey) RSA() (*rsa.PublicKey, error) {
	switch key.KeyBlock.KeyFormatType {
	case KeyFormatTypePKCS_1:
		raw, err := key.KeyBlock.GetBytes()
		if err != nil {
			return nil, err
		}
		return x509.ParsePKCS1PublicKey(raw)
	case KeyFormatTypeX_509:
		raw, err := key.KeyBlock.GetBytes()
		if err != nil {
			return nil, err
		}
		k, err := x509.ParsePKIXPublicKey(raw)
		if err != nil {
			return nil, err
		}
		rk, ok := k.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("SPKI key is not an RSA public key")
		}
		return rk, nil
	case KeyFormatTypeTransparentRSAPublicKey:
		// if alg := key.KeyBlock.CryptographicAlgorithm; alg == nil || *alg != RSA {
		// 	return nil, errors.New("Invalid cryptographic algorithm")
		// }
		mat, err := key.KeyBlock.GetMaterial()
		if err != nil {
			return nil, err
		}
		tkey := mat.TransparentRSAPublicKey
		if tkey == nil {
			return nil, errors.New("Empty key material")
		}
		if !tkey.PublicExponent.IsInt64() {
			return nil, errors.New("Public exponent is not an int64")
		}

		rkey := &rsa.PublicKey{
			N: &tkey.Modulus,
			E: int(tkey.PublicExponent.Int64()),
		}

		return rkey, nil
	default:
		return nil, fmt.Errorf("Unsupported key format type %s", ttlv.EnumStr(key.KeyBlock.KeyFormatType))
	}
}

func (key *PublicKey) ECDSA() (*ecdsa.PublicKey, error) {
	switch key.KeyBlock.KeyFormatType {
	case KeyFormatTypeX_509:
		raw, err := key.KeyBlock.GetBytes()
		if err != nil {
			return nil, err
		}
		k, err := x509.ParsePKIXPublicKey(raw)
		if err != nil {
			return nil, err
		}
		rk, ok := k.(*ecdsa.PublicKey)
		if !ok {
			return nil, errors.New("SPKI key is not an ECDSA public key")
		}
		return rk, nil
	case KeyFormatTypeTransparentECDSAPublicKey, KeyFormatTypeTransparentECPublicKey:
		// if alg := key.KeyBlock.CryptographicAlgorithm; alg == nil || (*alg != ECDSA && *alg != EC) {
		// 	return nil, errors.New("Invalid cryptographic algorithm")
		// }
		mat, err := key.KeyBlock.GetMaterial()
		if err != nil {
			return nil, err
		}
		tkey := (*TransparentECPublicKey)(mat.TransparentECDSAPublicKey)
		// KMIP 1.3 unified all elliptic curve keys into a single type
		if key.KeyBlock.KeyFormatType == KeyFormatTypeTransparentECPublicKey {
			tkey = mat.TransparentECPublicKey
		}
		var curve elliptic.Curve
		switch tkey.RecommendedCurve {
		case RecommendedCurveP_224:
			curve = elliptic.P224()
		case RecommendedCurveP_256:
			curve = elliptic.P256()
		case RecommendedCurveP_384:
			curve = elliptic.P384()
		case RecommendedCurveP_521:
			curve = elliptic.P521()
		default:
			return nil, fmt.Errorf("Unsupported elliptic curve %s", ttlv.EnumStr(tkey.RecommendedCurve))
		}

		rkey := &ecdsa.PublicKey{
			Curve: curve,
		}
		compressionType := KeyCompressionTypeECPublicKeyTypeUncompressed
		if key.KeyBlock.KeyCompressionType > 0 {
			compressionType = key.KeyBlock.KeyCompressionType
		}

		switch compressionType {
		case KeyCompressionTypeECPublicKeyTypeUncompressed:
			//nolint:staticcheck // We need this function compute ECDSA public key
			rkey.X, rkey.Y = elliptic.Unmarshal(curve, tkey.QString)
			if rkey.X == nil {
				return nil, errors.New("Invalid public key")
			}
		case KeyCompressionTypeECPublicKeyTypeX9_62CompressedPrime:
			rkey.X, rkey.Y = elliptic.UnmarshalCompressed(curve, tkey.QString)
			if rkey.X == nil {
				return nil, errors.New("Invalid public key")
			}
		default:
			return nil, errors.New("Invalid key compression type")
		}
		return rkey, nil
	default:
		return nil, fmt.Errorf("Unsupported key format type %s", ttlv.EnumStr(key.KeyBlock.KeyFormatType))
	}
}

// CryptoPublicKey parses and return the public key object into a go [crypto.PublicKey] object.
func (key *PublicKey) CryptoPublicKey() (crypto.PublicKey, error) {
	switch key.KeyBlock.KeyFormatType {
	case KeyFormatTypeTransparentECPublicKey, KeyFormatTypeTransparentECDSAPublicKey:
		return key.ECDSA()
	case KeyFormatTypePKCS_1, KeyFormatTypeTransparentRSAPublicKey:
		return key.RSA()
	case KeyFormatTypeX_509:
		raw, err := key.KeyBlock.GetBytes()
		if err != nil {
			return nil, err
		}
		return x509.ParsePKIXPublicKey(raw)
	default:
		return nil, errors.New("Unsupported key format")
	}
}

// PkixPem format the public key value into a PEM encoding of its PKIX, ASN.1 DER form.
// The encoded public key is a SubjectPublicKeyInfo structure
// (see RFC 5280, Section 4.1).
func (key *PublicKey) PkixPem() (string, error) {
	pubkey, err := key.CryptoPublicKey()
	if err != nil {
		return "", err
	}
	bytes, err := x509.MarshalPKIXPublicKey(pubkey)
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: bytes})), nil
}

type PrivateKey struct {
	KeyBlock KeyBlock
}

func (sd *PrivateKey) ObjectType() ObjectType {
	return ObjectTypePrivateKey
}

func (key *PrivateKey) RSA() (*rsa.PrivateKey, error) {
	switch key.KeyBlock.KeyFormatType {
	case KeyFormatTypePKCS_1:
		raw, err := key.KeyBlock.GetBytes()
		if err != nil {
			return nil, err
		}
		return x509.ParsePKCS1PrivateKey(raw)
	case KeyFormatTypePKCS_8:
		raw, err := key.KeyBlock.GetBytes()
		if err != nil {
			return nil, err
		}
		k, err := x509.ParsePKCS8PrivateKey(raw)
		if err != nil {
			return nil, err
		}
		rk, ok := k.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("PKCS8 key is not an RSA private key")
		}
		return rk, nil
	case KeyFormatTypeTransparentRSAPrivateKey:
		// if alg := key.KeyBlock.CryptographicAlgorithm; alg == nil || *alg != RSA {
		// 	return nil, errors.New("Invalid cryptographic algorithm")
		// }
		mat, err := key.KeyBlock.GetMaterial()
		if err != nil {
			return nil, err
		}
		tkey := mat.TransparentRSAPrivateKey
		if tkey == nil {
			return nil, errors.New("Empty key material")
		}

		if tkey.PublicExponent == nil {
			return nil, errors.New("Missing public exponent")
		}
		if tkey.PrivateExponent == nil {
			return nil, errors.New("Missing private exponent")
		}
		if !tkey.PublicExponent.IsInt64() {
			return nil, errors.New("Public exponent is not an int64")
		}
		//TODO: Check for other parameters nullity
		rkey := &rsa.PrivateKey{
			PublicKey: rsa.PublicKey{
				N: &tkey.Modulus,
				E: int(tkey.PublicExponent.Int64()),
			},
			D:      tkey.PrivateExponent,
			Primes: []*big.Int{tkey.P, tkey.Q},
			Precomputed: rsa.PrecomputedValues{
				Dp:        tkey.PrimeExponentP,
				Dq:        tkey.PrimeExponentQ,
				Qinv:      tkey.CRTCoefficient,
				CRTValues: []rsa.CRTValue{},
			},
		}
		rkey.Precompute()
		return rkey, nil
	default:
		return nil, fmt.Errorf("Unsupported key format type %s", ttlv.EnumStr(key.KeyBlock.KeyFormatType))
	}
}

func (key *PrivateKey) ECDSA() (*ecdsa.PrivateKey, error) {
	switch key.KeyBlock.KeyFormatType {
	case KeyFormatTypeECPrivateKey:
		raw, err := key.KeyBlock.GetBytes()
		if err != nil {
			return nil, err
		}
		return x509.ParseECPrivateKey(raw)
	case KeyFormatTypePKCS_8:
		raw, err := key.KeyBlock.GetBytes()
		if err != nil {
			return nil, err
		}
		k, err := x509.ParsePKCS8PrivateKey(raw)
		if err != nil {
			return nil, err
		}
		rk, ok := k.(*ecdsa.PrivateKey)
		if !ok {
			return nil, errors.New("PKCS8 key is not an ECDSA private key")
		}
		return rk, nil
	case KeyFormatTypeTransparentECDSAPrivateKey, KeyFormatTypeTransparentECPrivateKey:
		// if alg := key.KeyBlock.CryptographicAlgorithm; alg == nil || *alg != ECDSA {
		// 	return nil, errors.New("Invalid cryptographic algorithm")
		// }
		mat, err := key.KeyBlock.GetMaterial()
		if err != nil {
			return nil, err
		}
		tkey := (*TransparentECPrivateKey)(mat.TransparentECDSAPrivateKey)
		// KMIP 1.3 unified all elliptic curve keys into a single type
		if key.KeyBlock.KeyFormatType == KeyFormatTypeTransparentECPrivateKey {
			tkey = mat.TransparentECPrivateKey
		}

		var curve elliptic.Curve
		switch tkey.RecommendedCurve {
		case RecommendedCurveP_224:
			curve = elliptic.P224()
		case RecommendedCurveP_256:
			curve = elliptic.P256()
		case RecommendedCurveP_384:
			curve = elliptic.P384()
		case RecommendedCurveP_521:
			curve = elliptic.P521()
		default:
			return nil, fmt.Errorf("Unsupported elliptic curve %s", ttlv.EnumStr(tkey.RecommendedCurve))
		}

		rkey := &ecdsa.PrivateKey{
			PublicKey: ecdsa.PublicKey{
				Curve: curve,
			},
			D: &tkey.D,
		}
		//nolint:staticcheck // We need this function compute ECDSA public key
		rkey.X, rkey.Y = curve.ScalarBaseMult(rkey.D.Bytes())
		return rkey, nil
	default:
		return nil, fmt.Errorf("Unsupported key format type %s", ttlv.EnumStr(key.KeyBlock.KeyFormatType))
	}
}

// CryptoPrivateKey parses and return the private key object into a go [crypto.PrivateKey] object.
func (key *PrivateKey) CryptoPrivateKey() (crypto.PrivateKey, error) {
	switch key.KeyBlock.KeyFormatType {
	case KeyFormatTypeECPrivateKey, KeyFormatTypeTransparentECPrivateKey, KeyFormatTypeTransparentECDSAPrivateKey:
		return key.ECDSA()
	case KeyFormatTypePKCS_1, KeyFormatTypeTransparentRSAPrivateKey:
		return key.RSA()
	case KeyFormatTypePKCS_8:
		raw, err := key.KeyBlock.GetBytes()
		if err != nil {
			return nil, err
		}
		return x509.ParsePKCS8PrivateKey(raw)
	default:
		return nil, errors.New("Unsupported key format")
	}
}

// Pkcs8Pem format the private key into the PEM encoding of its PKCS #8, ASN.1 DER form.
func (key *PrivateKey) Pkcs8Pem() (string, error) {
	privkey, err := key.CryptoPrivateKey()
	if err != nil {
		return "", err
	}
	bytes, err := x509.MarshalPKCS8PrivateKey(privkey)
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: bytes})), nil
}

type KeyBlock struct {
	KeyFormatType          KeyFormatType
	KeyCompressionType     KeyCompressionType `ttlv:",omitempty"`
	KeyValue               *KeyValue
	CryptographicAlgorithm CryptographicAlgorithm `ttlv:",omitempty"`
	CryptographicLength    int32                  `ttlv:",omitempty"`
	KeyWrappingData        *KeyWrappingData
}

func (kb *KeyBlock) TagDecodeTTLV(d *ttlv.Decoder, tag int) error {
	return d.Struct(tag, func(d *ttlv.Decoder) error {
		if err := d.Any(&kb.KeyFormatType); err != nil {
			return err
		}
		if err := d.Opt(TagKeyCompressionType, &kb.KeyCompressionType); err != nil {
			return err
		}
		if d.Tag() == TagKeyValue {
			// KeyValue is optional and can be omitted for metadata only objects
			kb.KeyValue = new(KeyValue)
			if err := kb.KeyValue.decode(d, TagKeyValue, kb.KeyFormatType); err != nil {
				return err
			}
		}
		if err := d.Opt(TagCryptographicAlgorithm, &kb.CryptographicAlgorithm); err != nil {
			return err
		}
		if err := d.Opt(TagCryptographicLength, &kb.CryptographicLength); err != nil {
			return err
		}
		if err := d.Any(&kb.KeyWrappingData); err != nil {
			return err
		}
		return nil
	})
}

func (kb *KeyBlock) GetMaterial() (KeyMaterial, error) {
	if kb.KeyValue.Plain == nil {
		return KeyMaterial{}, errors.New("Empty key value")
	}
	return kb.KeyValue.Plain.KeyMaterial, nil
}

func (kb *KeyBlock) GetBytes() ([]byte, error) {
	mat, err := kb.GetMaterial()
	if err != nil {
		return nil, err
	}
	if mat.Bytes == nil {
		return nil, errors.New("Empty key material")
	}
	return *mat.Bytes, nil
}

func (kb *KeyBlock) GetAttributes() []Attribute {
	if kb.KeyValue.Plain == nil {
		return nil
	}
	return kb.KeyValue.Plain.Attribute
}

type KeyValue struct {
	Wrapped *[]byte
	Plain   *PlainKeyValue
}

func (kv *KeyValue) TagEncodeTTLV(e *ttlv.Encoder, tag int) {
	e.TagAny(tag, kv.Wrapped)
	e.TagAny(tag, kv.Plain)
}

func (kv *KeyValue) decode(d *ttlv.Decoder, tag int, format KeyFormatType) error {
	switch d.Type() {
	case ttlv.TypeByteString:
		kv.Wrapped = new([]byte)
		return d.TagAny(tag, &kv.Wrapped)
	case ttlv.TypeStructure:
		kv.Plain = new(PlainKeyValue)
		return kv.Plain.decode(d, format)
	}
	return fmt.Errorf("Unexpected type %s", d.Type().String())
}

type PlainKeyValue struct {
	KeyMaterial KeyMaterial
	Attribute   []Attribute
}

func (kv *PlainKeyValue) decode(d *ttlv.Decoder, format KeyFormatType) error {
	return d.Struct(TagKeyValue, func(d *ttlv.Decoder) error {
		if err := kv.KeyMaterial.decode(d, TagKeyMaterial, format); err != nil {
			return err
		}
		return d.Any(&kv.Attribute)
	})
}

type KeyMaterial struct {
	Bytes                      *[]byte
	TransparentSymmetricKey    *TransparentSymmetricKey
	TransparentRSAPrivateKey   *TransparentRSAPrivateKey
	TransparentRSAPublicKey    *TransparentRSAPublicKey
	TransparentECDSAPrivateKey *TransparentECDSAPrivateKey
	TransparentECDSAPublicKey  *TransparentECDSAPublicKey
	TransparentECPrivateKey    *TransparentECPrivateKey
	TransparentECPublicKey     *TransparentECPublicKey
}

func (km *KeyMaterial) decode(d *ttlv.Decoder, tag int, format KeyFormatType) error {
	switch format {
	case KeyFormatTypeRaw, KeyFormatTypeECPrivateKey, KeyFormatTypePKCS_1, KeyFormatTypePKCS_8, KeyFormatTypeX_509, KeyFormatTypeOpaque:
		return d.TagAny(tag, &km.Bytes)
	case KeyFormatTypeTransparentSymmetricKey:
		return d.TagAny(tag, &km.TransparentSymmetricKey)
	case KeyFormatTypeTransparentECDSAPrivateKey:
		return d.TagAny(tag, &km.TransparentECDSAPrivateKey)
	case KeyFormatTypeTransparentECDSAPublicKey:
		return d.TagAny(tag, &km.TransparentECDSAPublicKey)
	case KeyFormatTypeTransparentRSAPrivateKey:
		return d.TagAny(tag, &km.TransparentRSAPrivateKey)
	case KeyFormatTypeTransparentRSAPublicKey:
		return d.TagAny(tag, &km.TransparentRSAPublicKey)
	case KeyFormatTypeTransparentECPrivateKey:
		return d.TagAny(tag, &km.TransparentECPrivateKey)
	case KeyFormatTypeTransparentECPublicKey:
		return d.TagAny(tag, &km.TransparentECPublicKey)
	}
	return fmt.Errorf("Unsupported key format %X", format)
}

func (km *KeyMaterial) TagEncodeTTLV(e *ttlv.Encoder, tag int) {
	e.TagAny(tag, km.Bytes)
	e.TagAny(tag, km.TransparentSymmetricKey)
	e.TagAny(tag, km.TransparentRSAPrivateKey)
	e.TagAny(tag, km.TransparentRSAPublicKey)
	e.TagAny(tag, km.TransparentECDSAPrivateKey)
	e.TagAny(tag, km.TransparentECDSAPublicKey)
	e.TagAny(tag, km.TransparentECPrivateKey)
	e.TagAny(tag, km.TransparentECPublicKey)
}

type KeyWrappingData struct {
	WrappingMethod             WrappingMethod
	EncryptionKeyInformation   *EncryptionKeyInformation
	MACSignatureKeyInformation *MACSignatureKeyInformation
	MACSignature               []byte         `ttlv:",omitempty"`
	IVCounterNonce             []byte         `ttlv:",omitempty"`
	EncodingOption             EncodingOption `ttlv:",omitempty,version=v1.1.."`
}

type EncryptionKeyInformation struct {
	UniqueIdentifier        string
	CryptographicParameters *CryptographicParameters
}

type MACSignatureKeyInformation struct {
	UniqueIdentifier        string
	CryptographicParameters *CryptographicParameters
}

type TransparentSymmetricKey struct {
	Key []byte
}

type TransparentRSAPublicKey struct {
	Modulus        big.Int
	PublicExponent big.Int
}

type TransparentRSAPrivateKey struct {
	Modulus         big.Int
	PrivateExponent *big.Int
	PublicExponent  *big.Int
	P               *big.Int
	Q               *big.Int
	PrimeExponentP  *big.Int
	PrimeExponentQ  *big.Int
	CRTCoefficient  *big.Int
}

// Deprecated: deprecated in KMIP v1.3.
type TransparentECDSAPublicKey TransparentECPublicKey

// Deprecated: deprecated in KMIP v1.3.
type TransparentECDSAPrivateKey TransparentECPrivateKey

type TransparentECPrivateKey struct {
	RecommendedCurve RecommendedCurve
	D                big.Int
}

type TransparentECPublicKey struct {
	RecommendedCurve RecommendedCurve
	QString          []byte
}

type SplitKey struct {
	SplitKeyParts     int32
	KeyPartIdentifier int32
	SplitKeyThreshold int32
	SplitKeyMethod    SplitKeyMethod
	PrimeFieldSize    *big.Int
	KeyBlock          KeyBlock
}

func (sd *SplitKey) ObjectType() ObjectType {
	return ObjectTypeSplitKey
}

type OpaqueObject struct {
	OpaqueDataType  OpaqueDataType
	OpaqueDataValue []byte
}

func (sd *OpaqueObject) ObjectType() ObjectType {
	return ObjectTypeOpaqueObject
}

// Deprecated: deprecated as of KMIP 1.3.
type Template struct {
	Attribute []Attribute
}

func (sd *Template) ObjectType() ObjectType {
	return ObjectTypeTemplate
}

// KMIP 1.2.

type PGPKey struct {
	PGPKeyVersion int32
	KeyBlock      KeyBlock
}

func (sd *PGPKey) ObjectType() ObjectType {
	return ObjectTypePGPKey
}
