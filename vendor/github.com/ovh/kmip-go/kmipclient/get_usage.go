package kmipclient

import "github.com/ovh/kmip-go/payloads"

func (c *Client) GetUsageAllocation(id string, limitCount int64) ExecGetUsageAllocation {
	return ExecGetUsageAllocation{
		client: c,
		req: &payloads.GetUsageAllocationRequestPayload{
			UniqueIdentifier: id,
			UsageLimitsCount: limitCount,
		},
	}
}

type ExecGetUsageAllocation = Executor[*payloads.GetUsageAllocationRequestPayload, *payloads.GetUsageAllocationResponsePayload]
