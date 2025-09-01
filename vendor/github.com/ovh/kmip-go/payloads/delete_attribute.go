package payloads

import "github.com/ovh/kmip-go"

func init() {
	kmip.RegisterOperationPayload[DeleteAttributeRequestPayload, DeleteAttributeResponsePayload](kmip.OperationDeleteAttribute)
}

// This operation requests the server to delete an attribute associated with a Managed Object.
// The request contains the Unique Identifier of the Managed Object whose attribute is to be deleted, the attribute name,
// and the OPTIONAL Attribute Index of the attribute. If no Attribute Index is specified in the request,
// then the Attribute Index SHALL be assumed to be 0. Attributes that are always REQUIRED to have a value SHALL never be deleted by this operation.
// Attempting to delete a non-existent attribute or specifying an Attribute Index for which there exists no Attribute Value SHALL result in an error.
// The response returns the deleted Attribute and the Attribute Index MAY be omitted if the index of the deleted attribute instance is 0. Multiple Delete Attribute
// requests MAY be included in a single batched request to delete multiple attributes.
type DeleteAttributeRequestPayload struct {
	// Determines the object whose attributes are being deleted. If omitted, then the ID Placeholder value is used by the server as the Unique Identifier.
	UniqueIdentifier string `ttlv:",omitempty"`
	// Specifies the name of the attribute associated with the object to be deleted.
	AttributeName kmip.AttributeName
	// Specifies the Index of the Attribute.
	AttributeIndex *int32
}

func (pl *DeleteAttributeRequestPayload) Operation() kmip.Operation {
	return kmip.OperationDeleteAttribute
}

// Response for the delete-attribute operation.
type DeleteAttributeResponsePayload struct {
	// The Unique Identifier of the object.
	UniqueIdentifier string
	// The deleted attribute associated with the object.
	Attribute kmip.Attribute
}

func (pl *DeleteAttributeResponsePayload) Operation() kmip.Operation {
	return kmip.OperationDeleteAttribute
}
