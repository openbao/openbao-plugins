package kmip

import (
	"reflect"
	"strings"
	"time"

	"github.com/ovh/kmip-go/ttlv"
)

type AttributeName string

func (atn AttributeName) IsCustom() bool {
	n := string(atn)
	return strings.HasPrefix(n, "x-") || strings.HasPrefix(n, "y-")
}

const (
	AttributeNameUniqueIdentifier AttributeName = "Unique Identifier"
	AttributeNameName             AttributeName = "Name"
	AttributeNameObjectType       AttributeName = "Object Type"
	// Deprecated: deprecated as of kmip 1.3.
	AttributeNameOperationPolicyName           AttributeName = "Operation Policy Name"
	AttributeNameObjectGroup                   AttributeName = "Object Group"
	AttributeNameContactInformation            AttributeName = "Contact Information"
	AttributeNameInitialDate                   AttributeName = "Initial Date"
	AttributeNameActivationDate                AttributeName = "Activation Date"
	AttributeNameProcessStartDate              AttributeName = "Process Start Date"
	AttributeNameProtectStopDate               AttributeName = "Protect Stop Date"
	AttributeNameDeactivationDate              AttributeName = "Deactivation Date"
	AttributeNameDestroyDate                   AttributeName = "Destroy Date"
	AttributeNameCompromiseOccurrenceDate      AttributeName = "Compromise Occurrence Date"
	AttributeNameCompromiseDate                AttributeName = "Compromise Date"
	AttributeNameArchiveDate                   AttributeName = "Archive Date"
	AttributeNameLastChangeDate                AttributeName = "Last Change Date"
	AttributeNameCryptographicLength           AttributeName = "Cryptographic Length"
	AttributeNameLeaseTime                     AttributeName = "Lease Time"
	AttributeNameCryptographicAlgorithm        AttributeName = "Cryptographic Algorithm"
	AttributeNameCryptographicParameters       AttributeName = "Cryptographic Parameters"
	AttributeNameCryptographicDomainParameters AttributeName = "Cryptographic Domain Parameters"
	AttributeNameCertificateType               AttributeName = "Certificate Type"
	AttributeNameDigest                        AttributeName = "Digest"
	AttributeNameCryptographicUsageMask        AttributeName = "Cryptographic Usage Mask"
	AttributeNameState                         AttributeName = "State"
	AttributeNameRevocationReason              AttributeName = "Revocation Reason"
	AttributeNameLink                          AttributeName = "Link"
	// Deprecated: deprecated as of kmip 1.1.
	AttributeNameCertificateIdentifier AttributeName = "Certificate Identifier"
	// Deprecated: deprecated as of kmip 1.1.
	AttributeNameCertificateSubject AttributeName = "Certificate Subject"
	// Deprecated: deprecated as of kmip 1.1.
	AttributeNameCertificateIssuer              AttributeName = "Certificate Issuer"
	AttributeNameUsageLimits                    AttributeName = "Usage Limits"
	AttributeNameApplicationSpecificInformation AttributeName = "Application Specific Information"

	// KMIP 1.1.
	AttributeNameCertificateLength         AttributeName = "Certificate Length"
	AttributeNameFresh                     AttributeName = "Fresh"
	AttributeNameX509CertificateIdentifier AttributeName = "X.509 Certificate Identifier"
	AttributeNameX509CertificateSubject    AttributeName = "X.509 Certificate Subject"
	AttributeNameX509CertificateIssuer     AttributeName = "X.509 Certificate Issuer"
	AttributeNameDigitalSignatureAlgorithm AttributeName = "Digital Signature Algorithm"

	// KMIP 1.2.
	AttributeNameAlternativeName      AttributeName = "Alternative Name"
	AttributeNameKeyValuePresent      AttributeName = "Key Value Present"
	AttributeNameKeyValueLocation     AttributeName = "Key Value Location"
	AttributeNameOriginalCreationDate AttributeName = "Original Creation Date"

	// KMIP 1.3.
	AttributeNameRandomNumberGenerator AttributeName = "Random Number Generator"

	// KMIP 1.4.
	AttributeNamePKCS_12FriendlyName AttributeName = "PKCS#12 Friendly Name"
	AttributeNameDescription         AttributeName = "Description"
	AttributeNameComment             AttributeName = "Comment"
	AttributeNameSensitive           AttributeName = "Sensitive"
	AttributeNameAlwaysSensitive     AttributeName = "Always Sensitive"
	AttributeNameExtractable         AttributeName = "Extractable"
	AttributeNameNeverExtractable    AttributeName = "Never Extractable"
)

