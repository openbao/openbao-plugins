package payloads

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"fmt"

	"github.com/ovh/kmip-go"
	"github.com/ovh/kmip-go/ttlv"
)

func init() {
	kmip.RegisterOperationPayload[GetRequestPayload, GetResponsePayload](kmip.OperationGet)
}

// This operation requests that the server returns the Managed Object specified by its Unique Identifier.
//
// Only a single object is returned. The response contains the Unique Identifier of the object, along with the object itself,
// which MAY be wrapped using a wrapping key as specified in the request.
//
// The following key format capabilities SHALL be assumed by the client; restrictions apply when the client  requests the server to return an object in a particular format:
//   - If a client registered a key in a given format, the server SHALL be able to return the key during the Get operation in the same format that was used when the key was registered.
//   - Any other format conversion MAY be supported by the server.
//
// If Key Format Type is specified to be PKCS#12 then the response payload SHALL be a PKCS#12 container as specified by [RFC7292].
// The Unique Identifier SHALL be that of a private key to be included in the response.  The container SHALL be protected using the Secret Data
// object specified via the private key’s Secret Data Link.  The current certificate chain SHALL also be included as determined by using
// the private key’s Public Key link to get the corresponding public key, and then using that public key’s Certificate Link to get the base certificate,
// and then using each certificate’s Certificate Link to build the certificate chain.  It is an error if there is more than one valid certificate chain.
type GetRequestPayload struct {
	// Determines the object being requested. If omitted, then the ID Placeholder value is used by the server as the Unique Identifier.
	UniqueIdentifier string `ttlv:",omitempty"`
	// Determines the key format type to be returned.
	KeyFormatType kmip.KeyFormatType `ttlv:",omitempty"`
	// Determines the Key Wrap Type of the returned key value.
	KeyWrapType kmip.KeyFormatType `ttlv:",omitempty,version=v1.4.."`
	// Determines the compression method for elliptic curve public keys.
	KeyCompressionType kmip.KeyCompressionType `ttlv:",omitempty"`
	// Specifies keys and other information for wrapping the returned object. This field SHALL NOT be specified if the requested object is a Template.
	KeyWrappingSpecification *kmip.KeyWrappingSpecification
}

func (pl *GetRequestPayload) Operation() kmip.Operation {
	return kmip.OperationGet
}

// Response for the Get operation.
type GetResponsePayload struct {
	// Type of object.
	ObjectType kmip.ObjectType
	// The Unique Identifier of the object.
	UniqueIdentifier string
	// The object being returned.
	Object kmip.Object
}

func (pl *GetResponsePayload) Operation() kmip.Operation {
	return kmip.OperationGet
}

func (pl *GetResponsePayload) TagDecodeTTLV(d *ttlv.Decoder, tag int) error {
	return d.Struct(tag, func(d *ttlv.Decoder) error {
		if err := d.Any(&pl.ObjectType); err != nil {
			return err
		}
		if err := d.TagAny(kmip.TagUniqueIdentifier, &pl.UniqueIdentifier); err != nil {
			return err
		}

		var err error
		if pl.Object, err = kmip.NewObjectForType(pl.ObjectType); err != nil {
			return err
		}
		return d.Any(&pl.Object)
	})
}

func (pl *GetResponsePayload) SecretString() (string, error) {
	sec, err := pl.Secret()
	if err != nil {
		return "", err
	}
	return string(sec), nil
}

func (pl *GetResponsePayload) Secret() ([]byte, error) {
	if pl.ObjectType != kmip.ObjectTypeSecretData {
		return nil, fmt.Errorf("Invalid object type. Got %s but want %s", ttlv.EnumStr(pl.ObjectType), ttlv.EnumStr(kmip.ObjectTypeSecretData))
	}
	secret := pl.Object.(*kmip.SecretData)
	return secret.Data()
}

func (pl *GetResponsePayload) SymmetricKey() ([]byte, error) {
	if pl.ObjectType != kmip.ObjectTypeSymmetricKey {
		return nil, fmt.Errorf("Invalid object type. Got %s but want %s", ttlv.EnumStr(pl.ObjectType), ttlv.EnumStr(kmip.ObjectTypeSymmetricKey))
	}
	key := pl.Object.(*kmip.SymmetricKey)
	return key.KeyMaterial()
}

func (pl *GetResponsePayload) X509Certificate() (*x509.Certificate, error) {
	if pl.ObjectType != kmip.ObjectTypeCertificate {
		return nil, fmt.Errorf("Invalid object type. Got %s but want %s", ttlv.EnumStr(pl.ObjectType), ttlv.EnumStr(kmip.ObjectTypeCertificate))
	}
	cert := pl.Object.(*kmip.Certificate)
	return cert.X509Certificate()
}

