package plugin

import (
	"os"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/hashicorp/errwrap"
)

func NewAuthorizer(azureConfig *azureConfig) (autorest.Authorizer, error) {
	tenantID := os.Getenv("AZURE_TENANT_ID")
	clientID := os.Getenv("AZURE_CLIENT_ID")
	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
	envName := os.Getenv("AZURE_ENVIRONMENT")
	resource := os.Getenv("AZURE_AD_RESOURCE")

	var env azure.Environment
	if envName == "" {
		env = azure.PublicCloud
	} else {
		var err error
		env, err = azure.EnvironmentFromName(envName)
		if err != nil {
			return nil, err
		}
	}

	if resource == "" {
		resource = env.ResourceManagerEndpoint
	}

	// Use environment first
	if clientSecret != "" {
		config := auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)
		config.AADEndpoint = env.ActiveDirectoryEndpoint
		config.Resource = resource
		return config.Authorizer()
	}

	// Stored config next
	if azureConfig.ClientSecret != "" {
		config := auth.NewClientCredentialsConfig(azureConfig.ClientID, azureConfig.ClientSecret, azureConfig.TenantID)
		config.AADEndpoint = env.ActiveDirectoryEndpoint
		config.Resource = resource
		return config.Authorizer()
	}

	// By default use MSI
	config := auth.NewMSIConfig()
	config.Resource = resource

	return config.Authorizer()
}

type azureSettings struct {
	tenantID     string
	clientID     string
	clientSecret string
	environment  azure.Environment
	resource     string
}

func (b *azureAuthBackend) getAzureSettings(config *azureConfig) (*azureSettings, error) {
	settings := new(azureSettings)

	envTenantID := os.Getenv("AZURE_TENANT_ID")
	switch {
	case envTenantID != "":
		settings.tenantID = envTenantID
	case config.TenantID != "":
		settings.tenantID = config.TenantID
	default:
		var err error
		settings.tenantID, err = b.getTentantID()
		if err != nil {
			return nil, errwrap.Wrapf("unable to determine tenant id: {{err}}", err)
		}
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
