package payloads

import "github.com/ovh/kmip-go"

func init() {
	kmip.RegisterOperationPayload[GetUsageAllocationRequestPayload, GetUsageAllocationResponsePayload](kmip.OperationGetUsageAllocation)
}

// This operation requests the server to obtain an allocation from the current Usage Limits value to allow the client to use the Managed Cryptographic Object
// for applying cryptographic protection. The allocation only applies to Managed Cryptographic Objects that are able to be used for applying protection
// (e.g., symmetric keys for encryption, private keys for signing, etc.) and is only valid if the Managed Cryptographic Object has a Usage Limits attribute.
// Usage for processing cryptographically protected information (e.g., decryption, verification, etc.) is not limited and is not able to be allocated.
// A Managed Cryptographic Object that has a Usage Limits attribute SHALL NOT be used by a client for applying cryptographic protection unless an allocation
// has been obtained using this operation. The operation SHALL only be requested during the time that protection is enabled for these objects
// (i.e., after the Activation Date and before the Protect Stop Date). If the operation is requested for an object that has no Usage Limits attribute,
// or is not an object that MAY be used for applying cryptographic protection, then the server SHALL return an error.
//
// The field in the request specifies the number of units that the client needs to protect.
// If the requested amount is not available or if the Managed Object is not able to be used for applying cryptographic protection at this time,
// then the server SHALL return an error. The server SHALL assume that the entire allocated amount is going to be consumed.
// Once the entire allocated amount has been consumed, the client SHALL NOT continue to use the Managed Cryptographic Object for applying cryptographic protection
// until a new allocation is obtained.
type GetUsageAllocationRequestPayload struct {
	// Determines the object whose usage allocation is being requested. If omitted, then the ID Placeholder is substituted by the server.
	UniqueIdentifier string `ttlv:",omitempty"`
	// The number of Usage Limits Units to be protected.
	UsageLimitsCount int64
}

func (pl *GetUsageAllocationRequestPayload) Operation() kmip.Operation {
	return kmip.OperationGetUsageAllocation
}

// Response for the get-usage-allocation operation.
type GetUsageAllocationResponsePayload struct {
	// The Unique Identifier of the object.
	UniqueIdentifier string
}

func (pl *GetUsageAllocationResponsePayload) Operation() kmip.Operation {
	return kmip.OperationGetUsageAllocation
}
