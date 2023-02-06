// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package azureauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	az "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/coreos/go-oidc"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/vault/sdk/helper/useragent"
	"github.com/hashicorp/vault/sdk/logical"
	"golang.org/x/oauth2"
)

type computeClient interface {
	Get(ctx context.Context, resourceGroupName string, vmName string, options *armcompute.VirtualMachinesClientGetOptions) (armcompute.VirtualMachinesClientGetResponse, error)
}

type vmssClient interface {
	Get(ctx context.Context, resourceGroupName string, vmScaleSetName string, options *armcompute.VirtualMachineScaleSetsClientGetOptions) (armcompute.VirtualMachineScaleSetsClientGetResponse, error)
}

type msiClient interface {
	Get(ctx context.Context, resourceGroupName string, resourceName string, options *armmsi.UserAssignedIdentitiesClientGetOptions) (armmsi.UserAssignedIdentitiesClientGetResponse, error)
}

type resourceClient interface {
	GetByID(ctx context.Context, resourceID, apiVersion string, options *armresources.ClientGetByIDOptions) (armresources.ClientGetByIDResponse, error)
}

type providersClient interface {
	Get(ctx context.Context, resourceProviderNamespace string, options *armresources.ProvidersClientGetOptions) (armresources.ProvidersClientGetResponse, error)
}

type tokenVerifier interface {
	Verify(ctx context.Context, token string) (*oidc.IDToken, error)
}

type provider interface {
	Verifier() tokenVerifier
	ComputeClient(subscriptionID string) (computeClient, error)
	VMSSClient(subscriptionID string) (vmssClient, error)
	MSIClient(subscriptionID string) (msiClient, error)
	ResourceClient(subscriptionID string) (resourceClient, error)
	ProvidersClient(subscriptionID string) (providersClient, error)
}

type azureProvider struct {
	oidcVerifier *oidc.IDTokenVerifier
	settings     *azureSettings
	httpClient   *http.Client
}

type oidcDiscoveryInfo struct {
	Issuer  string `json:"issuer"`
	JWKSURL string `json:"jwks_uri"`
}

// transporter implements the azure exported.Transporter interface to send HTTP
// requests. This allows us to set our custom http client and user agent.
type transporter struct {
	pluginEnv *logical.PluginEnvironment
	sender    *http.Client
}

func (tp transporter) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", useragent.PluginString(tp.pluginEnv,
		userAgentPluginName))

	client := tp.sender

	// don't attempt redirects so we aren't acting as an unintended network proxy
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *azureAuthBackend) newAzureProvider(ctx context.Context, config *azureConfig) (*azureProvider, error) {
	httpClient := cleanhttp.DefaultClient()
	settings, err := b.getAzureSettings(ctx, config)
	if err != nil {
		return nil, err
	}

	// In many OIDC providers, the discovery endpoint matches the issuer. For Azure AD, the discovery
	// endpoint is the AD endpoint which does not match the issuer defined in the discovery payload. This
	// makes a request to the discovery URL to determine the issuer and key set information to configure
	// the OIDC verifier
	discoveryURL := fmt.Sprintf("%s%s/.well-known/openid-configuration", settings.CloudConfig.ActiveDirectoryAuthorityHost, settings.TenantID)
	req, err := http.NewRequestWithContext(ctx, "GET", discoveryURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", useragent.PluginString(settings.PluginEnv,
		userAgentPluginName))

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", resp.Status, body)
	}
	var discoveryInfo oidcDiscoveryInfo
	if err := json.Unmarshal(body, &discoveryInfo); err != nil {
		return nil, fmt.Errorf("unable to unmarshal discovery url: %w", err)
	}

	// Create a remote key set from the discovery endpoint
	keySetCtx := context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)
	remoteKeySet := oidc.NewRemoteKeySet(keySetCtx, discoveryInfo.JWKSURL)

	verifierConfig := &oidc.Config{
		ClientID:             settings.Resource,
		SupportedSigningAlgs: []string{oidc.RS256},
	}
	oidcVerifier := oidc.NewVerifier(discoveryInfo.Issuer, remoteKeySet, verifierConfig)

	return &azureProvider{
		settings:     settings,
		oidcVerifier: oidcVerifier,
		httpClient:   httpClient,
	}, nil
}

func (p *azureProvider) Verifier() tokenVerifier {
	return p.oidcVerifier
}

