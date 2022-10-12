package azureauth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-11-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"
	"github.com/coreos/go-oidc"
)

// mockKeySet is used in tests to bypass signature validation and return only
// the jwt payload
type mockKeySet struct{}

func (s *mockKeySet) VerifySignature(_ context.Context, idToken string) ([]byte, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid jwt")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("error decoding payload: %v", err)
	}
	return payload, nil
}

func newMockVerifier() tokenVerifier {
	config := &oidc.Config{
		SkipClientIDCheck: true,
		SkipExpiryCheck:   false,
	}
	ks := new(mockKeySet)
	return oidc.NewVerifier("", ks, config)
}

type mockComputeClient struct {
	computeClientFunc func(vmName string) (compute.VirtualMachine, error)
}

type mockVMSSClient struct {
	vmssClientFunc func(vmssName string) (compute.VirtualMachineScaleSet, error)
}

type mockMSIClient struct {
	msiClientFunc func(resourceName string) (msi.Identity, error)
}

func (c *mockComputeClient) Get(_ context.Context, _, vmName string, _ compute.InstanceViewTypes) (compute.VirtualMachine, error) {
	if c.computeClientFunc != nil {
		return c.computeClientFunc(vmName)
	}
	return compute.VirtualMachine{}, nil
}

func (c *mockVMSSClient) Get(_ context.Context, _, vmssName string, _ compute.ExpandTypesForGetVMScaleSets) (compute.VirtualMachineScaleSet, error) {
	if c.vmssClientFunc != nil {
		return c.vmssClientFunc(vmssName)
	}
	return compute.VirtualMachineScaleSet{}, nil
}

func (c *mockMSIClient) Get(_ context.Context, _, resourceName string) (msi.Identity, error) {
	if c.msiClientFunc != nil {
		return c.msiClientFunc(resourceName)
	}
	return msi.Identity{}, nil
}

type computeClientFunc func(vmName string) (compute.VirtualMachine, error)

type vmssClientFunc func(vmssName string) (compute.VirtualMachineScaleSet, error)

type msiClientFunc func(resourceName string) (msi.Identity, error)

type mockProvider struct {
	computeClientFunc
	vmssClientFunc
	msiClientFunc
}

func newMockProvider(c computeClientFunc, v vmssClientFunc, m msiClientFunc) *mockProvider {
	return &mockProvider{
		computeClientFunc: c,
		vmssClientFunc:    v,
		msiClientFunc:     m,
	}
}

func (*mockProvider) Verifier() tokenVerifier {
	return newMockVerifier()
}

func (p *mockProvider) ComputeClient(string) (computeClient, error) {
	return &mockComputeClient{
		computeClientFunc: p.computeClientFunc,
	}, nil
}

func (p *mockProvider) VMSSClient(string) (vmssClient, error) {
	return &mockVMSSClient{
		vmssClientFunc: p.vmssClientFunc,
	}, nil
}

func (p *mockProvider) MSIClient(string) (msiClient, error) {
	return &mockMSIClient{
		msiClientFunc: p.msiClientFunc,
	}, nil
}
