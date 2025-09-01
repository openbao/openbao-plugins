package kmipclient

import "github.com/ovh/kmip-go/payloads"

func (c *Client) ObtainLease(id string) ExecObtainLease {
	return ExecObtainLease{
		client: c,
		req: &payloads.ObtainLeaseRequestPayload{
			UniqueIdentifier: id,
		},
	}
}

type ExecObtainLease = Executor[*payloads.ObtainLeaseRequestPayload, *payloads.ObtainLeaseResponsePayload]
