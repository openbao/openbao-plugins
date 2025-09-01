package payloads

import (
	"time"

	"github.com/ovh/kmip-go"
)

func init() {
	kmip.RegisterOperationPayload[ObtainLeaseRequestPayload, ObtainLeaseResponsePayload](kmip.OperationObtainLease)
}

// This operation requests the server to obtain a new Lease Time for a specified Managed Object.
// The Lease Time is an interval value that determines when the client's internal cache of information about the object expires and needs to be renewed.
// If the returned value of the lease time is zero, then the server is indicating that no lease interval is effective, and the client MAY use the object without any lease time limit.
// If a client's lease expires, then the client SHALL NOT use the associated cryptographic object until a new lease is obtained.
// If the server determines that a new lease SHALL NOT be issued for the specified cryptographic object, then the server SHALL respond to the Obtain Lease request with an error.
//
// The response payload for the operation contains the current value of the Last Change Date attribute for the object.
// This MAY be used by the client to determine if any of the attributes cached by the client need to be refreshed,
// by comparing this time to the time when the attributes were previously obtained.
type ObtainLeaseRequestPayload struct {
	// Determines the object for which the lease is being obtained. If omitted, then the ID Placeholder value is used by the server as the Unique Identifier.
	UniqueIdentifier string `ttlv:",omitempty"`
}

func (pl *ObtainLeaseRequestPayload) Operation() kmip.Operation {
	return kmip.OperationObtainLease
}

// Response for the ObtainLease operation.
type ObtainLeaseResponsePayload struct {
	// The Unique Identifier of the object.
	UniqueIdentifier string
	// An interval (in seconds) that specifies the amount of time that the object MAY be used until a new lease needs to be obtained.
	LeaseTime time.Duration
	// The date and time indicating when the latest change was made to the contents or any attribute of the specified object.
	LastChangeDate time.Time
}

func (pl *ObtainLeaseResponsePayload) Operation() kmip.Operation {
	return kmip.OperationObtainLease
}
