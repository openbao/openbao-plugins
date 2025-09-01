package kmipclient

import (
	"github.com/ovh/kmip-go"
	"github.com/ovh/kmip-go/payloads"
)

type ExecEncrypt struct {
	Executor[*payloads.EncryptRequestPayload, *payloads.EncryptResponsePayload]
}

type ExecDecrypt struct {
	Executor[*payloads.DecryptRequestPayload, *payloads.DecryptResponsePayload]
}

type ExecEncryptWantsData struct {
	req    *payloads.EncryptRequestPayload
	client *Client
}

type ExecDecryptWantsData struct {
	req    *payloads.DecryptRequestPayload
	client *Client
}

func (c *Client) Encrypt(id string) ExecEncryptWantsData {
	return ExecEncryptWantsData{
		client: c,
		req: &payloads.EncryptRequestPayload{
			UniqueIdentifier: id,
		},
	}
}

func (ex ExecEncryptWantsData) WithIvCounterNonce(iv []byte) ExecEncryptWantsData {
	ex.req.IVCounterNonce = iv
	return ex
}

func (ex ExecEncryptWantsData) WithAAD(aad []byte) ExecEncryptWantsData {
	ex.req.AuthenticatedEncryptionAdditionalData = aad
	return ex
}

func (ex ExecEncryptWantsData) WithCryptographicParameters(params kmip.CryptographicParameters) ExecEncryptWantsData {
	ex.req.CryptographicParameters = &params
	return ex
}

func (ex ExecEncryptWantsData) Data(data []byte) ExecEncrypt {
	ex.req.Data = data
	return ExecEncrypt{
		Executor[*payloads.EncryptRequestPayload, *payloads.EncryptResponsePayload]{
			client: ex.client,
			req:    ex.req,
		},
	}
}

func (c *Client) Decrypt(id string) ExecDecryptWantsData {
	return ExecDecryptWantsData{
		client: c,
		req: &payloads.DecryptRequestPayload{
			UniqueIdentifier: id,
		},
	}
}

func (ex ExecDecryptWantsData) WithIvCounterNonce(iv []byte) ExecDecryptWantsData {
	ex.req.IVCounterNonce = iv
	return ex
}

func (ex ExecDecryptWantsData) WithAAD(aad []byte) ExecDecryptWantsData {
	ex.req.AuthenticatedEncryptionAdditionalData = aad
	return ex
}

func (ex ExecDecryptWantsData) WithCryptographicParameters(params kmip.CryptographicParameters) ExecDecryptWantsData {
	ex.req.CryptographicParameters = &params
	return ex
}

func (ex ExecDecryptWantsData) WithAuthTag(tag []byte) ExecDecryptWantsData {
	ex.req.AuthenticatedEncryptionTag = tag
	return ex
}

func (ex ExecDecryptWantsData) Data(data []byte) ExecDecrypt {
	ex.req.Data = data
	return ExecDecrypt{
		Executor[*payloads.DecryptRequestPayload, *payloads.DecryptResponsePayload]{
			client: ex.client,
			req:    ex.req,
		},
	}
}
