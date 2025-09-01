package kmipclient

import (
	"github.com/ovh/kmip-go/payloads"
)

func (c *Client) Archive(id string) ExecArchive {
	return ExecArchive{
		client: c,
		req: &payloads.ArchiveRequestPayload{
			UniqueIdentifier: id,
		},
	}
}

type ExecArchive = Executor[*payloads.ArchiveRequestPayload, *payloads.ArchiveResponsePayload]

func (c *Client) Recover(id string) ExecRecover {
	return ExecRecover{
		client: c,
		req: &payloads.RecoverRequestPayload{
			UniqueIdentifier: id,
		},
	}
}

type ExecRecover = Executor[*payloads.RecoverRequestPayload, *payloads.RecoverResponsePayload]
