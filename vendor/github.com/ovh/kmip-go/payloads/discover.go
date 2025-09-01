package payloads

import (
	"github.com/ovh/kmip-go"
)

func init() {
	kmip.RegisterOperationPayload[DiscoverVersionsRequestPayload, DiscoverVersionsResponsePayload](kmip.OperationDiscoverVersions)
}

// This operation is used by the client to determine a list of protocol versions that is supported by the server. The request payload contains an OPTIONAL
// list of protocol versions that is supported by the client. The protocol versions SHALL be ranked in decreasing order of preference.
//
// The response payload contains a list of protocol versions that are supported by the server. The protocol versions are ranked in decreasing order of preference.
// If the client provides the server with a list of supported protocol versions in the request payload, the server SHALL return
// only the protocol versions that are supported by both the client and server. The server SHOULD list all the protocol versions supported by both client and server.
// If the protocol version specified in the request header is not specified in the request payload and the server does not support any
// protocol version specified in the request payload, the server SHALL return an empty list in the response payload. If no protocol versions are specified in the request payload,
// the server SHOULD return all the protocol versions that are supported by the server.
type DiscoverVersionsRequestPayload struct {
	// The list of protocol versions supported by the client ordered in decreasing order of preference.
	ProtocolVersion []kmip.ProtocolVersion
}

func (*DiscoverVersionsRequestPayload) Operation() kmip.Operation {
	return kmip.OperationDiscoverVersions
}

// Response for the discover-versions operation.
type DiscoverVersionsResponsePayload struct {
	// The list of protocol versions supported by the server ordered in decreasing order of preference.
	ProtocolVersion []kmip.ProtocolVersion
}

func (*DiscoverVersionsResponsePayload) Operation() kmip.Operation {
	return kmip.OperationDiscoverVersions
}
