package payloads

import (
	"time"

	"github.com/ovh/kmip-go"
)

func init() {
	kmip.RegisterOperationPayload[RevokeRequestPayload, RevokeResponsePayload](kmip.OperationRevoke)
}

var _ kmip.OperationPayload = (*RevokeRequestPayload)(nil)

// This operation requests the server to revoke a Managed Cryptographic Object or an Opaque Object. The request SHALL NOT specify a Template object.
// The request contains a reason for the revocation (e.g., “key compromise”, “cessation of operation”, etc.).
// Special authentication and authorization SHOULD be enforced to perform this request. Only the object owner or an authorized security officer
// SHOULD be allowed to issue this request. The operation has one of two effects. If the revocation reason is “key compromise” or “CA compromise”,
// then the object is placed into the “compromised” state; the Date is set to the current date and time; and the Compromise Occurrence Date is
// set to the value (if provided) in the Revoke request and if a value is not provided in the Revoke request then Compromise Occurrence Date SHOULD
// be set to the Initial Date for the object. If the revocation reason is neither “key compromise” nor “CA compromise”,
// the object is placed into the “deactivated” state, and the Deactivation Date is set to the current date and time.
type RevokeRequestPayload struct {
	// Determines the object being revoked. If omitted, then the ID Placeholder value is used by the server as the Unique Identifier.
	UniqueIdentifier string `ttlv:",omitempty"`
	// Specifies the reason for revocation.
	RevocationReason kmip.RevocationReason
	// SHOULD be specified if the Revocation Reason is 'key compromise' or ‘CA compromise’ and SHALL NOT be specified for other Revocation Reason enumerations.
	CompromiseOccurrenceDate *time.Time
}

// Operation implements kmip.OperationPayload.
func (a *RevokeRequestPayload) Operation() kmip.Operation {
	return kmip.OperationRevoke
}

// Response for the Revoke operation.
type RevokeResponsePayload struct {
	// The Unique Identifier of the object.
	UniqueIdentifier string
}

var _ kmip.OperationPayload = (*RevokeResponsePayload)(nil)

// Operation implements kmip.OperationPayload.
func (a *RevokeResponsePayload) Operation() kmip.Operation {
	return kmip.OperationRevoke
}
