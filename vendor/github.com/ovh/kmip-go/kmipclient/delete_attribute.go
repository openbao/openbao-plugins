package kmipclient

import (
	"github.com/ovh/kmip-go"
	"github.com/ovh/kmip-go/payloads"
)

func (c *Client) DeleteAttribute(id string, name kmip.AttributeName) ExecDeleteAttribute {
	return ExecDeleteAttribute{
		Executor[*payloads.DeleteAttributeRequestPayload, *payloads.DeleteAttributeResponsePayload]{
			client: c,
			req: &payloads.DeleteAttributeRequestPayload{
				UniqueIdentifier: id,
				AttributeName:    name,
			},
		},
	}
}

type ExecDeleteAttribute struct {
	Executor[*payloads.DeleteAttributeRequestPayload, *payloads.DeleteAttributeResponsePayload]
}

func (ex ExecDeleteAttribute) WithIndex(index int32) ExecDeleteAttribute {
	ex.req.AttributeIndex = &index
	return ex
}
