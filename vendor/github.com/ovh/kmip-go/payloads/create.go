package payloads

import "github.com/ovh/kmip-go"

func init() {
	kmip.RegisterOperationPayload[CreateRequestPayload, CreateResponsePayload](kmip.OperationCreate)
}

// This operation requests the server to generate a new symmetric key as a Managed Cryptographic Object.
// This operation is not used to create a Template object (see Register operation, Section 4.3).
//
// The request contains information about the type of object being created, and some of the attributes to be assigned to the object
// (e.g., Cryptographic Algorithm, Cryptographic Length, etc.). This information MAY be specified by the names of Template objects that already exist.
//
// The response contains the Unique Identifier of the created object. The server SHALL copy the Unique Identifier returned by this operation into the ID Placeholder variable.
type CreateRequestPayload struct {
	// Determines the type of object to be created.
	ObjectType kmip.ObjectType
	// Specifies desired attributes using to be associated with the new object templates and/or individual attributes.
	//
	// The Template Managed Object is deprecated as of version 1.3 of this specification and MAY be removed from subsequent versions of the specification.
	// Individual Attributes SHOULD be used in operations which currently support use of a Name within a Template-Attribute to reference a Template.
	TemplateAttribute kmip.TemplateAttribute
}

func (a *CreateRequestPayload) Operation() kmip.Operation {
	return kmip.OperationCreate
}

// Response for the create operation.
type CreateResponsePayload struct {
	// Type of object created.
	ObjectType kmip.ObjectType
	// The Unique Identifier of the newly created object.
	UniqueIdentifier string
	// An OPTIONAL list of object attributes with values that were not specified in the request, but have been implicitly set by the key management server.
	//
	// The Template Managed Object is deprecated as of version 1.3 of this specification and MAY be removed from subsequent versions of the specification.
	// Individual Attributes SHOULD be used in operations which currently support use of a Name within a Template-Attribute to reference a Template.
	Attributes *kmip.TemplateAttribute
}

func (a *CreateResponsePayload) Operation() kmip.Operation {
	return kmip.OperationCreate
}
