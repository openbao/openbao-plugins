package kmipclient

import (
	"github.com/ovh/kmip-go"
	"github.com/ovh/kmip-go/payloads"
)

func (c *Client) AddAttribute(id string, name kmip.AttributeName, value any) ExecAddAttribute {
	return ExecAddAttribute{
		Executor[*payloads.AddAttributeRequestPayload, *payloads.AddAttributeResponsePayload]{
			client: c,
			req: &payloads.AddAttributeRequestPayload{
				UniqueIdentifier: id,
				Attribute:        kmip.Attribute{AttributeName: name, AttributeValue: value},
			},
		},
	}
}

type ExecAddAttribute struct {
	Executor[*payloads.AddAttributeRequestPayload, *payloads.AddAttributeResponsePayload]
}

func (ex ExecAddAttribute) WithIndex(index int32) ExecAddAttribute {
	ex.req.Attribute.AttributeIndex = &index
	return ex
}
