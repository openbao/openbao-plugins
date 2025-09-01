package kmipclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync/atomic"

	"github.com/ovh/kmip-go"
	"github.com/ovh/kmip-go/ttlv"
)

type recvMsg struct {
	msg any
}

func (msg *recvMsg) DecodeTTLV(d *ttlv.Decoder) error {
	switch d.Tag() {
	case kmip.TagRequestMessage:
		msg.msg = new(kmip.RequestMessage)
	case kmip.TagResponseMessage:
		msg.msg = new(kmip.ResponseMessage)
	default:
		return fmt.Errorf("Unexpected tag %q", ttlv.TagString(d.Tag()))
	}
	return d.Any(&msg.msg)
}

type rxMsg struct {
	msg *kmip.ResponseMessage
}

type txMsg struct {
	msg *kmip.RequestMessage
	err chan<- error
}

type conn struct {
	stream ttlv.Stream
	rx     chan rxMsg
	tx     atomic.Value
	ctx    context.Context
	cancel func(error)
	closed atomic.Bool
}

func newConn(netCon net.Conn) *conn {
	ctx, cancel := context.WithCancelCause(context.Background())
	c := &conn{
		stream: ttlv.NewStream(netCon, -1),
		tx:     atomic.Value{},
		rx:     make(chan rxMsg),
		ctx:    ctx,
		cancel: cancel,
		closed: atomic.Bool{},
	}
	c.tx.Store(make(chan txMsg))
	go c.readloop()
	go c.writeloop()
	return c
}

func (c *conn) Close() error {
	if c.closed.Swap(true) {
		// Server is already closed. Nothing to do
		return nil
	}
	// println("Closing connection")
	// TODO: Wait exit of goroutines
	return c.terminate(net.ErrClosed)
}

func (c *conn) terminate(err error) error {
	c.cancel(err) // Cancel the server context
	if tx := c.tx.Swap(chan txMsg(nil)); tx != nil && tx != chan txMsg(nil) {
		close(tx.(chan txMsg))
	}
	return c.stream.Close() // Close the connection
}

func (c *conn) checkAvailable(ctx context.Context) error {
	if c.closed.Load() {
		return net.ErrClosed
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.ctx.Done():
		return context.Cause(c.ctx)
	default:
		return nil
	}
}

func (c *conn) readloop() {
	// defer println("Exittig readloop")
	defer close(c.rx)
	for !c.closed.Load() {
		msg := recvMsg{}
		resp := rxMsg{}
		if err := c.stream.Recv(&msg); err != nil {
			// println("read fail:", resp.err.Error())
			if errors.Is(err, net.ErrClosed) {
				err = io.ErrClosedPipe
			}
			// Close the client
			_ = c.terminate(err)
			return
		}
		m, ok := msg.msg.(*kmip.ResponseMessage)
		if !ok {
			// Ignore server originated requests (for now)
			continue
		}
		resp.msg = m

		select {
		case c.rx <- resp:
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *conn) writeloop() {
	// defer println("Exittig writeloop")
	tx := c.tx.Load().(chan txMsg)
	for !c.closed.Load() {
		select {
		case req, ok := <-tx:
			if !ok {
				return
			}
			if err := c.stream.Send(req.msg); err != nil {
				// println("write fail:", err.Error())
				if errors.Is(err, net.ErrClosed) {
					err = io.ErrClosedPipe
				}
				req.err <- err
				close(req.err)
				// Close the client
				_ = c.terminate(err)
				return
			}
			close(req.err)
		case <-c.ctx.Done():
			// TODO: Drain tx
			return
		}
	}
}

func (c *conn) send(ctx context.Context, msg *kmip.RequestMessage) error {
	if err := c.checkAvailable(ctx); err != nil {
		return err
	}
	tx := c.tx.Load().(chan txMsg)
	errCh := make(chan error)
	select {
	case tx <- txMsg{msg: msg, err: errCh}:
		select {
		case err := <-errCh:
			return err
		case <-c.ctx.Done():
			// No need to close the client as c.ctx is canceled only when client is closed
			return context.Cause(c.ctx)
		case <-ctx.Done():
			// Close client as the request may have been sent already
			_ = c.terminate(io.ErrClosedPipe)
			return ctx.Err()
		}
	case <-c.ctx.Done():
		close(errCh)
		// No need to close the client as c.ctx is canceled only when client is closed
		return context.Cause(c.ctx)
	case <-ctx.Done():
		close(errCh)
		// No need to close the client as the request has not been sent yet
		return ctx.Err()
	}
}

func (c *conn) recv(ctx context.Context) (*kmip.ResponseMessage, error) {
	if err := c.checkAvailable(ctx); err != nil {
		return nil, err
	}
	select {
	case resp, ok := <-c.rx:
		if !ok {
			return nil, io.ErrClosedPipe
		}
		return resp.msg, nil
	case <-c.ctx.Done():
		// No need to close the client as c.ctx is canceled only when client is closed
		return nil, context.Cause(c.ctx)
	case <-ctx.Done():
		// Close the client to cancel the operation on server
		_ = c.terminate(io.ErrClosedPipe)
		return nil, ctx.Err()
	}
}

func (c *conn) roundtrip(ctx context.Context, msg *kmip.RequestMessage) (*kmip.ResponseMessage, error) {
	if err := c.send(ctx, msg); err != nil {
		return nil, err
	}
	return c.recv(ctx)
}