var AllAttributeNames = []AttributeName{
	AttributeNameUniqueIdentifier, AttributeNameName, AttributeNameObjectType, AttributeNameOperationPolicyName, AttributeNameObjectGroup,
	AttributeNameContactInformation, AttributeNameInitialDate, AttributeNameActivationDate, AttributeNameProcessStartDate, AttributeNameProtectStopDate,
	AttributeNameDeactivationDate, AttributeNameDestroyDate, AttributeNameCompromiseOccurrenceDate, AttributeNameCompromiseDate, AttributeNameArchiveDate,
	AttributeNameLastChangeDate, AttributeNameCryptographicLength, AttributeNameLeaseTime, AttributeNameCryptographicAlgorithm, AttributeNameCryptographicParameters,
	AttributeNameCryptographicDomainParameters, AttributeNameCertificateType, AttributeNameDigest, AttributeNameCryptographicUsageMask, AttributeNameState, AttributeNameRevocationReason,
	AttributeNameLink, AttributeNameCertificateIdentifier, AttributeNameCertificateSubject, AttributeNameCertificateIssuer, AttributeNameUsageLimits,
	AttributeNameApplicationSpecificInformation, AttributeNameCertificateLength, AttributeNameFresh, AttributeNameX509CertificateIdentifier, AttributeNameX509CertificateSubject,
	AttributeNameX509CertificateIssuer, AttributeNameDigitalSignatureAlgorithm, AttributeNameAlternativeName, AttributeNameKeyValuePresent, AttributeNameKeyValueLocation,
	AttributeNameOriginalCreationDate, AttributeNameRandomNumberGenerator, AttributeNamePKCS_12FriendlyName, AttributeNameDescription, AttributeNameComment, AttributeNameSensitive,
	AttributeNameAlwaysSensitive, AttributeNameExtractable, AttributeNameNeverExtractable,
}

