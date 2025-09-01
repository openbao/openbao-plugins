package payloads

import (
	"time"

	"github.com/ovh/kmip-go"
)

func init() {
	kmip.RegisterOperationPayload[RekeyRequestPayload, RekeyResponsePayload](kmip.OperationReKey)
}

// This request is used to generate a replacement key for an existing symmetric key. It is analogous to the Create operation,
// except that attributes of the replacement key are copied from the existing key, with the exception of the attributes listed in Random Number Generator 3.44.
//
// As the replacement key takes over the name attribute of the existing key, Re-key SHOULD only be performed once on a given key.
//
// The server SHALL copy the Unique Identifier of the replacement key returned by this operation into the ID Placeholder variable.
//
// For the existing key, the server SHALL create a Link attribute of Link Type Replacement Object pointing to the replacement key.
// For the replacement key, the server SHALL create a Link attribute of Link Type Replaced Key pointing to the existing key.
//
// An Offset MAY be used to indicate the difference between the Initialization Date and the Activation Date of the replacement key.
// If no Offset is specified, the Activation Date, Process Start Date, Protect Stop Date and Deactivation Date values are copied from the existing key.
//
// If Offset is set and dates exist for the existing key, then the dates of the replacement key SHALL be set based on the dates of the existing key as follows:
//   - Initial Date (IT1) -> Initial Date (IT2) > IT1
//   - Activation Date (AT1) -> Activation Date (AT2) =  IT2+ Offset
//   - Process Start Date (CT1) -> Process Start Date = CT1+(AT2- AT1)
//   - Protect Stop Date (TT1) -> Protect Stop Date = TT1+(AT2- AT1)
//   - Deactivation Date (DT1) -> Deactivation Date = DT1+(AT2- AT1)
//
// Attributes requiring special handling when creating the replacement key are:
//   - Initial Date: Set to the current time
//   - Destroy Date: Not set
//   - Compromise Occurrence Date: Not set
//   - Compromise Date: Not set
//   - Revocation Reason: Not set
//   - Unique Identifier: New value generated
//   - Usage Limits: The Total value is copied from the existing key, and the Count value in the existing key is set to the Total value.
//   - Name: Set to the name(s) of the existing key; all name attributes are removed from the existing key.
//   - State: Set based on attributes values, such as dates
//   - Digest: Recomputed from the replacement key value
//   - Link: Set to point to the existing key as the replaced key
//   - Last Change Date: Set to current time
//   - Random Number Generator: Set to the random number generator used for creating the new managed object. Not copied from the original object.
type RekeyRequestPayload struct {
	// Determines the existing Symmetric Key being re-keyed. If omitted, then the ID Placeholder value is used by the server as the Unique Identifier.
	UniqueIdentifier string `ttlv:",omitempty"`
	// An Interval object indicating the difference between the Initialization Date and the Activation Date of the replacement key to be created.
	Offset *time.Duration
	// Specifies desired object attributes using templates and/or individual attributes.
	//
	// The Template Managed Object is deprecated as of version 1.3 of this specification and MAY be removed from subsequent versions of the specification.
	// Individual Attributes SHOULD be used in operations which currently support use of a Name within a Template-Attribute to reference a Template.
	TemplateAttribute *kmip.TemplateAttribute
}

func (a *RekeyRequestPayload) Operation() kmip.Operation {
	return kmip.OperationReKey
}

// Response for the Re-Key operation.
type RekeyResponsePayload struct {
	// The Unique Identifier of the newly-created replacement Symmetric Key.
	UniqueIdentifier string
	// An OPTIONAL list of object attributes with values that were not specified in the request, but have been implicitly set by the key management server.
	//
	// The Template Managed Object is deprecated as of version 1.3 of this specification and MAY be removed from subsequent versions of the specification.
	// Individual Attributes SHOULD be used in operations which currently support use of a Name within a Template-Attribute to reference a Template.
	TemplateAttribute *kmip.TemplateAttribute
}

func (a *RekeyResponsePayload) Operation() kmip.Operation {
	return kmip.OperationReKey
}
