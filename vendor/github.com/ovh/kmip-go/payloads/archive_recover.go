package payloads

import "github.com/ovh/kmip-go"

func init() {
	kmip.RegisterOperationPayload[ArchiveRequestPayload, ArchiveResponsePayload](kmip.OperationArchive)
	kmip.RegisterOperationPayload[RecoverRequestPayload, RecoverResponsePayload](kmip.OperationRecover)
}

// This operation is used to specify that a Managed Object MAY be archived. The actual time when the object is archived,
// the location of the archive, or level of archive hierarchy is determined by the policies within the key management system
// and is not specified by the client. The request contains the Unique Identifier of the Managed Object.
// Special authentication and authorization SHOULD be enforced to perform this request.
// Only the object owner or an authorized security officer SHOULD be allowed to issue this request. This request is only an indication from a client that,
// from its point of view, the key management system MAY archive the object.
type ArchiveRequestPayload struct {
	// Determines the object being archived.
	// If omitted, then the ID Placeholder value is used by the server as the Unique Identifier.
	UniqueIdentifier string `ttlv:",omitempty"`
}

func (pl *ArchiveRequestPayload) Operation() kmip.Operation {
	return kmip.OperationArchive
}

// Respsonse for the archive operation.
type ArchiveResponsePayload struct {
	// The Unique Identifier of the object.
	UniqueIdentifier string
}

func (pl *ArchiveResponsePayload) Operation() kmip.Operation {
	return kmip.OperationArchive
}

// This operation is used to obtain access to a Managed Object that has been archived.
// This request MAY need asynchronous polling to obtain the response due to delays caused by retrieving the object from the archive.
// Once the response is received, the object is now on-line, and MAY be obtained (e.g., via a Get operation).
// Special authentication and authorization SHOULD be enforced to perform this request.
type RecoverRequestPayload struct {
	UniqueIdentifier string `ttlv:",omitempty"`
}

func (pl *RecoverRequestPayload) Operation() kmip.Operation {
	return kmip.OperationRecover
}

// Response for the recover operation.
type RecoverResponsePayload struct {
	// The Unique Identifier of the object.
	UniqueIdentifier string
}

func (pl *RecoverResponsePayload) Operation() kmip.Operation {
	return kmip.OperationRecover
}