var attrTypes = map[AttributeName]reflect.Type{
	AttributeNameUniqueIdentifier:               reflect.TypeFor[string](),
	AttributeNameName:                           reflect.TypeFor[Name](),
	AttributeNameObjectType:                     reflect.TypeFor[ObjectType](),
	AttributeNameOperationPolicyName:            reflect.TypeFor[string](),
	AttributeNameObjectGroup:                    reflect.TypeFor[string](),
	AttributeNameContactInformation:             reflect.TypeFor[string](),
	AttributeNameInitialDate:                    reflect.TypeFor[time.Time](),
	AttributeNameActivationDate:                 reflect.TypeFor[time.Time](),
	AttributeNameProcessStartDate:               reflect.TypeFor[time.Time](),
	AttributeNameProtectStopDate:                reflect.TypeFor[time.Time](),
	AttributeNameDeactivationDate:               reflect.TypeFor[time.Time](),
	AttributeNameDestroyDate:                    reflect.TypeFor[time.Time](),
	AttributeNameCompromiseOccurrenceDate:       reflect.TypeFor[time.Time](),
	AttributeNameCompromiseDate:                 reflect.TypeFor[time.Time](),
	AttributeNameArchiveDate:                    reflect.TypeFor[time.Time](),
	AttributeNameLastChangeDate:                 reflect.TypeFor[time.Time](),
	AttributeNameCryptographicLength:            reflect.TypeFor[int32](),
	AttributeNameLeaseTime:                      reflect.TypeFor[time.Duration](),
	AttributeNameCryptographicAlgorithm:         reflect.TypeFor[CryptographicAlgorithm](),
	AttributeNameCryptographicParameters:        reflect.TypeFor[CryptographicParameters](),
	AttributeNameCryptographicDomainParameters:  reflect.TypeFor[CryptographicDomainParameters](),
	AttributeNameCertificateType:                reflect.TypeFor[CertificateType](),
	AttributeNameDigest:                         reflect.TypeFor[Digest](),
	AttributeNameCryptographicUsageMask:         reflect.TypeFor[CryptographicUsageMask](),
	AttributeNameState:                          reflect.TypeFor[State](),
	AttributeNameRevocationReason:               reflect.TypeFor[RevocationReason](),
	AttributeNameLink:                           reflect.TypeFor[Link](),
	AttributeNameCertificateIdentifier:          reflect.TypeFor[CertificateIdentifier](),
	AttributeNameCertificateSubject:             reflect.TypeFor[CertificateSubject](),
	AttributeNameCertificateIssuer:              reflect.TypeFor[CertificateIssuer](),
	AttributeNameUsageLimits:                    reflect.TypeFor[UsageLimits](),
	AttributeNameApplicationSpecificInformation: reflect.TypeFor[ApplicationSpecificInformation](),

	AttributeNameCertificateLength:         reflect.TypeFor[int32](),
	AttributeNameFresh:                     reflect.TypeFor[bool](),
	AttributeNameX509CertificateIdentifier: reflect.TypeFor[X_509CertificateIdentifier](),
	AttributeNameX509CertificateSubject:    reflect.TypeFor[X_509CertificateSubject](),
	AttributeNameX509CertificateIssuer:     reflect.TypeFor[X_509CertificateIssuer](),
	AttributeNameDigitalSignatureAlgorithm: reflect.TypeFor[DigitalSignatureAlgorithm](),

	AttributeNameAlternativeName:      reflect.TypeFor[AlternativeName](),
	AttributeNameKeyValuePresent:      reflect.TypeFor[bool](),
	AttributeNameKeyValueLocation:     reflect.TypeFor[KeyValueLocation](),
	AttributeNameOriginalCreationDate: reflect.TypeFor[time.Time](),

	AttributeNameRandomNumberGenerator: reflect.TypeFor[RNGParameters](),

	AttributeNamePKCS_12FriendlyName: reflect.TypeFor[string](),
	AttributeNameDescription:         reflect.TypeFor[string](),
	AttributeNameComment:             reflect.TypeFor[string](),
	AttributeNameSensitive:           reflect.TypeFor[bool](),
	AttributeNameAlwaysSensitive:     reflect.TypeFor[bool](),
	AttributeNameExtractable:         reflect.TypeFor[bool](),
	AttributeNameNeverExtractable:    reflect.TypeFor[bool](),
}

func newAttribute(name AttributeName) reflect.Value {
	if name.IsCustom() {
		return reflect.ValueOf(&ttlv.Value{})
	}
	ty, ok := attrTypes[name]
	if !ok {
		return reflect.ValueOf(&ttlv.Value{})
	}
	return reflect.New(ty)
}

type Name struct {
	NameValue string   `ttlv:",omitempty"`
	NameType  NameType `ttlv:",omitempty"`
}

type TemplateAttribute struct {
	// Deprecated: deprecated as of kmip 1.3
	Name      []Name
	Attribute []Attribute
}

// func (tpl *TemplateAttribute) AppendAttribute(name AttributeName, value any) {
// 	cnt := int32(0)
// 	for _, attr := range tpl.Attribute {
// 		if attr.AttributeName == name {
// 			if idx := attr.AttributeIndex; idx != nil && *idx >= cnt {
// 				cnt = *idx + 1
// 				continue
// 			}
// 			cnt++
// 		}
// 	}
// 	tpl.Attribute = append(tpl.Attribute, Attribute{AttributeName: name, AttributeIndex: &cnt, AttributeValue: value})
// }

type Attribute struct {
	AttributeName  AttributeName
	AttributeIndex *int32
	AttributeValue any
}

func (att *Attribute) TagDecodeTTLV(d *ttlv.Decoder, tag int) error {
	return d.Struct(tag, func(d *ttlv.Decoder) error {
		attName, err := d.TextString(TagAttributeName)
		if err != nil {
			return err
		}
		att.AttributeName = AttributeName(attName)
		if d.Tag() == TagAttributeIndex {
			idx, err := d.Integer(TagAttributeIndex)
			if err != nil {
				return err
			}
			att.AttributeIndex = &idx
		}
		attr := newAttribute(att.AttributeName)
		if attr.IsNil() {
			return d.Next()
		}
		if err := d.TagAny(TagAttributeValue, attr.Interface()); err != nil {
			return err
		}
		att.AttributeValue = attr.Elem().Interface()
		return nil
	})
}
