package kmipclient

import (
	"github.com/ovh/kmip-go"
	"github.com/ovh/kmip-go/payloads"
)

func (c *Client) Get(id string) ExecGet {
	return ExecGet{
		Executor[*payloads.GetRequestPayload, *payloads.GetResponsePayload]{
			client: c,
			req: &payloads.GetRequestPayload{
				UniqueIdentifier: id,
			},
		},
	}
}

type ExecGet struct {
	Executor[*payloads.GetRequestPayload, *payloads.GetResponsePayload]
}

func (ex ExecGet) WithKeyFormat(format kmip.KeyFormatType) ExecGet {
	ex.req.KeyFormatType = format
	return ex
}

func (ex ExecGet) WithKeyWrapType(format kmip.KeyFormatType) ExecGet {
	ex.req.KeyWrapType = format
	return ex
}

func (ex ExecGet) WithKeyCompression(compression kmip.KeyCompressionType) ExecGet {
	ex.req.KeyCompressionType = compression
	return ex
}

func (ex ExecGet) WithKeyWrapping(spec kmip.KeyWrappingSpecification) ExecGet {
	ex.req.KeyWrappingSpecification = &spec
	return ex
}
