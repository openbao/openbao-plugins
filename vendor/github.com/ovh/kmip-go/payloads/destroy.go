package payloads

import (
	"github.com/ovh/kmip-go"
)

func init() {
	kmip.RegisterOperationPayload[DestroyRequestPayload, DestroyResponsePayload](kmip.OperationDestroy)
}

var _ kmip.OperationPayload = (*DestroyRequestPayload)(nil)

// This operation is used to indicate to the server that the key material for the specified Managed Object SHALL be destroyed.
// The meta-data for the key material MAY be retained by the server (e.g., used to ensure that an expired or revoked private signing key is no longer available).
// Special authentication and authorization SHOULD be enforced to perform this request.
// Only the object owner or an authorized security officer SHOULD be allowed to issue this request. If the Unique Identifier specifies a Template object,
// then the object itself, including all meta-data, SHALL be destroyed. Cryptographic Objects MAY only be destroyed if they are in either Pre-Active or Deactivated state.
type DestroyRequestPayload struct {
	// Determines the object being destroyed. If omitted, then the ID Placeholder value is used by the server as the Unique Identifier.
	UniqueIdentifier string `ttlv:",omitempty"`
}

// Operation implements kmip.OperationPayload.
func (a *DestroyRequestPayload) Operation() kmip.Operation {
	return kmip.OperationDestroy
}

// Response for the destroy operation.
type DestroyResponsePayload struct {
	// The Unique Identifier of the object.
	UniqueIdentifier string
}

var _ kmip.OperationPayload = (*DestroyResponsePayload)(nil)

// Operation implements kmip.OperationPayload.
func (a *DestroyResponsePayload) Operation() kmip.Operation {
	return kmip.OperationDestroy
}
