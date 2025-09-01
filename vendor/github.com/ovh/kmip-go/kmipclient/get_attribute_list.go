package kmipclient

import "github.com/ovh/kmip-go/payloads"

func (c *Client) GetAttributeList(id string) ExecGetAttributeList {
	return ExecGetAttributeList{
		client: c,
		req: &payloads.GetAttributeListRequestPayload{
			UniqueIdentifier: id,
		},
	}
}

type ExecGetAttributeList = Executor[*payloads.GetAttributeListRequestPayload, *payloads.GetAttributeListResponsePayload]
