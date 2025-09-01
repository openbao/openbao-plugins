package ttlv

import (
	"io"
	"slices"
)

func computeNeededBytes(buf []byte) int {
	if len(buf) < 8 {
		return 8
	}
	dec := ttlvReader{buf: buf}
	return 8 + dec.paddedLen()
}

// Stream is a helper type to wrap io.ReadWrite stream to serialize and deserialize
// binary TTLV encoded golang types to / from the stream.
type Stream struct {
	inner io.ReadWriteCloser
	max   int
}

// NewStream creates a new TTLV stream around the given I/O stream.
// If maxSize is greater than 0, then it will limit the maximum allowed
// size in bytes for a message to receive.
func NewStream(inner io.ReadWriteCloser, maxSize int) Stream {
	return Stream{
		inner: inner,
		max:   maxSize,
	}
}

// Close wloses the inner stream.
func (s *Stream) Close() error {
	if s.inner == nil {
		return nil
	}
	return s.inner.Close()
}

// Send serializes to TTLV binary the given `msg` then writes it to the inner stream.
func (s *Stream) Send(msg any) error {
	data := MarshalTTLV(msg)
	_, err := s.inner.Write(data)
	return err
}

// Recv reads the next TTLV binary payload from the inner stream, then deserialize it into the value pointed
// by `msg`. Note that `msg` must be a pointer.
func (s *Stream) Recv(msg any) error {
	read := 0
	buf := make([]byte, 512)
	need := 8
	for {
		if need > cap(buf) {
			buf = slices.Grow(buf, need-cap(buf))
		}
		n, err := s.inner.Read(buf[read:need])
		if err != nil {
			return err
		}
		if n == 0 {
			if read == 0 {
				return io.ErrUnexpectedEOF
			}
			return io.EOF
		}
		read += n
		need = computeNeededBytes(buf[:read])
		if s.max > 0 && need > s.max {
			return Errorf("Message is too big. Max allowed size is %d bytes", s.max)
		}
		if read >= need {
			return UnmarshalTTLV(buf[:need], msg)
		}
	}
}

// Roundtrip simply perform a Send() followed by a Recv(),
// sending `req` then receiving `resp`.
func (s *Stream) Roundtrip(req, resp any) error {
	if err := s.Send(req); err != nil {
		return err
	}
	return s.Recv(resp)
}