// PemCertificate returns the PEM encoded value of an x509 certificate. It returns an error
// if the kmip object is not a certificate of type X509, or if the certificate data is invalid.
func (pl *GetResponsePayload) PemCertificate() (string, error) {
	if pl.ObjectType != kmip.ObjectTypeCertificate {
		return "", fmt.Errorf("Invalid object type. Got %s but want %s", ttlv.EnumStr(pl.ObjectType), ttlv.EnumStr(kmip.ObjectTypeCertificate))
	}
	cert := pl.Object.(*kmip.Certificate)
	return cert.PemCertificate()
}

func (pl *GetResponsePayload) RsaPrivateKey() (*rsa.PrivateKey, error) {
	if pl.ObjectType != kmip.ObjectTypePrivateKey {
		return nil, fmt.Errorf("Invalid object type. Got %s but want %s", ttlv.EnumStr(pl.ObjectType), ttlv.EnumStr(kmip.ObjectTypePrivateKey))
	}
	key := pl.Object.(*kmip.PrivateKey)
	return key.RSA()
}

func (pl *GetResponsePayload) EcdsaPrivateKey() (*ecdsa.PrivateKey, error) {
	if pl.ObjectType != kmip.ObjectTypePrivateKey {
		return nil, fmt.Errorf("Invalid object type. Got %s but want %s", ttlv.EnumStr(pl.ObjectType), ttlv.EnumStr(kmip.ObjectTypePrivateKey))
	}
	key := pl.Object.(*kmip.PrivateKey)
	return key.ECDSA()
}

// PrivateKey parses and return the private key object into a go [crypto.PrivateKey] object.
func (pl *GetResponsePayload) PrivateKey() (crypto.PrivateKey, error) {
	if pl.ObjectType != kmip.ObjectTypePrivateKey {
		return nil, fmt.Errorf("Invalid object type. Got %s but want %s", ttlv.EnumStr(pl.ObjectType), ttlv.EnumStr(kmip.ObjectTypePrivateKey))
	}
	return pl.Object.(*kmip.PrivateKey).CryptoPrivateKey()
}

// PemPrivateKey format the private key into the PEM encoding of its PKCS #8, ASN.1 DER form.
func (pl *GetResponsePayload) PemPrivateKey() (string, error) {
	if pl.ObjectType != kmip.ObjectTypePrivateKey {
		return "", fmt.Errorf("Invalid object type. Got %s but want %s", ttlv.EnumStr(pl.ObjectType), ttlv.EnumStr(kmip.ObjectTypePrivateKey))
	}
	return pl.Object.(*kmip.PrivateKey).Pkcs8Pem()
}

func (pl *GetResponsePayload) RsaPublicKey() (*rsa.PublicKey, error) {
	if pl.ObjectType != kmip.ObjectTypePublicKey {
		return nil, fmt.Errorf("Invalid object type. Got %s but want %s", ttlv.EnumStr(pl.ObjectType), ttlv.EnumStr(kmip.ObjectTypePublicKey))
	}
	key := pl.Object.(*kmip.PublicKey)
	return key.RSA()
}

func (pl *GetResponsePayload) EcdsaPublicKey() (*ecdsa.PublicKey, error) {
	if pl.ObjectType != kmip.ObjectTypePublicKey {
		return nil, fmt.Errorf("Invalid object type. Got %s but want %s", ttlv.EnumStr(pl.ObjectType), ttlv.EnumStr(kmip.ObjectTypePublicKey))
	}
	key := pl.Object.(*kmip.PublicKey)
	return key.ECDSA()
}

// PublicKey parses and return the public key object into a go [crypto.PublicKey] object.
func (pl *GetResponsePayload) PublicKey() (crypto.PublicKey, error) {
	if pl.ObjectType != kmip.ObjectTypePublicKey {
		return nil, fmt.Errorf("Invalid object type. Got %s but want %s", ttlv.EnumStr(pl.ObjectType), ttlv.EnumStr(kmip.ObjectTypePublicKey))
	}
	return pl.Object.(*kmip.PublicKey).CryptoPublicKey()
}

// PemPublicKey format the public key value into a PEM encoding of its PKIX, ASN.1 DER form.
// The encoded public key is a SubjectPublicKeyInfo structure
// (see RFC 5280, Section 4.1).
func (pl *GetResponsePayload) PemPublicKey() (string, error) {
	if pl.ObjectType != kmip.ObjectTypePublicKey {
		return "", fmt.Errorf("Invalid object type. Got %s but want %s", ttlv.EnumStr(pl.ObjectType), ttlv.EnumStr(kmip.ObjectTypePublicKey))
	}
	return pl.Object.(*kmip.PublicKey).PkixPem()
}
