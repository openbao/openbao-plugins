package kmip

import (
	"reflect"

	"github.com/ovh/kmip-go/ttlv"
)

func init() {
	ttlv.RegisterEnum(TagOperation, map[Operation]string{
		OperationCreate:             "Create",
		OperationCreateKeyPair:      "CreateKeyPair",
		OperationRegister:           "Register",
		OperationReKey:              "ReKey",
		OperationDeriveKey:          "DeriveKey",
		OperationCertify:            "Certify",
		OperationReCertify:          "ReCertify",
		OperationLocate:             "Locate",
		OperationCheck:              "Check",
		OperationGet:                "Get",
		OperationGetAttributes:      "GetAttributes",
		OperationGetAttributeList:   "GetAttributeList",
		OperationAddAttribute:       "AddAttribute",
		OperationModifyAttribute:    "ModifyAttribute",
		OperationDeleteAttribute:    "DeleteAttribute",
		OperationObtainLease:        "ObtainLease",
		OperationGetUsageAllocation: "GetUsageAllocation",
		OperationActivate:           "Activate",
		OperationRevoke:             "Revoke",
		OperationDestroy:            "Destroy",
		OperationArchive:            "Archive",
		OperationRecover:            "Recover",
		OperationValidate:           "Validate",
		OperationQuery:              "Query",
		OperationCancel:             "Cancel",
		OperationPoll:               "Poll",
		OperationNotify:             "Notify",
		OperationPut:                "Put",

		// KMIP 1.1
		OperationReKeyKeyPair:     "ReKeyKeyPair",
		OperationDiscoverVersions: "DiscoverVersions",

		// KMIP 1.2
		OperationEncrypt:         "Encrypt",
		OperationDecrypt:         "Decrypt",
		OperationSign:            "Sign",
		OperationSignatureVerify: "SignatureVerify",
		OperationMAC:             "MAC",
		OperationMACVerify:       "MACVerify",
		OperationRNGRetrieve:     "RNGRetrieve",
		OperationRNGSeed:         "RNGSeed",
		OperationHash:            "Hash",
		OperationCreateSplitKey:  "CreateSplitKey",
		OperationJoinSplitKey:    "JoinSplitKey",

		// KMIP 1.4
		OperationImport: "Import",
		OperationExport: "Export",
	})
}

type Operation uint32

const (
	OperationCreate             Operation = 0x00000001
	OperationCreateKeyPair      Operation = 0x00000002
	OperationRegister           Operation = 0x00000003
	OperationReKey              Operation = 0x00000004
	OperationDeriveKey          Operation = 0x00000005
	OperationCertify            Operation = 0x00000006
	OperationReCertify          Operation = 0x00000007
	OperationLocate             Operation = 0x00000008
	OperationCheck              Operation = 0x00000009
	OperationGet                Operation = 0x0000000A
	OperationGetAttributes      Operation = 0x0000000B
	OperationGetAttributeList   Operation = 0x0000000C
	OperationAddAttribute       Operation = 0x0000000D
	OperationModifyAttribute    Operation = 0x0000000E
	OperationDeleteAttribute    Operation = 0x0000000F
	OperationObtainLease        Operation = 0x00000010
	OperationGetUsageAllocation Operation = 0x00000011
	OperationActivate           Operation = 0x00000012
	OperationRevoke             Operation = 0x00000013
	OperationDestroy            Operation = 0x00000014
	OperationArchive            Operation = 0x00000015
	OperationRecover            Operation = 0x00000016
	OperationValidate           Operation = 0x00000017
	OperationQuery              Operation = 0x00000018
	OperationCancel             Operation = 0x00000019
	OperationPoll               Operation = 0x0000001A
	OperationNotify             Operation = 0x0000001B
	OperationPut                Operation = 0x0000001C

	// KMIP 1.1.
	OperationReKeyKeyPair     Operation = 0x0000001D
	OperationDiscoverVersions Operation = 0x0000001E

	// KMIP 1.2.
	OperationEncrypt         Operation = 0x0000001F
	OperationDecrypt         Operation = 0x00000020
	OperationSign            Operation = 0x00000021
	OperationSignatureVerify Operation = 0x00000022
	OperationMAC             Operation = 0x00000023
	OperationMACVerify       Operation = 0x00000024
	OperationRNGRetrieve     Operation = 0x00000025
	OperationRNGSeed         Operation = 0x00000026
	OperationHash            Operation = 0x00000027
	OperationCreateSplitKey  Operation = 0x00000028
	OperationJoinSplitKey    Operation = 0x00000029

	// KMIP 1.4.
	OperationImport Operation = 0x0000002A
	OperationExport Operation = 0x0000002B
)

func (enum Operation) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}

type operationPayloadTypes struct {
	request  reflect.Type
	response reflect.Type
}

func (opt *operationPayloadTypes) newRequest() OperationPayload {
	return reflect.New(opt.request).Interface().(OperationPayload)
}

func (opt *operationPayloadTypes) newResponse() OperationPayload {
	return reflect.New(opt.response).Interface().(OperationPayload)
}

func typeForOperation[Req, Resp any]() operationPayloadTypes {
	return operationPayloadTypes{
		request:  reflect.TypeFor[Req](),
		response: reflect.TypeFor[Resp](),
	}
}

var operationRegistry = map[Operation]operationPayloadTypes{}

func RegisterOperationPayload[Req, Resp any](op Operation) {
	operationRegistry[op] = typeForOperation[Req, Resp]()
}

func newRequestPayload(op Operation) OperationPayload {
	types, ok := operationRegistry[op]
	if !ok {
		return &UnknownPayload{
			opType: op,
		}
	}
	return types.newRequest()
}

func newResponsePayload(op Operation) OperationPayload {
	types, ok := operationRegistry[op]
	if !ok {
		return &UnknownPayload{
			opType: op,
		}
	}
	return types.newResponse()
}

type OperationPayload interface {
	Operation() Operation
}

type UnknownPayload struct {
	opType Operation `ttlv:"-"`
	Fields ttlv.Struct
}

func NewUnknownPayload(op Operation, fields ...ttlv.Value) *UnknownPayload {
	return &UnknownPayload{
		opType: op,
		Fields: ttlv.Struct(fields),
	}
}

func (pl *UnknownPayload) Operation() Operation {
	return pl.opType
}

func (v *UnknownPayload) TagEncodeTTLV(e *ttlv.Encoder, tag int) {
	v.Fields.TagEncodeTTLV(e, tag)
}

func (v *UnknownPayload) TagDecodeTTLV(d *ttlv.Decoder, tag int) error {
	return v.Fields.TagDecodeTTLV(d, tag)
}
