// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package plugin

import (
	"context"
	"errors"

	"github.com/openbao/go-kms-wrapping/plugin/v2/pb"
	"github.com/openbao/go-kms-wrapping/v2"
)

// ErrPluginShutdown is returned when a plugin client fails because the backing
// plugin server/process has shut down. Catching this error can be used to
// respawn plugin processes and retry the call if desired.
var ErrPluginShutdown = errors.New("plugin is shut down")

var (
	_ wrapping.Wrapper       = (*gRPCWrapperClient)(nil)
	_ wrapping.InitFinalizer = (*gRPCWrapperClient)(nil)
)

type gRPCWrapperClient struct {
	id     string
	ctx    context.Context
	client pb.WrapperClient
}

func (c *gRPCWrapperClient) handleError(err error) error {
	if c.ctx.Err() != nil {
		return ErrPluginShutdown
	}
	return err
}

func (c *gRPCWrapperClient) SetConfig(ctx context.Context, options ...wrapping.Option) (*wrapping.WrapperConfig, error) {
	if c.id != "" {
		// While the API of wrapping.Wrapper theoretically allows wrappers to
		// be reconfigured, in practice most wrappers expect to be configured
		// exactly once and would not correctly handle reconfiguration or
		// function at all without being configured first. For the plugin
		// implementation, SetConfig effectively becomes the constructor. This
		// also means that Finalize must AND should only be called if SetConfig
		// succeeds.
		return nil, errors.New("already configured")
	}

	opts, err := wrapping.GetOpts(options...)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.SetConfig(ctx, &pb.SetConfigRequest{Options: opts})
	if err != nil {
		return nil, c.handleError(err)
	}

	c.id = resp.WrapperId
	return resp.WrapperConfig, nil
}

func (c *gRPCWrapperClient) Type(ctx context.Context) (wrapping.WrapperType, error) {
	resp, err := c.client.Type(ctx, &pb.TypeRequest{WrapperId: c.id})
	if err != nil {
		return wrapping.WrapperTypeUnknown, c.handleError(err)
	}
	return wrapping.WrapperType(resp.Type), nil
}

func (c *gRPCWrapperClient) KeyId(ctx context.Context) (string, error) {
	resp, err := c.client.KeyId(ctx, &pb.KeyIdRequest{WrapperId: c.id})
	if err != nil {
		return "", c.handleError(err)
	}
	return resp.KeyId, nil
}

func (c *gRPCWrapperClient) Encrypt(ctx context.Context, pt []byte, options ...wrapping.Option) (*wrapping.BlobInfo, error) {
	opts, err := wrapping.GetOpts(options...)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Encrypt(ctx, &pb.EncryptRequest{
		Plaintext: pt,
		Options:   opts,
		WrapperId: c.id,
	})
	if err != nil {
		return nil, c.handleError(err)
	}
	return resp.Ciphertext, nil
}

func (c *gRPCWrapperClient) Decrypt(ctx context.Context, ct *wrapping.BlobInfo, options ...wrapping.Option) ([]byte, error) {
	opts, err := wrapping.GetOpts(options...)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Decrypt(ctx, &pb.DecryptRequest{
		Ciphertext: ct,
		Options:    opts,
		WrapperId:  c.id,
	})
	if err != nil {
		return nil, c.handleError(err)
	}
	return resp.Plaintext, nil
}

func (c *gRPCWrapperClient) Init(ctx context.Context, options ...wrapping.Option) error {
	opts, err := wrapping.GetOpts(options...)
	if err != nil {
		return err
	}
	_, err = c.client.Init(ctx, &pb.InitRequest{
		Options:   opts,
		WrapperId: c.id,
	})
	return c.handleError(err)
}

func (c *gRPCWrapperClient) Finalize(ctx context.Context, options ...wrapping.Option) error {
	opts, err := wrapping.GetOpts(options...)
	if err != nil {
		return err
	}
	_, err = c.client.Finalize(ctx, &pb.FinalizeRequest{
		Options:   opts,
		WrapperId: c.id,
	})
	return c.handleError(err)
}