func (p *azureProvider) ComputeClient(subscriptionID string) (computeClient, error) {
	cred, err := p.getTokenCredential()
	if err != nil {
		return nil, err
	}

	clientOptions := p.getClientOptions()
	client, err := armcompute.NewVirtualMachinesClient(subscriptionID, cred, clientOptions)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (p *azureProvider) VMSSClient(subscriptionID string) (vmssClient, error) {
	cred, err := p.getTokenCredential()
	if err != nil {
		return nil, err
	}

	clientOptions := p.getClientOptions()
	client, err := armcompute.NewVirtualMachineScaleSetsClient(subscriptionID, cred, clientOptions)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (p *azureProvider) MSIClient(subscriptionID string) (msiClient, error) {
	cred, err := p.getTokenCredential()
	if err != nil {
		return nil, err
	}

	clientOptions := p.getClientOptions()
	client, err := armmsi.NewUserAssignedIdentitiesClient(subscriptionID, cred, clientOptions)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (p *azureProvider) ProvidersClient(subscriptionID string) (providersClient, error) {
	cred, err := p.getTokenCredential()
	if err != nil {
		return nil, err
	}

	clientOptions := p.getClientOptions()
	client, err := armresources.NewProvidersClient(subscriptionID, cred, clientOptions)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (p *azureProvider) ResourceClient(subscriptionID string) (resourceClient, error) {
	cred, err := p.getTokenCredential()
	if err != nil {
		return nil, err
	}

	clientOptions := p.getClientOptions()
	client, err := armresources.NewClient(subscriptionID, cred, clientOptions)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (p *azureProvider) getClientOptions() *arm.ClientOptions {
	return &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Cloud: p.settings.CloudConfig,
			Transport: transporter{
				pluginEnv: p.settings.PluginEnv,
				sender:    p.httpClient,
			},
		},
	}
}

func (p *azureProvider) getTokenCredential() (azcore.TokenCredential, error) {
	if p.settings.ClientSecret != "" {
		cred, err := az.NewClientSecretCredential(p.settings.TenantID, p.settings.ClientID, p.settings.ClientSecret, nil)
		if err != nil {
			return nil, err
		}

		return cred, nil
	} else {
		cred, err := az.NewManagedIdentityCredential(nil)
		if err != nil {
			return nil, err
		}

		return cred, nil
	}
}

type azureSettings struct {
	TenantID     string
	ClientID     string
	ClientSecret string
	CloudConfig  cloud.Configuration
	Resource     string
	PluginEnv    *logical.PluginEnvironment
}

func (b *azureAuthBackend) getAzureSettings(ctx context.Context, config *azureConfig) (*azureSettings, error) {
	settings := new(azureSettings)

	envTenantID := os.Getenv("AZURE_TENANT_ID")
	switch {
	case envTenantID != "":
		settings.TenantID = envTenantID
	case config.TenantID != "":
		settings.TenantID = config.TenantID
	default:
		return nil, errors.New("tenant_id is required")
	}

	envResource := os.Getenv("AZURE_AD_RESOURCE")
	switch {
	case envResource != "":
		settings.Resource = envResource
	case config.Resource != "":
		settings.Resource = config.Resource
	default:
		return nil, errors.New("resource is required")
	}

	clientID := os.Getenv("AZURE_CLIENT_ID")
	if clientID == "" {
		clientID = config.ClientID
	}
	settings.ClientID = clientID

	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
	if clientSecret == "" {
		clientSecret = config.ClientSecret
	}
	settings.ClientSecret = clientSecret

	configName := os.Getenv("AZURE_ENVIRONMENT")
	if configName == "" {
		configName = config.Environment
	}
	if configName == "" {
		settings.CloudConfig = cloud.AzurePublic
	} else {
		var err error
		settings.CloudConfig, err = ConfigurationFromName(configName)
		if err != nil {
			return nil, err
		}
	}

	pluginEnv, err := b.System().PluginEnv(ctx)
	if err != nil {
		b.Logger().Warn("failed to read plugin environment, user-agent will not be set",
			"error", err)
	}
	settings.PluginEnv = pluginEnv

	return settings, nil
}

func ConfigurationFromName(name string) (cloud.Configuration, error) {
	configs := map[string]cloud.Configuration{
		"AZURECHINACLOUD":        cloud.AzureChina,
		"AZUREPUBLICCLOUD":       cloud.AzurePublic,
		"AZUREUSGOVERNMENTCLOUD": cloud.AzureGovernment,
	}

	name = strings.ToUpper(name)
	c, ok := configs[name]
	if !ok {
		return c, fmt.Errorf("err: no cloud configuration matching the name %q", name)
	}

	return c, nil
}
