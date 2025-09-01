package kmipclient

import (
	"github.com/ovh/kmip-go/payloads"
)

func (c *Client) Activate(id string) ExecActivate {
	return ExecActivate{
		client: c,
		req: &payloads.ActivateRequestPayload{
			UniqueIdentifier: id,
		},
	}
}

type ExecActivate = Executor[*payloads.ActivateRequestPayload, *payloads.ActivateResponsePayload]
