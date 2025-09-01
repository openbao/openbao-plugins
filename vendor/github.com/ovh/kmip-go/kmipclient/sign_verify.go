package kmipclient

import (
	"github.com/ovh/kmip-go"
	"github.com/ovh/kmip-go/payloads"
)

type ExecSign struct {
	Executor[*payloads.SignRequestPayload, *payloads.SignResponsePayload]
}

type ExecSignatureVerify struct {
	Executor[*payloads.SignatureVerifyRequestPayload, *payloads.SignatureVerifyResponsePayload]
}

type ExecSignWantsData struct {
	req    *payloads.SignRequestPayload
	client *Client
}

type ExecSignatureVerifyWantsData struct {
	req    *payloads.SignatureVerifyRequestPayload
	client *Client
}

type ExecSignatureVerifyWantsSignature struct {
	req    *payloads.SignatureVerifyRequestPayload
	client *Client
}

func (c *Client) Sign(id string) ExecSignWantsData {
	return ExecSignWantsData{
		client: c,
		req: &payloads.SignRequestPayload{
			UniqueIdentifier: id,
		},
	}
}

func (ex ExecSignWantsData) WithCryptographicParameters(params kmip.CryptographicParameters) ExecSignWantsData {
	ex.req.CryptographicParameters = &params
	return ex
}

func (ex ExecSignWantsData) Data(data []byte) ExecSign {
	ex.req.Data = data
	return ExecSign{
		Executor[*payloads.SignRequestPayload, *payloads.SignResponsePayload]{
			client: ex.client,
			req:    ex.req,
		},
	}
}

func (ex ExecSignWantsData) DigestedData(data []byte) ExecSign {
	ex.req.DigestedData = data
	return ExecSign{
		Executor[*payloads.SignRequestPayload, *payloads.SignResponsePayload]{
			client: ex.client,
			req:    ex.req,
		},
	}
}

func (c *Client) SignatureVerify(id string) ExecSignatureVerifyWantsData {
	return ExecSignatureVerifyWantsData{
		client: c,
		req: &payloads.SignatureVerifyRequestPayload{
			UniqueIdentifier: id,
		},
	}
}

func (ex ExecSignatureVerifyWantsData) WithCryptographicParameters(params kmip.CryptographicParameters) ExecSignatureVerifyWantsData {
	ex.req.CryptographicParameters = &params
	return ex
}

func (ex ExecSignatureVerifyWantsData) Data(data []byte) ExecSignatureVerifyWantsSignature {
	ex.req.Data = data
	return ExecSignatureVerifyWantsSignature(ex)
}

func (ex ExecSignatureVerifyWantsData) DigestedData(data []byte) ExecSignatureVerifyWantsSignature {
	ex.req.DigestedData = data
	return ExecSignatureVerifyWantsSignature(ex)
}

func (ex ExecSignatureVerifyWantsData) Signature(sig []byte) ExecSignatureVerify {
	ex.req.SignatureData = sig
	return ExecSignatureVerify{
		Executor[*payloads.SignatureVerifyRequestPayload, *payloads.SignatureVerifyResponsePayload]{
			client: ex.client,
			req:    ex.req,
		},
	}
}

func (ex ExecSignatureVerifyWantsSignature) Signature(sig []byte) ExecSignatureVerify {
	ex.req.SignatureData = sig
	return ExecSignatureVerify{
		Executor[*payloads.SignatureVerifyRequestPayload, *payloads.SignatureVerifyResponsePayload]{
			client: ex.client,
			req:    ex.req,
		},
	}
}
