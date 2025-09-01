package kmipclient

import (
	"time"

	"github.com/ovh/kmip-go"
	"github.com/ovh/kmip-go/payloads"
)

func (c *Client) Rekey(id string) ExecRekey {
	return ExecRekey{
		AttributeExecutor[*payloads.RekeyRequestPayload, *payloads.RekeyResponsePayload, ExecRekey]{
			Executor[*payloads.RekeyRequestPayload, *payloads.RekeyResponsePayload]{
				client: c,
				req: &payloads.RekeyRequestPayload{
					UniqueIdentifier:  id,
					TemplateAttribute: &kmip.TemplateAttribute{},
				},
			},
			func(lrp **payloads.RekeyRequestPayload) *[]kmip.Attribute {
				return &(*lrp).TemplateAttribute.Attribute
			},
			func(ae AttributeExecutor[*payloads.RekeyRequestPayload, *payloads.RekeyResponsePayload, ExecRekey]) ExecRekey {
				return ExecRekey{ae}
			},
		},
	}
}

type ExecRekey struct {
	AttributeExecutor[*payloads.RekeyRequestPayload, *payloads.RekeyResponsePayload, ExecRekey]
}

func (ex ExecRekey) WithOffset(offset time.Duration) ExecRekey {
	ex.req.Offset = &offset
	return ex
}

// Deprecated: Templates have been deprecated in KMIP v1.3.
func (ex ExecRekey) WithTemplates(names ...kmip.Name) ExecRekey {
	//nolint:staticcheck // for backward compatibility
	ex.req.TemplateAttribute.Name = append(ex.req.TemplateAttribute.Name, names...)
	return ex
}

// Deprecated: Templates have been deprecated in KMIP v1.3.
func (ex ExecRekey) WithTemplate(name string, nameType kmip.NameType) ExecRekey {
	//nolint:staticcheck // for backward compatibility
	ex.req.TemplateAttribute.Name = append(ex.req.TemplateAttribute.Name, kmip.Name{NameValue: name, NameType: nameType})
	return ex
}
