package kmipclient

import (
	"github.com/ovh/kmip-go"
	"github.com/ovh/kmip-go/payloads"
)

func (c *Client) Locate() ExecLocate {
	return ExecLocate{
		AttributeExecutor[*payloads.LocateRequestPayload, *payloads.LocateResponsePayload, ExecLocate]{
			Executor[*payloads.LocateRequestPayload, *payloads.LocateResponsePayload]{
				client: c,
				req:    &payloads.LocateRequestPayload{},
			},
			func(lrp **payloads.LocateRequestPayload) *[]kmip.Attribute {
				return &(*lrp).Attribute
			},
			func(ae AttributeExecutor[*payloads.LocateRequestPayload, *payloads.LocateResponsePayload, ExecLocate]) ExecLocate {
				return ExecLocate{ae}
			},
		},
	}
}

type ExecLocate struct {
	AttributeExecutor[*payloads.LocateRequestPayload, *payloads.LocateResponsePayload, ExecLocate]
}

func (ex ExecLocate) WithStorageStatusMask(mask kmip.StorageStatusMask) ExecLocate {
	ex.req.StorageStatusMask = mask
	return ex
}

func (ex ExecLocate) WithMaxItems(maximum int32) ExecLocate {
	ex.req.MaximumItems = maximum
	return ex
}

func (ex ExecLocate) WithOffset(offset int32) ExecLocate {
	ex.req.OffsetItems = offset
	return ex
}

func (ex ExecLocate) WithObjectGroupMember(groupMember kmip.ObjectGroupMember) ExecLocate {
	ex.req.ObjectGroupMember = groupMember
	return ex
}
