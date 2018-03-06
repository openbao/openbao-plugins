package plugin

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	oidc "github.com/coreos/go-oidc"
)

type tokenVerifier interface {
	Verify(ctx context.Context, token string) (*oidc.IDToken, error)
}

type Client interface {
	Verifier() tokenVerifier
	Authorizer() autorest.Authorizer
}

var _ Client = &azureClient{}

type azureClient struct {
	settings     *azureSettings
	oidcProvider *oidc.Provider
	authorizer   autorest.Authorizer
}

func NewAzureClient(config *azureConfig) (*azureClient, error) {
	settings, err := getAzureSettings(config)
	if err != nil {
		return nil, err
	}

	issuer := fmt.Sprintf("%s/%s/", issuerBaseURI, settings.tenantID)
	provider, err := oidc.NewProvider(context.Background(), issuer)
	if err != nil {
		return nil, err
	}

	client := &azureClient{
		settings:     settings,
		oidcProvider: provider,
	}

	switch {
	// Use environment/config first
	case settings.clientSecret != "":
		config := auth.NewClientCredentialsConfig(settings.clientID, settings.clientSecret, settings.tenantID)
		config.AADEndpoint = settings.environment.ActiveDirectoryEndpoint
		config.Resource = settings.resource
		client.authorizer, err = config.Authorizer()
		if err != nil {
			return nil, err
		}
	// By default use MSI
	default:
		config := auth.NewMSIConfig()
		config.Resource = settings.resource
		client.authorizer, err = config.Authorizer()
		if err != nil {
			return nil, err
		}
	}
	return client, err
}

func (c *azureClient) Verifier() tokenVerifier {
	verifierConfig := &oidc.Config{
		ClientID: c.settings.resource,
	}
	return c.oidcProvider.Verifier(verifierConfig)
}

func (c *azureClient) Authorizer() autorest.Authorizer {
	return c.authorizer
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

	return settings, nil
}
