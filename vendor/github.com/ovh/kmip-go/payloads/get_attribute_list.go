package payloads

import "github.com/ovh/kmip-go"

func init() {
	kmip.RegisterOperationPayload[GetAttributeListRequestPayload, GetAttributeListResponsePayload](kmip.OperationGetAttributeList)
}

// This operation requests a list of the attribute names associated with a Managed Object. The object is specified by its Unique Identifier.
type GetAttributeListRequestPayload struct {
	// Determines the object whose attribute names are being requested. If omitted, then the ID Placeholder value is used by the server as the Unique Identifier.
	UniqueIdentifier string `ttlv:",omitempty"`
}

func (pl *GetAttributeListRequestPayload) Operation() kmip.Operation {
	return kmip.OperationGetAttributeList
}

// Response for the get-attribute-list operation.
type GetAttributeListResponsePayload struct {
	// The Unique Identifier of the object.
	UniqueIdentifier string
	// The names of the available attributes associated with the object.
	AttributeName []kmip.AttributeName
}

func (pl *GetAttributeListResponsePayload) Operation() kmip.Operation {
	return kmip.OperationGetAttributeList
}
