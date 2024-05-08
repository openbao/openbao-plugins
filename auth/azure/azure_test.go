// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package azureauth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/coreos/go-oidc"

	"github.com/hashicorp/vault-plugin-auth-azure/client"
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

func newMockVerifier() client.TokenVerifier {
	config := &oidc.Config{
		SkipClientIDCheck: true,
		SkipExpiryCheck:   false,
	}
	ks := new(mockKeySet)
	return oidc.NewVerifier("", ks, config)
}

type mockComputeClient struct {
	computeClientFunc computeClientFunc
}

type mockVMSSClient struct {
	vmssClientFunc vmssClientFunc
}

type mockMSIClient struct {
	msiClientFunc msiClientFunc
	msiListFunc   msiListFunc
}

type mockResourceClient struct {
	resourceClientFunc resourceClientFunc
}

type mockProvidersClient struct {
	providersClientFunc providersClientFunc
}

func (c *mockComputeClient) Get(ctx context.Context, _, vmName string, _ *armcompute.VirtualMachinesClientGetOptions) (armcompute.VirtualMachinesClientGetResponse, error) {
	if c.computeClientFunc != nil {
		return c.computeClientFunc(vmName)
	}
	return armcompute.VirtualMachinesClientGetResponse{}, nil
}

func (c *mockVMSSClient) Get(ctx context.Context, _, vmssName string, _ *armcompute.VirtualMachineScaleSetsClientGetOptions) (armcompute.VirtualMachineScaleSetsClientGetResponse, error) {
	if c.vmssClientFunc != nil {
		return c.vmssClientFunc(vmssName)
	}
	return armcompute.VirtualMachineScaleSetsClientGetResponse{}, nil
}

func (c *mockMSIClient) Get(ctx context.Context, _, resourceName string, _ *armmsi.UserAssignedIdentitiesClientGetOptions) (armmsi.UserAssignedIdentitiesClientGetResponse, error) {
	if c.msiClientFunc != nil {
		return c.msiClientFunc(resourceName)
	}
	return armmsi.UserAssignedIdentitiesClientGetResponse{}, nil
}

func (c *mockMSIClient) NewListByResourceGroupPager(resourceGroup string, _ *armmsi.UserAssignedIdentitiesClientListByResourceGroupOptions) *runtime.Pager[armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse] {
	if c.msiListFunc != nil {
		resp := c.msiListFunc(resourceGroup)
		// the listfunc returns the response, here we wrap it in a pager, so that the mock-er only has to worry about the response we want.
		return runtime.NewPager[armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse](runtime.PagingHandler[armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse]{
			// since we only have one response, there are no more responses.
			More: func(response armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse) bool { return false },
			Fetcher: func(ctx context.Context, data *armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse) (armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse, error) {
				return resp, nil
			},
		})
	}
	return nil
}

func (c *mockResourceClient) GetByID(ctx context.Context, resourceID, _ string, _ *armresources.ClientGetByIDOptions) (armresources.ClientGetByIDResponse, error) {
	if c.resourceClientFunc != nil {
		return c.resourceClientFunc(resourceID)
	}
	return armresources.ClientGetByIDResponse{}, nil
}

func (c *mockProvidersClient) Get(ctx context.Context, resourceID string, _ *armresources.ProvidersClientGetOptions) (armresources.ProvidersClientGetResponse, error) {
	if c.providersClientFunc != nil {
		return c.providersClientFunc(resourceID)
	}
	return armresources.ProvidersClientGetResponse{}, nil
}

type computeClientFunc func(vmName string) (armcompute.VirtualMachinesClientGetResponse, error)

type vmssClientFunc func(vmssName string) (armcompute.VirtualMachineScaleSetsClientGetResponse, error)

type msiClientFunc func(resourceName string) (armmsi.UserAssignedIdentitiesClientGetResponse, error)

type msiListFunc func(resoucename string) armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse

type msGraphClientFunc func() (client.MSGraphClient, error)

type resourceClientFunc func(resourceID string) (armresources.ClientGetByIDResponse, error)

type providersClientFunc func(s string) (armresources.ProvidersClientGetResponse, error)

type mockProvider struct {
	computeClientFunc
	vmssClientFunc
	msiClientFunc
	msiListFunc
	msGraphClientFunc
	resourceClientFunc
	providersClientFunc
}

func newMockProvider(c computeClientFunc, v vmssClientFunc, m msiClientFunc, ml msiListFunc, g msGraphClientFunc) *mockProvider {
	return &mockProvider{
		computeClientFunc: c,
		vmssClientFunc:    v,
		msiClientFunc:     m,
		msiListFunc:       ml,
		msGraphClientFunc: g,
	}
}

func (*mockProvider) TokenVerifier() client.TokenVerifier {
	return newMockVerifier()
}

func (p *mockProvider) ComputeClient(subscriptionID string) (client.ComputeClient, error) {
	return &mockComputeClient{
		computeClientFunc: p.computeClientFunc,
	}, nil
}

func (p *mockProvider) VMSSClient(subscriptionID string) (client.VMSSClient, error) {
	return &mockVMSSClient{
		vmssClientFunc: p.vmssClientFunc,
	}, nil
}

func (p *mockProvider) MSIClient(subscriptionID string) (client.MSIClient, error) {
	return &mockMSIClient{
		msiClientFunc: p.msiClientFunc,
		msiListFunc:   p.msiListFunc,
	}, nil
}

func (p *mockProvider) MSGraphClient() (client.MSGraphClient, error) {
	return nil, nil
}

func (p *mockProvider) ResourceClient(subscriptionID string) (client.ResourceClient, error) {
	return &mockResourceClient{
		resourceClientFunc: p.resourceClientFunc,
	}, nil
}

func (p *mockProvider) ProvidersClient(subscriptionID string) (client.ProvidersClient, error) {
	return &mockProvidersClient{
		providersClientFunc: p.providersClientFunc,
	}, nil
}
