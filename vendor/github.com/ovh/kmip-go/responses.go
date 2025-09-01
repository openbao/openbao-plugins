package kmip

import (
	"fmt"
	"time"

	"github.com/ovh/kmip-go/ttlv"
)

type ResponseMessage struct {
	Header    ResponseHeader
	BatchItem []ResponseBatchItem
}

type ResponseHeader struct {
	ProtocolVersion        ProtocolVersion `ttlv:",set-version"`
	TimeStamp              time.Time
	Nonce                  *Nonce            `ttlv:",version=v1.2.."`
	AttestationType        []AttestationType `ttlv:",version=v1.2.."`
	ClientCorrelationValue string            `ttlv:",omitempty,version=v1.4.."`
	ServerCorrelationValue string            `ttlv:",omitempty,version=v1.4.."`
	BatchCount             int32
}

type ResponseBatchItem struct {
	Operation                    Operation `ttlv:",omitempty"`
	UniqueBatchItemID            []byte    `ttlv:",omitempty"`
	ResultStatus                 ResultStatus
	ResultReason                 ResultReason `ttlv:",omitempty"`
	ResultMessage                string       `ttlv:",omitempty"`
	AsynchronousCorrelationValue []byte       `ttlv:",omitempty"`
	ResponsePayload              OperationPayload
	MessageExtension             *MessageExtension
}

func (bi *ResponseBatchItem) Err() error {
	if bi.ResultStatus != ResultStatusSuccess {
		msg := bi.ResultMessage
		return fmt.Errorf("Operation failed (status=%q, reason=%q) %s", ttlv.EnumStr(bi.ResultStatus), ttlv.EnumStr(bi.ResultReason), msg)
	}
	return nil
}

func (pv *ResponseBatchItem) TagEncodeTTLV(e *ttlv.Encoder, tag int) {
	e.Struct(TagBatchItem, func(e *ttlv.Encoder) {
		e.Any(pv.Operation)
		if len(pv.UniqueBatchItemID) > 0 {
			e.ByteString(TagUniqueBatchItemID, pv.UniqueBatchItemID)
		}
		e.Any(pv.ResultStatus)
		if pv.ResultStatus != ResultStatusSuccess || pv.ResultReason != 0 {
			e.Any(pv.ResultReason)
		}
		if pv.ResultMessage != "" {
			e.TextString(TagResultMessage, pv.ResultMessage)
		}
		if len(pv.AsynchronousCorrelationValue) > 0 {
			e.ByteString(TagAsynchronousCorrelationValue, pv.AsynchronousCorrelationValue)
		}
		e.TagAny(TagResponsePayload, pv.ResponsePayload)
		if pv.MessageExtension != nil {
			e.Any(pv.MessageExtension)
		}
	})
}

func (pv *ResponseBatchItem) TagDecodeTTLV(d *ttlv.Decoder, tag int) error {
	return d.Struct(tag, func(d *ttlv.Decoder) error {
		if err := d.Opt(TagOperation, &pv.Operation); err != nil {
			return err
		}
		if err := d.Opt(TagUniqueBatchItemID, &pv.UniqueBatchItemID); err != nil {
			return err
		}

		if err := d.TagAny(TagResultStatus, &pv.ResultStatus); err != nil {
			return err
		}

		if err := d.Opt(TagResultReason, &pv.ResultReason); err != nil {
			return err
		}
		if err := d.Opt(TagResultMessage, &pv.ResultMessage); err != nil {
			return err
		}
		if err := d.Opt(TagAsynchronousCorrelationValue, &pv.AsynchronousCorrelationValue); err != nil {
			return err
		}
		if pv.Operation > 0 && d.Tag() == TagResponsePayload {
			pv.ResponsePayload = newResponsePayload(pv.Operation)
			return d.TagAny(TagResponsePayload, &pv.ResponsePayload)
		}
		return d.Opt(TagMessageExtension, &pv.MessageExtension)
	})
}
