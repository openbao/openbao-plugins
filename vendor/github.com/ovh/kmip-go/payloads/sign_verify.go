package payloads

import "github.com/ovh/kmip-go"

func init() {
	kmip.RegisterOperationPayload[SignRequestPayload, SignResponsePayload](kmip.OperationSign)
	kmip.RegisterOperationPayload[SignatureVerifyRequestPayload, SignatureVerifyResponsePayload](kmip.OperationSignatureVerify)
}

// This operation requests the server to perform a signature operation on the provided data using a
// Managed Cryptographic Object as the key for the signature operation.
// The request contains information about the cryptographic parameters (digital signature algorithm or
// cryptographic algorithm and hash algorithm) and the data to be signed. The cryptographic parameters
// MAY be omitted from the request as they can be specified as associated attributes of the Managed
// Cryptographic Object.
//
// If the Managed Cryptographic Object referenced has a Usage Limits attribute then the server SHALL
// obtain an allocation from the current Usage Limits value prior to performing the signing operation. If the
// allocation is unable to be obtained the operation SHALL return with a result status of Operation Failed
// and result reason of Permission Denied.
type SignRequestPayload struct {
	// The Unique Identifier of the Managed Cryptographic Object that is the key to use for the signature operation. If
	// omitted, then the ID Placeholder value SHALL be used by the server as the Unique Identifier.
	UniqueIdentifier string `ttlv:",omitempty"`
	// The Cryptographic Parameters (Digital Signature Algorithm or Cryptographic Algorithm and Hashing Algorithm) corresponding
	// to the particular signature generation method requested. If omitted then the Cryptographic Parameters associated
	// with the Managed Cryptographic Object with the lowest Attribute Index SHALL be used.
	// If there are no Cryptographic Parameters associated with the Managed Cryptographic Object and the algorithm requires parameters then
	// the operation SHALL return with a Result Status of Operation Failed.
	CryptographicParameters *kmip.CryptographicParameters
	// The data to be signed. Mandatory for kmip 1.2 or single-part operation, unless Digested Data is supplied. Optional for multi-part.
	Data []byte `ttlv:",omitempty"`
	// The digested data to be signed.
	DigestedData []byte `ttlv:",omitempty,version=v1.4.."`
	// Specifies the existing stream or by parts cryptographic operation (as returned from a previous call to this operation).
	CorrelationValue []byte `ttlv:",omitempty,version=v1.3.."`
	// Initial operation.
	InitIndicator *bool `ttlv:",version=v1.3.."`
	// Final operation.
	FinalIndicator *bool `ttlv:",version=v1.3.."`
}

func (pl *SignRequestPayload) Operation() kmip.Operation {
	return kmip.OperationSign
}

// Response for the sign operation.
//
// The response contains the Unique Identifier of the Managed Cryptographic Object used as the key and
// the result of the signature operation.
//
// The success or failure of the operation is indicated by the Result Status (and if failure the Result Reason)
// in the response header.
type SignResponsePayload struct {
	// The Unique Identifier of the Managed Cryptographic Object that is the key used for the signature operation.
	UniqueIdentifier string
	// The signed data. Mandatory for kmip 1.2 or single-part operation, not for multi-part.
	SignatureData []byte `ttlv:",omitempty"`
	// Specifies the stream or by-parts value to be provided in subsequent calls to this operation for performing cryptographic operations.
	CorrelationValue []byte `ttlv:",omitempty,version=v1.3.."`
}

func (pl *SignResponsePayload) Operation() kmip.Operation {
	return kmip.OperationSign
}

// This operation requests the server to perform a signature verify operation on the provided data using a
// Managed Cryptographic Object as the key for the signature verification operation.
// The request contains information about the cryptographic parameters (digital signature algorithm or
// cryptographic algorithm and hash algorithm) and the signature to be verified and MAY contain the data
// that was passed to the signing operation (for those algorithms which need the original data to verify a
// signature).
//
// The cryptographic parameters MAY be omitted from the request as they can be specified as associated
// attributes of the Managed Cryptographic Object.
type SignatureVerifyRequestPayload struct {
	// The Unique Identifier of the Managed Cryptographic Object that is the key to use for the signature verify operation.
	// If omitted, then the ID Placeholder value SHALL be used by the server as the Unique Identifier.
	UniqueIdentifier string `ttlv:",omitempty"`
	// The Cryptographic Parameters (Digital Signature Algorithm or Cryptographic Algorithm and Hashing Algorithm)
	// corresponding to the particular signature verification method requested. If omitted then the Cryptographic
	// Parameters associated with the Managed Cryptographic Object with the lowest Attribute Index SHALL be used.
	//
	// If there are no Cryptographic Parameters associated with the Managed Cryptographic Object and the algorithm requires
	// parameters then the operation SHALL return with a Result Status of Operation Failed.
	CryptographicParameters *kmip.CryptographicParameters
	// The data that was signed.
	Data []byte `ttlv:",omitempty"`
	// The digested data to be verified.
	DigestedData []byte `ttlv:",omitempty,version=v1.4.."`
	// The signature to be verified. Mandatory for kmip 1.2 or for single-part operation. Not for multi-part.
	SignatureData []byte `ttlv:",omitempty"`
	// Specifies the existing stream or by-parts cryptographic operation (as returned from a previous call to this operation).
	CorrelationValue []byte `ttlv:",omitempty,version=v1.3.."`
	// Initial operation.
	InitIndicator *bool `ttlv:",version=v1.3.."`
	// Final operation.
	FinalIndicator *bool `ttlv:",version=v1.3.."`
}

func (pl *SignatureVerifyRequestPayload) Operation() kmip.Operation {
	return kmip.OperationSignatureVerify
}

// Response for SignatureVerify operation.
//
// The response contains the Unique Identifier of the Managed Cryptographic Object used as the key and
// the OPTIONAL data recovered from the signature (for those signature algorithms where data recovery
// from the signature is supported). The validity of the signature is indicated by the Validity Indicator field.
//
// The success or failure of the operation is indicated by the Result Status (and if failure the Result Reason)
// in the response header.
type SignatureVerifyResponsePayload struct {
	// The Unique Identifier of the Managed Cryptographic Object that is the key used for the verification operation.
	UniqueIdentifier string
	// An Enumeration object indicating whether the signature is valid, invalid, or unknown.
	ValidityIndicator kmip.ValidityIndicator
	// The OPTIONAL recovered data (as a Byte String) for those signature algorithms where data recovery from the signature is supported.
	Data []byte `ttlv:",omitempty"`
	// Specifies the stream or by-parts value to be provided in subsequent calls to this operation for performing cryptographic operations.
	CorrelationValue []byte `ttlv:",omitempty,version=v1.3.."`
}

func (pl *SignatureVerifyResponsePayload) Operation() kmip.Operation {
	return kmip.OperationSignatureVerify
}
