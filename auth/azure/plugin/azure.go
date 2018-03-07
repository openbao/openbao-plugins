package plugin

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2017-12-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	oidc "github.com/coreos/go-oidc"
)

const (
	issuerBaseURI = "https://sts.windows.net"
)

type computeClient interface {
	Get(ctx context.Context, resourceGroup, vmName string, instanceView compute.InstanceViewTypes) (compute.VirtualMachine, error)
}

type tokenVerifier interface {
	Verify(ctx context.Context, token string) (*oidc.IDToken, error)
}

type provider interface {
	Verifier() tokenVerifier
	ComputeClient(subscriptionID string) computeClient
}

var _ provider = &azureProvider{}

type azureProvider struct {
	settings     *azureSettings
	oidcProvider *oidc.Provider
	authorizer   autorest.Authorizer
}

func NewAzureProvider(config *azureConfig) (*azureProvider, error) {
	settings, err := getAzureSettings(config)
	if err != nil {
		return nil, err
	}

	issuer := fmt.Sprintf("%s/%s/", issuerBaseURI, settings.tenantID)
	oidcProvider, err := oidc.NewProvider(context.Background(), issuer)
	if err != nil {
		return nil, err
	}

	provider := &azureProvider{
		settings:     settings,
		oidcProvider: oidcProvider,
	}

	// OAuth2 client for querying VM data
	switch {
	// Use environment/config first
	case settings.clientSecret != "":
		config := auth.NewClientCredentialsConfig(settings.clientID, settings.clientSecret, settings.tenantID)
		config.AADEndpoint = settings.environment.ActiveDirectoryEndpoint
		config.Resource = settings.environment.ResourceManagerEndpoint
		provider.authorizer, err = config.Authorizer()
		if err != nil {
			return nil, err
		}
	// By default use MSI
	default:
		config := auth.NewMSIConfig()
		config.Resource = settings.environment.ResourceManagerEndpoint
		provider.authorizer, err = config.Authorizer()
		if err != nil {
			return nil, err
		}
	}
	return provider, err
}

func (p *azureProvider) Verifier() tokenVerifier {
	verifierConfig := &oidc.Config{
		ClientID: p.settings.resource,
	}
	return p.oidcProvider.Verifier(verifierConfig)
}

func (p *azureProvider) ComputeClient(subscriptionID string) computeClient {
	client := compute.NewVirtualMachinesClient(subscriptionID)
	client.Authorizer = p.authorizer
	return client
}

type azureSettings struct {
	tenantID     string
	clientID     string
	clientSecret string
	environment  azure.Environment
	resource     string
}

func getAzureSettings(config *azureConfig) (*azureSettings, error) {
	settings := new(azureSettings)

	envTenantID := os.Getenv("AZURE_TENANT_ID")
	switch {
	case envTenantID != "":
		settings.tenantID = envTenantID
	case config.TenantID != "":
		settings.tenantID = config.TenantID
	default:
		return nil, errors.New("tenant id is required")
	}

	clientID := os.Getenv("AZURE_CLIENT_ID")
	if clientID == "" {
		clientID = config.ClientID
	}
	settings.clientID = clientID

	clientSecret := os.Getenv("AZURE_CLIENT_ID")
	if clientSecret == "" {
		clientSecret = config.ClientSecret
	}
	settings.clientSecret = clientSecret

	envName := os.Getenv("AZURE_ENVIRONMENT")
	if envName == "" {
		envName = config.Environment
	}
	if envName == "" {
		settings.environment = azure.PublicCloud
	} else {
		var err error
		settings.environment, err = azure.EnvironmentFromName(envName)
		if err != nil {
			return nil, err
		}
	}

	resource := os.Getenv("AZURE_AD_RESOURCE")
	if resource == "" && config.Resource != "" {
		resource = config.Resource
	} else {
		resource = settings.environment.ResourceManagerEndpoint
	}
	settings.resource = resource

	return settings, nil
}
