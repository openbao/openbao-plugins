package azureauth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
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
	computeClientFunc func(vmName string) (armcompute.VirtualMachinesClientGetResponse, error)
}

type mockVMSSClient struct {
	vmssClientFunc func(vmssName string) (armcompute.VirtualMachineScaleSetsClientGetResponse, error)
}

type mockMSIClient struct {
	msiClientFunc func(resourceName string) (armmsi.UserAssignedIdentitiesClientGetResponse, error)
}

type mockResourceClient struct {
	resourceClientFunc func(resourceID string) (armresources.ClientGetByIDResponse, error)
}

func (c *mockComputeClient) Get(_ context.Context, _, vmName string, _ *armcompute.VirtualMachinesClientGetOptions) (armcompute.VirtualMachinesClientGetResponse, error) {
	if c.computeClientFunc != nil {
		return c.computeClientFunc(vmName)
	}
	return armcompute.VirtualMachinesClientGetResponse{}, nil
}

func (c *mockVMSSClient) Get(_ context.Context, _, vmssName string, _ *armcompute.VirtualMachineScaleSetsClientGetOptions) (armcompute.VirtualMachineScaleSetsClientGetResponse, error) {
	if c.vmssClientFunc != nil {
		return c.vmssClientFunc(vmssName)
	}
	return armcompute.VirtualMachineScaleSetsClientGetResponse{}, nil
}

func (c *mockMSIClient) Get(_ context.Context, _, resourceName string, _ *armmsi.UserAssignedIdentitiesClientGetOptions) (armmsi.UserAssignedIdentitiesClientGetResponse, error) {
	if c.msiClientFunc != nil {
		return c.msiClientFunc(resourceName)
	}
	return armmsi.UserAssignedIdentitiesClientGetResponse{}, nil
}

func (c *mockResourceClient) GetByID(_ context.Context, resourceID, _ string, _ *armresources.ClientGetByIDOptions) (armresources.ClientGetByIDResponse, error) {
	if c.resourceClientFunc != nil {
		return c.resourceClientFunc(resourceID)
	}
	return armresources.ClientGetByIDResponse{}, nil
}

type computeClientFunc func(vmName string) (armcompute.VirtualMachinesClientGetResponse, error)

type vmssClientFunc func(vmssName string) (armcompute.VirtualMachineScaleSetsClientGetResponse, error)

type msiClientFunc func(resourceName string) (armmsi.UserAssignedIdentitiesClientGetResponse, error)

type resourceClientFunc func(resourceID string) (armresources.ClientGetByIDResponse, error)

type mockProvider struct {
	computeClientFunc
	vmssClientFunc
	msiClientFunc
	resourceClientFunc
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

func (p *mockProvider) ResourceClient(string) (resourceClient, error) {
	return &mockResourceClient{
		resourceClientFunc: p.resourceClientFunc,
	}, nil
}
