package payloads

import (
	"github.com/ovh/kmip-go"
	"github.com/ovh/kmip-go/ttlv"
)

func init() {
	kmip.RegisterOperationPayload[QueryRequestPayload, QueryResponsePayload](kmip.OperationQuery)
}

// This operation is used by the client to interrogate the server to determine its capabilities and/or protocol mechanisms. The Query operation SHOULD be invocable by
// unauthenticated clients to interrogate server features and functions. The Query Function field in the request SHALL contain one or more of the following items:
//   - Query Operations
//   - Query Objects
//   - Query Server Information
//   - Query Application Namespaces
//   - Query Extension List
//   - Query Extension Map
//   - Query Attestation Types
//   - Query RNGs
//   - Query Validations
//   - Query Profiles
//   - Query Capabilities
//   - Query Client Registration Methods
//
// The Operation fields in the response contain Operation enumerated values, which SHALL list all the operations that the server supports.
// If the request contains a Query Operations value in the Query Function field, then these fields SHALL be returned in the response.
//
// The Object Type fields in the response contain Object Type enumerated values, which SHALL list all the object types that the server supports.
// If the request contains a Query Objects value in the Query Function field, then these fields SHALL be returned in the response.
//
// The Server Information field in the response is a structure containing vendor-specific fields and/or substructures.
// If the request contains a Query Server Information value in the Query Function field, then this field SHALL be returned in the response.
//
// The Application Namespace fields in the response contain the namespaces that the server SHALL generate values for if requested by the client (see Section 3.36).
// These fields SHALL only be returned in the response if the request contains a Query Application Namespaces value in the Query Function field.
//
// The Extension Information fields in the response contain the descriptions of Objects with Item Tag values in the Extensions range that are supported
// by the server (see Section 2.1.9). If the request contains a Query Extension List and/or Query Extension Map value in the Query Function field,
// then the Extensions Information fields SHALL be returned in the response. If the Query Function field contains the Query Extension Map value,
// then the Extension Tag and Extension Type fields SHALL be specified in the Extension Information values. If both Query Extension List and Query Extension Map
// are specified in the request, then only the response to Query Extension Map SHALL be returned and the Query Extension List SHALL be ignored.
//
// The Attestation Type fields in the response contain Attestation Type enumerated values, which SHALL list all the attestation types that the server supports.
// If the request contains a Query Attestation Types value in the Query Function field, then this field SHALL be returned in the response if the server supports any Attestation Types.
//
// The RNG Parameters fields in the response SHALL list all the Random Number Generators that the server supports.
// If the request contains a Query RNGs value in the Query Function field, then this field SHALL be returned in the response.
// If the server is unable to specify details of the RNG then it SHALL return an RNG Parameters with the RNG Algorithm enumeration of Unspecified.
//
// The Validation Information field in the response is a structure containing details of each formal validation which the server asserts.
// If the request contains a Query Validations value, then zero or more Validation Information fields SHALL be returned in the response.
// A server MAY elect to return no validation information in the response.
//
// A Profile Information field in the response is a structure containing details of the profiles that a server supports including potentially how it supports that profile.
// If the request contains a Query Profiles value in the Query Function field, then this field SHALL be returned in the response if the server supports any Profiles.
//
// The Capability Information fields in the response contain details of the capability of the server.
//
// The Client Registration Method fields in the response contain Client Registration Method enumerated values, which SHALL list all the client
// registration methods that the server supports. If the request contains a Query Client Registration Methods value in the Query Function field,
// then this field SHALL be returned in the response if the server supports any Client Registration Methods.
//
// Note that the response payload is empty if there are no values to return.
type QueryRequestPayload struct {
	// Determines the information being queried.
	QueryFunction []kmip.QueryFunction
}

func (pl *QueryRequestPayload) Operation() kmip.Operation {
	return kmip.OperationQuery
}

// Response for the Query operation.
type QueryResponsePayload struct {
	// Specifies an Operation that is supported by the server.
	Operations []kmip.Operation `ttlv:"Operation"`
	// Specifies a Managed Object Type that is supported by the server.
	ObjectType []kmip.ObjectType
	// SHALL be returned if Query Server Information is requested. The Vendor Identification SHALL be a text string that uniquely identifies the vendor.
	VendorIdentification string `ttlv:",omitempty"`
	// Contains vendor-specific information possibly be of interest to the client.
	ServerInformation *ttlv.Value
	// Specifies an Application Namespace supported by the server.
	ApplicationNamespace []string
	// SHALL be returned if Query Extension List or Query Extension Map is requested and supported by the server.
	ExtensionInformation []kmip.ExtensionInformation `ttlv:",version=v1.1.."`
	// Specifies an Attestation Type that is supported by the server.
	AttestationType []kmip.AttestationType `ttlv:",version=v1.2.."`
	// Specifies the RNG that is supported by the server.
	RNGParameters []kmip.RNGParameters `ttlv:",version=v1.3.."`
	// Specifies the Profiles that are supported by the server.
	ProfileInformation []kmip.ProfileInformation `ttlv:",version=v1.3.."`
	// Specifies the validations that are supported by the server.
	ValidationInformation []kmip.ValidationInformation `ttlv:",version=v1.3.."`
	// Specifies the capabilities that are supported by the server.
	CapabilityInformation []kmip.CapabilityInformation `ttlv:",version=v1.3.."`
	// Specifies a Client Registration Method that is supported by the server.
	ClientRegistrationMethod []kmip.ClientRegistrationMethod `ttlv:",version=v1.3.."`
}

func (pl *QueryResponsePayload) Operation() kmip.Operation {
	return kmip.OperationQuery
}
