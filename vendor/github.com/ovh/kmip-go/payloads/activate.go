package payloads

import (
	"github.com/ovh/kmip-go"
)

func init() {
	kmip.RegisterOperationPayload[ActivateRequestPayload, ActivateResponsePayload](kmip.OperationActivate)
}

var _ kmip.OperationPayload = (*ActivateRequestPayload)(nil)

// This operation requests the server to activate a Managed Cryptographic Object. The request SHALL NOT specify a Template object.
// The operation SHALL only be performed on an object in the Pre-Active state and has the effect of changing its state to Active,
// and setting its Activation Date to the current date and time.
type ActivateRequestPayload struct {
	// Determines the object being activated.
	// If omitted, then the ID Placeholder value is used by the server as the Unique Identifier.
	UniqueIdentifier string `ttlv:",omitempty"`
}

// Operation implements kmip.OperationPayload.
func (a *ActivateRequestPayload) Operation() kmip.Operation {
	return kmip.OperationActivate
}

// Response for the activate operation.
type ActivateResponsePayload struct {
	// The Unique Identifier of the object.
	UniqueIdentifier string
}

var _ kmip.OperationPayload = (*ActivateResponsePayload)(nil)

// Operation implements kmip.OperationPayload.
func (a *ActivateResponsePayload) Operation() kmip.Operation {
	return kmip.OperationActivate
}
