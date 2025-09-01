package payloads

import "github.com/ovh/kmip-go"

func init() {
	kmip.RegisterOperationPayload[ModifyAttributeRequestPayload, ModifyAttributeResponsePayload](kmip.OperationModifyAttribute)
}

// This operation requests the server to modify the value of an existing attribute instance associated with a Managed Object.
// The request contains the Unique Identifier of the Managed Object whose attribute is to be modified, the attribute name,
// the OPTIONAL Attribute Index, and the new value. If no Attribute Index is specified in the request,
// then the Attribute Index SHALL be assumed to be 0. Only existing attributes MAY be changed via this operation.
// New attributes SHALL only be added by the Add Attribute operation. Only the specified instance of the attribute SHALL be modified.
// Specifying an Attribute Index for which there exists no Attribute object SHALL result in an error.
//
//	The response returns the modified Attribute (new value) and the Attribute Index MAY be omitted if the index of the modified attribute instance is 0.
//
// Multiple Modify Attribute requests MAY be included in a single batched request to modify multiple attributes.
type ModifyAttributeRequestPayload struct {
	// The Unique Identifier of the object. If omitted, then the ID Placeholder value is used by the server as the Unique Identifier.
	UniqueIdentifier string `ttlv:",omitempty"`
	// Specifies the attribute associated with the object to be modified.
	Attribute kmip.Attribute
}

func (pl *ModifyAttributeRequestPayload) Operation() kmip.Operation {
	return kmip.OperationModifyAttribute
}

// Response for the Modify-Attribute operation.
type ModifyAttributeResponsePayload struct {
	// The Unique Identifier of the object.
	UniqueIdentifier string
	// The modified attribute associated with the object with the new value.
	Attribute kmip.Attribute
}

func (pl *ModifyAttributeResponsePayload) Operation() kmip.Operation {
	return kmip.OperationModifyAttribute
}
