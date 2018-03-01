package plugin

import (
	"os"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
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
