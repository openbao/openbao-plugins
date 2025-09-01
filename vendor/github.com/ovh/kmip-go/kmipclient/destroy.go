package kmipclient

import (
	"github.com/ovh/kmip-go/payloads"
)

func (c *Client) Destroy(id string) ExecDestroy {
	return ExecDestroy{
		client: c,
		req: &payloads.DestroyRequestPayload{
			UniqueIdentifier: id,
		},
	}
}

type ExecDestroy = Executor[*payloads.DestroyRequestPayload, *payloads.DestroyResponsePayload]
