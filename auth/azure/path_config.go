// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package azureauth

import (
	"context"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func pathConfig(b *azureAuthBackend) *framework.Path {
	return &framework.Path{
		Pattern: "config",
		Fields: map[string]*framework.FieldSchema{
			"tenant_id": {
				Type:        framework.TypeString,
				Description: `The tenant id for the Azure Active Directory. This is sometimes referred to as Directory ID in AD. This value can also be provided with the AZURE_TENANT_ID environment variable.`,
				DisplayAttrs: &framework.DisplayAttributes{
					Name: "Tenant ID",
				},
			},
			"resource": {
				Type:        framework.TypeString,
				Description: `The resource URL for the vault application in Azure Active Directory. This value can also be provided with the AZURE_AD_RESOURCE environment variable.`,
			},
			"environment": {
				Type:        framework.TypeString,
				Description: `The Azure environment name. If not provided, AzurePublicCloud is used. This value can also be provided with the AZURE_ENVIRONMENT environment variable.`,
			},
			"client_id": {
				Type:        framework.TypeString,
				Description: `The OAuth2 client id to connection to Azure. This value can also be provided with the AZURE_CLIENT_ID environment variable.`,
				DisplayAttrs: &framework.DisplayAttributes{
					Name: "Client ID",
				},
			},
			"client_secret": {
				Type:        framework.TypeString,
				Description: `The OAuth2 client secret to connection to Azure. This value can also be provided with the AZURE_CLIENT_SECRET environment variable.`,
			},
			"root_password_ttl": {
				Type:        framework.TypeDurationSecond,
				Default:     defaultRootPasswordTTL,
				Description: "The TTL of the root password in Azure. This can be either a number of seconds or a time formatted duration (ex: 24h, 48ds)",
				Required:    false,
			},
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.pathConfigRead,
			},
			logical.CreateOperation: &framework.PathOperation{
				Callback: b.pathConfigWrite,
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathConfigWrite,
			},
			logical.DeleteOperation: &framework.PathOperation{
				Callback: b.pathConfigDelete,
			},
		},
		ExistenceCheck: b.pathConfigExistenceCheck,

		HelpSynopsis:    confHelpSyn,
		HelpDescription: confHelpDesc,
	}
}

type azureConfig struct {
	TenantID                      string        `json:"tenant_id"`
	Resource                      string        `json:"resource"`
	Environment                   string        `json:"environment"`
	ClientID                      string        `json:"client_id"`
	ClientSecret                  string        `json:"client_secret"`
	ClientSecretKeyID             string        `json:"client_secret_key_id"`
	NewClientSecret               string        `json:"new_client_secret"`
	NewClientSecretCreated        time.Time     `json:"new_client_secret_created"`
	NewClientSecretExpirationDate time.Time     `json:"new_client_secret_expiration_date"`
	NewClientSecretKeyID          string        `json:"new_client_secret_key_id"`
	RootPasswordTTL               time.Duration `json:"root_password_ttl"`
	RootPasswordExpirationDate    time.Time     `json:"root_password_expiration_date"`
}

func (b *azureAuthBackend) config(ctx context.Context, s logical.Storage) (*azureConfig, error) {
	entry, err := s.Get(ctx, "config")
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}

	config := new(azureConfig)
	if err := entry.DecodeJSON(config); err != nil {
		return nil, err
	}
	return config, nil
}

func (b *azureAuthBackend) pathConfigExistenceCheck(ctx context.Context, req *logical.Request, _ *framework.FieldData) (bool, error) {
	config, err := b.config(ctx, req.Storage)
	if err != nil {
		return false, err
	}
	return config != nil, nil
}

func (b *azureAuthBackend) pathConfigWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	config, err := b.config(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if config == nil {
		config = new(azureConfig)
	}

	tenantID, ok := data.GetOk("tenant_id")
	if ok {
		config.TenantID = tenantID.(string)
	}

	resource, ok := data.GetOk("resource")
	if ok {
		config.Resource = resource.(string)
	}

	environment, ok := data.GetOk("environment")
	if ok {
		config.Environment = environment.(string)
	}

	clientID, ok := data.GetOk("client_id")
	if ok {
		config.ClientID = clientID.(string)
	}

	clientSecret, ok := data.GetOk("client_secret")
	if ok {
		config.ClientSecret = clientSecret.(string)
	}

	config.RootPasswordTTL = defaultRootPasswordTTL
	rootExpirationRaw, ok := data.GetOk("root_password_ttl")
	if ok {
		config.RootPasswordTTL = time.Second * time.Duration(rootExpirationRaw.(int))
	}

	// Create a settings object to validate all required settings
	// are available
	if _, err := b.getAzureSettings(ctx, config); err != nil {
		return nil, err
	}

	entry, err := logical.StorageEntryJSON("config", config)
	if err != nil {
		return nil, err
	}
	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, err
	}

	// Reset backend
	b.reset()

	return nil, nil
}

func (b *azureAuthBackend) pathConfigRead(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	config, err := b.config(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, nil
	}

	resp := &logical.Response{
		Data: map[string]interface{}{
			"tenant_id":         config.TenantID,
			"resource":          config.Resource,
			"environment":       config.Environment,
			"client_id":         config.ClientID,
			"root_password_ttl": int(config.RootPasswordTTL.Seconds()),
		},
	}

	if !config.RootPasswordExpirationDate.IsZero() {
		resp.Data["root_password_expiration_date"] = config.RootPasswordExpirationDate
	}

	return resp, nil
}

func (b *azureAuthBackend) pathConfigDelete(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	err := req.Storage.Delete(ctx, "config")

	if err == nil {
		b.reset()
	}

	return nil, err
}

func (b *azureAuthBackend) saveConfig(ctx context.Context, config *azureConfig, s logical.Storage) error {
	entry, err := logical.StorageEntryJSON(configStoragePath, config)
	if err != nil {
		return err
	}

	err = s.Put(ctx, entry)
	if err != nil {
		return err
	}

	// reset the backend since the client and provider will have been
	// built using old versions of this data
	b.reset()

	return nil
}

const (
	// The default password expiration duration is 6 months in
	// the Azure UI, so we're setting it to 6 months (in hours)
	// as the default.
	defaultRootPasswordTTL = 4380 * time.Hour
	configStoragePath      = "config"
	confHelpSyn            = `Configures the Azure authentication backend.`
	confHelpDesc           = `
The Azure authentication backend validates the login JWTs using the
configured credentials.  In order to validate machine information, the
OAuth2 client id and secret are used to query the Azure API.  The OAuth2
credentials require Microsoft.Compute/virtualMachines/read permission on
the resource requesting credentials.
`
)
