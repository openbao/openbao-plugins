package kmip

import (
	"encoding/binary"
	"time"

	"github.com/ovh/kmip-go/ttlv"
)

type RequestMessage struct {
	Header    RequestHeader
	BatchItem []RequestBatchItem
}

func NewRequestMessage(version ProtocolVersion, payloads ...OperationPayload) RequestMessage {
	timestamp := time.Now().Truncate(time.Second)
	msg := RequestMessage{
		Header: RequestHeader{
			ProtocolVersion: version,
			TimeStamp:       &timestamp,
			BatchCount:      int32(len(payloads)),
		},
	}

	for i, pl := range payloads {
		item := RequestBatchItem{
			Operation:      pl.Operation(),
			RequestPayload: pl,
		}
		if len(payloads) > 1 {
			item.UniqueBatchItemID = binary.BigEndian.AppendUint64(item.UniqueBatchItemID, uint64(i))
		}
		msg.BatchItem = append(msg.BatchItem, item)
	}

	return msg
}

type RequestHeader struct {
	ProtocolVersion     ProtocolVersion `ttlv:",set-version"`
	MaximumResponseSize int32           `ttlv:",omitempty"`

	ClientCorrelationValue       string `ttlv:",omitempty,version=v1.4.."`
	ServerCorrelationValue       string `ttlv:",omitempty,version=v1.4.."`
	AsynchronousIndicator        *bool
	AttestationCapableIndicator  *bool             `ttlv:",version=v1.2.."`
	AttestationType              []AttestationType `ttlv:",version=v1.2.."`
	Authentication               *Authentication
	BatchErrorContinuationOption BatchErrorContinuationOption `ttlv:",omitempty"`
	BatchOrderOption             *bool
	TimeStamp                    *time.Time
	BatchCount                   int32
}

type RequestBatchItem struct {
	Operation         Operation
	UniqueBatchItemID []byte `ttlv:",omitempty"`
	RequestPayload    OperationPayload
	MessageExtension  *MessageExtension
}

func (pv *RequestBatchItem) TagEncodeTTLV(e *ttlv.Encoder, tag int) {
	e.Struct(tag, func(e *ttlv.Encoder) {
		e.Any(pv.Operation)
		if len(pv.UniqueBatchItemID) > 0 {
			e.ByteString(TagUniqueBatchItemID, pv.UniqueBatchItemID)
		}
		e.TagAny(TagRequestPayload, pv.RequestPayload)
		e.Any(pv.MessageExtension)
	})
}

func (pv *RequestBatchItem) TagDecodeTTLV(d *ttlv.Decoder, tag int) error {
	return d.Struct(tag, func(d *ttlv.Decoder) error {
		if err := d.Any(&pv.Operation); err != nil {
			return err
		}
		if err := d.Opt(TagUniqueBatchItemID, &pv.UniqueBatchItemID); err != nil {
			return err
		}
		pv.RequestPayload = newRequestPayload(pv.Operation)
		if err := d.TagAny(TagRequestPayload, &pv.RequestPayload); err != nil {
			return err
		}
		return d.Opt(TagMessageExtension, &pv.MessageExtension)
	})
}
