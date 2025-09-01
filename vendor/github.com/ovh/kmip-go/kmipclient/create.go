package kmipclient

import (
	"github.com/ovh/kmip-go"
	"github.com/ovh/kmip-go/payloads"
)

func (c *Client) Create() ExecCreateWantType {
	return ExecCreateWantType{
		client: c,
	}
}

type ExecCreateWantType struct {
	client *Client
}

func (ex ExecCreateWantType) Object(objectType kmip.ObjectType, attrs ...kmip.Attribute) ExecCreate {
	return ExecCreate{
		AttributeExecutor[*payloads.CreateRequestPayload, *payloads.CreateResponsePayload, ExecCreate]{
			Executor[*payloads.CreateRequestPayload, *payloads.CreateResponsePayload]{
				client: ex.client,
				req: &payloads.CreateRequestPayload{
					ObjectType:        objectType,
					TemplateAttribute: kmip.TemplateAttribute{Attribute: attrs},
				},
			},
			func(crp **payloads.CreateRequestPayload) *[]kmip.Attribute {
				return &(*crp).TemplateAttribute.Attribute
			},
			func(ae AttributeExecutor[*payloads.CreateRequestPayload, *payloads.CreateResponsePayload, ExecCreate]) ExecCreate {
				return ExecCreate{ae}
			},
		},
	}
}

func (ex ExecCreateWantType) SymmetricKey(alg kmip.CryptographicAlgorithm, length int, usage kmip.CryptographicUsageMask) ExecCreate {
	return ex.Object(kmip.ObjectTypeSymmetricKey).
		WithAttribute(kmip.AttributeNameCryptographicAlgorithm, alg).
		WithAttribute(kmip.AttributeNameCryptographicLength, int32(length)).
		WithAttribute(kmip.AttributeNameCryptographicUsageMask, usage)
}

func (ex ExecCreateWantType) AES(length int, usage kmip.CryptographicUsageMask) ExecCreate {
	return ex.SymmetricKey(kmip.CryptographicAlgorithmAES, length, usage)
}

func (ex ExecCreateWantType) TDES(length int, usage kmip.CryptographicUsageMask) ExecCreate {
	return ex.SymmetricKey(kmip.CryptographicAlgorithmTDES, length, usage)
}

func (ex ExecCreateWantType) Skipjack(usage kmip.CryptographicUsageMask) ExecCreate {
	return ex.SymmetricKey(kmip.CryptographicAlgorithmSKIPJACK, 80, usage)
}

type ExecCreate struct {
	AttributeExecutor[*payloads.CreateRequestPayload, *payloads.CreateResponsePayload, ExecCreate]
}

// Deprecated: Templates have been deprecated in KMIP v1.3.
func (ex ExecCreate) WithTemplates(names ...kmip.Name) ExecCreate {
	//nolint:staticcheck // for backward compatibility
	ex.req.TemplateAttribute.Name = append(ex.req.TemplateAttribute.Name, names...)
	return ex
}

// Deprecated: Templates have been deprecated in KMIP v1.3.
func (ex ExecCreate) WithTemplate(name string, nameType kmip.NameType) ExecCreate {
	//nolint:staticcheck // for backward compatibility
	ex.req.TemplateAttribute.Name = append(ex.req.TemplateAttribute.Name, kmip.Name{NameValue: name, NameType: nameType})
	return ex
}
