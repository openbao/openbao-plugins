package kmipclient

import (
	"github.com/ovh/kmip-go"
	"github.com/ovh/kmip-go/payloads"
)

func (c *Client) CreateKeyPair() ExecCreateKeyPair {
	return ExecCreateKeyPair{
		Executor[*payloads.CreateKeyPairRequestPayload, *payloads.CreateKeyPairResponsePayload]{
			client: c,
			req: &payloads.CreateKeyPairRequestPayload{
				CommonTemplateAttribute:     &kmip.TemplateAttribute{},
				PrivateKeyTemplateAttribute: &kmip.TemplateAttribute{},
				PublicKeyTemplateAttribute:  &kmip.TemplateAttribute{},
			},
		},
	}
}

type ExecCreateKeyPair struct {
	Executor[*payloads.CreateKeyPairRequestPayload, *payloads.CreateKeyPairResponsePayload]
}

func (ex ExecCreateKeyPair) RSA(bitlen int, privateUsage, publicUsage kmip.CryptographicUsageMask) ExecCreateKeyPairAttr {
	return ex.Common().
		WithAttribute(kmip.AttributeNameCryptographicAlgorithm, kmip.CryptographicAlgorithmRSA).
		WithAttribute(kmip.AttributeNameCryptographicLength, int32(bitlen)).
		PublicKey().WithAttribute(kmip.AttributeNameCryptographicUsageMask, publicUsage).
		PrivateKey().WithAttribute(kmip.AttributeNameCryptographicUsageMask, privateUsage).
		Common()
}

func (ex ExecCreateKeyPair) ECDSA(curve kmip.RecommendedCurve, privateUsage, publicUsage kmip.CryptographicUsageMask) ExecCreateKeyPairAttr {
	return ex.Common().
		WithAttribute(kmip.AttributeNameCryptographicAlgorithm, kmip.CryptographicAlgorithmECDSA).
		WithAttribute(kmip.AttributeNameCryptographicLength, curve.Bitlen()).
		WithAttribute(kmip.AttributeNameCryptographicDomainParameters, kmip.CryptographicDomainParameters{RecommendedCurve: curve}).
		PublicKey().WithAttribute(kmip.AttributeNameCryptographicUsageMask, publicUsage).
		PrivateKey().WithAttribute(kmip.AttributeNameCryptographicUsageMask, privateUsage).
		Common()
}

func (ex ExecCreateKeyPair) Common() ExecCreateKeyPairAttr {
	return ExecCreateKeyPairAttr{
		AttributeExecutor[*payloads.CreateKeyPairRequestPayload, *payloads.CreateKeyPairResponsePayload, ExecCreateKeyPairAttr]{
			ex.Executor,
			func(ckprp **payloads.CreateKeyPairRequestPayload) *[]kmip.Attribute {
				return &(*ckprp).CommonTemplateAttribute.Attribute
			},
			func(ae AttributeExecutor[*payloads.CreateKeyPairRequestPayload, *payloads.CreateKeyPairResponsePayload, ExecCreateKeyPairAttr]) ExecCreateKeyPairAttr {
				return ExecCreateKeyPairAttr{ae, func(ckprp *payloads.CreateKeyPairRequestPayload) *[]kmip.Name {
					//nolint:staticcheck // for backward compatibility
					return &ckprp.CommonTemplateAttribute.Name
				}}
			},
		},
		func(ckprp *payloads.CreateKeyPairRequestPayload) *[]kmip.Name {
			//nolint:staticcheck // for backward compatibility
			return &ckprp.CommonTemplateAttribute.Name
		},
	}
}

func (ex ExecCreateKeyPair) PrivateKey() ExecCreateKeyPairAttr {
	return ExecCreateKeyPairAttr{
		AttributeExecutor[*payloads.CreateKeyPairRequestPayload, *payloads.CreateKeyPairResponsePayload, ExecCreateKeyPairAttr]{
			ex.Executor,
			func(ckprp **payloads.CreateKeyPairRequestPayload) *[]kmip.Attribute {
				return &(*ckprp).PrivateKeyTemplateAttribute.Attribute
			},
			func(ae AttributeExecutor[*payloads.CreateKeyPairRequestPayload, *payloads.CreateKeyPairResponsePayload, ExecCreateKeyPairAttr]) ExecCreateKeyPairAttr {
				return ExecCreateKeyPairAttr{ae, func(ckprp *payloads.CreateKeyPairRequestPayload) *[]kmip.Name {
					//nolint:staticcheck // for backward compatibility
					return &ckprp.PrivateKeyTemplateAttribute.Name
				}}
			},
		},
		func(ckprp *payloads.CreateKeyPairRequestPayload) *[]kmip.Name {
			//nolint:staticcheck // for backward compatibility
			return &ckprp.PrivateKeyTemplateAttribute.Name
		},
	}
}

func (ex ExecCreateKeyPair) PublicKey() ExecCreateKeyPairAttr {
	return ExecCreateKeyPairAttr{
		AttributeExecutor[*payloads.CreateKeyPairRequestPayload, *payloads.CreateKeyPairResponsePayload, ExecCreateKeyPairAttr]{
			ex.Executor,
			func(ckprp **payloads.CreateKeyPairRequestPayload) *[]kmip.Attribute {
				return &(*ckprp).PublicKeyTemplateAttribute.Attribute
			},
			func(ae AttributeExecutor[*payloads.CreateKeyPairRequestPayload, *payloads.CreateKeyPairResponsePayload, ExecCreateKeyPairAttr]) ExecCreateKeyPairAttr {
				return ExecCreateKeyPairAttr{ae, func(ckprp *payloads.CreateKeyPairRequestPayload) *[]kmip.Name {
					//nolint:staticcheck // for backward compatibility
					return &ckprp.PublicKeyTemplateAttribute.Name
				}}
			},
		},
		func(ckprp *payloads.CreateKeyPairRequestPayload) *[]kmip.Name {
			//nolint:staticcheck // for backward compatibility
			return &ckprp.PublicKeyTemplateAttribute.Name
		},
	}
}

type ExecCreateKeyPairAttr struct {
	AttributeExecutor[*payloads.CreateKeyPairRequestPayload, *payloads.CreateKeyPairResponsePayload, ExecCreateKeyPairAttr]
	tmplFunc func(*payloads.CreateKeyPairRequestPayload) *[]kmip.Name
}

// Deprecated: Templates have been deprecated in KMIP v1.3.
func (ex ExecCreateKeyPairAttr) WithTemplates(names ...kmip.Name) ExecCreateKeyPairAttr {
	tmpl := ex.tmplFunc(ex.req)
	*tmpl = append(*tmpl, names...)
	return ex
}

// Deprecated: Templates have been deprecated in KMIP v1.3.
func (ex ExecCreateKeyPairAttr) WithTemplate(name string, nameType kmip.NameType) ExecCreateKeyPairAttr {
	ex.WithTemplates(kmip.Name{NameValue: name, NameType: nameType})
	return ex
}

func (ex ExecCreateKeyPairAttr) Common() ExecCreateKeyPairAttr {
	return ExecCreateKeyPair{ex.Executor}.Common()
}

func (ex ExecCreateKeyPairAttr) PrivateKey() ExecCreateKeyPairAttr {
	return ExecCreateKeyPair{ex.Executor}.PrivateKey()
}

func (ex ExecCreateKeyPairAttr) PublicKey() ExecCreateKeyPairAttr {
	return ExecCreateKeyPair{ex.Executor}.PublicKey()
}
