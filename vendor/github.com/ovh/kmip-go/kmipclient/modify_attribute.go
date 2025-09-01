package kmipclient

import (
	"github.com/ovh/kmip-go"
	"github.com/ovh/kmip-go/payloads"
)

func (c *Client) ModifyAttribute(id string, name kmip.AttributeName, value any) ExecModifyAttribute {
	return ExecModifyAttribute{
		Executor[*payloads.ModifyAttributeRequestPayload, *payloads.ModifyAttributeResponsePayload]{
			client: c,
			req: &payloads.ModifyAttributeRequestPayload{
				UniqueIdentifier: id,
				Attribute:        kmip.Attribute{AttributeName: name, AttributeValue: value},
			},
		},
	}
}

type ExecModifyAttribute struct {
	Executor[*payloads.ModifyAttributeRequestPayload, *payloads.ModifyAttributeResponsePayload]
}

func (ex ExecModifyAttribute) WithIndex(index int32) ExecModifyAttribute {
	ex.req.Attribute.AttributeIndex = &index
	return ex
}
