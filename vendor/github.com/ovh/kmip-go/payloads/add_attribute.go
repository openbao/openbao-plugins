package payloads

import "github.com/ovh/kmip-go"

func init() {
	kmip.RegisterOperationPayload[AddAttributeRequestPayload, AddAttributeResponsePayload](kmip.OperationAddAttribute)
}

// This operation requests the server to add a new attribute instance to be associated with a Managed Object and set its value.
// The request contains the Unique Identifier of the Managed Object to which the attribute pertains, along with the attribute name and value.
// For single-instance attributes, this is how the attribute value is created.
// For multi-instance attributes, this is how the first and subsequent values are created.
// Existing attribute values SHALL only be changed by the Modify Attribute operation. Read-Only attributes SHALL NOT be added using the Add Attribute operation.
// The Attribute Index SHALL NOT be specified in the request. The response returns a new Attribute Index and the Attribute Index MAY be omitted if the index
// of the added attribute instance is 0. Multiple Add Attribute requests MAY be included in a single batched request to add multiple attributes.
type AddAttributeRequestPayload struct {
	// The Unique Identifier of the object.
	// If omitted, then the ID Placeholder value is used by the server as the Unique Identifier.
	UniqueIdentifier string `ttlv:",omitempty"`
	// Specifies the attribute to be added as an attribute for the object.
	Attribute kmip.Attribute
}

func (pl *AddAttributeRequestPayload) Operation() kmip.Operation {
	return kmip.OperationAddAttribute
}

// Response for the add-attribute operation.
type AddAttributeResponsePayload struct {
	// The Unique Identifier of the object.
	UniqueIdentifier string
	// The added attribute associated with the object.
	Attribute kmip.Attribute
}

func (pl *AddAttributeResponsePayload) Operation() kmip.Operation {
	return kmip.OperationAddAttribute
}
