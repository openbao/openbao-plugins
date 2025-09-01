package kmipclient

import (
	"github.com/ovh/kmip-go"
	"github.com/ovh/kmip-go/payloads"
)

func (c *Client) GetAttributes(id string, attributes ...kmip.AttributeName) ExecGetAttributes {
	return ExecGetAttributes{
		Executor[*payloads.GetAttributesRequestPayload, *payloads.GetAttributesResponsePayload]{
			client: c,
			req: &payloads.GetAttributesRequestPayload{
				UniqueIdentifier: id,
			},
		},
	}.WithAttributes(attributes...)
}

type ExecGetAttributes struct {
	Executor[*payloads.GetAttributesRequestPayload, *payloads.GetAttributesResponsePayload]
}

func (ex ExecGetAttributes) WithAttributes(names ...kmip.AttributeName) ExecGetAttributes {
	ex.req.AttributeName = append(ex.req.AttributeName, names...)
	return ex
}
