package kmipclient

import (
	"time"

	"github.com/ovh/kmip-go"
	"github.com/ovh/kmip-go/payloads"
)

func (c *Client) Revoke(id string) ExecRevoke {
	return ExecRevoke{
		Executor[*payloads.RevokeRequestPayload, *payloads.RevokeResponsePayload]{
			client: c,
			req: &payloads.RevokeRequestPayload{
				UniqueIdentifier: id,
				RevocationReason: kmip.RevocationReason{
					RevocationReasonCode: kmip.RevocationReasonCodeUnspecified,
				},
			},
		},
	}
}

type ExecRevoke struct {
	Executor[*payloads.RevokeRequestPayload, *payloads.RevokeResponsePayload]
}

func (ex ExecRevoke) WithRevocationReasonCode(code kmip.RevocationReasonCode) ExecRevoke {
	ex.req.RevocationReason.RevocationReasonCode = code
	return ex
}

func (ex ExecRevoke) WithRevocationMessage(msg string) ExecRevoke {
	if msg != "" {
		ex.req.RevocationReason.RevocationMessage = msg
	}
	return ex
}

func (ex ExecRevoke) WithCompromiseOccurrenceDate(dt time.Time) ExecRevoke {
	ex.req.CompromiseOccurrenceDate = &dt
	return ex
}
