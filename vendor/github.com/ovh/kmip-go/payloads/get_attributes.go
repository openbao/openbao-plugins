package payloads

import (
	"github.com/ovh/kmip-go"
)

func init() {
	kmip.RegisterOperationPayload[GetAttributesRequestPayload, GetAttributesResponsePayload](kmip.OperationGetAttributes)
}

var _ kmip.OperationPayload = (*GetAttributesRequestPayload)(nil)

// This operation requests one or more attributes associated with a Managed Object.
// The object is specified by its Unique Identifier, and the attributes are specified by their name in the request.
// If a specified attribute has multiple instances, then all instances are returned. If a specified attribute does not exist (i.e., has no value),
// then it SHALL NOT be present in the returned response. If no requested attributes exist, then the response SHALL consist only of the Unique Identifier.
// If no attribute name is specified in the request, all attributes SHALL be deemed to match the Get Attributes request.
// The same attribute name SHALL NOT be present more than once in a request.
type GetAttributesRequestPayload struct {
	// Determines the object whose attributes are being requested. If omitted, then the ID Placeholder value is used by the server as the Unique Identifier.
	UniqueIdentifier string `ttlv:",omitempty"`
	// Specifies the name of an attribute associated with the object.
	AttributeName []kmip.AttributeName
}

// Operation implements kmip.OperationPayload.
func (a *GetAttributesRequestPayload) Operation() kmip.Operation {
	return kmip.OperationGetAttributes
}

type GetAttributesResponsePayload struct {
	// The Unique Identifier of the object.
	UniqueIdentifier string
	// The requested attribute associated with the object.
	Attribute []kmip.Attribute
}

var _ kmip.OperationPayload = (*GetAttributesResponsePayload)(nil)

// Operation implements kmip.OperationPayload.
func (a *GetAttributesResponsePayload) Operation() kmip.Operation {
	return kmip.OperationGetAttributes
}
