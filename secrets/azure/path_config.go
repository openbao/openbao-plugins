package azuresecrets

import (
	"context"
	"errors"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

const (
	configStoragePath = "config"
)

type azureConfig struct {
	SubscriptionID string        `json:"subscription_id"`
	TenantID       string        `json:"tenant_id"`
	ClientID       string        `json:"client_id"`
	ClientSecret   string        `json:"client_secret"`
	DefaultTTL     time.Duration `json:"ttl"`
	MaxTTL         time.Duration `json:"max_ttl"`
	Environment    string        `json:"environment"`
}

func pathConfig(b *azureSecretBackend) *framework.Path {
	return &framework.Path{
		Pattern: "config",
		Fields: map[string]*framework.FieldSchema{
			"subscription_id": &framework.FieldSchema{
				Type: framework.TypeString,
				Description: `The subscription id for the Azure Active Directory.
				This value can also be provided with the AZURE_SUBSCRIPTION_ID environment variable.`,
			},
			"tenant_id": &framework.FieldSchema{
				Type: framework.TypeString,
				Description: `The tenant id for the Azure Active Directory. This value can also
				be provided with the AZURE_TENANT_ID environment variable.`,
			},
			"environment": &framework.FieldSchema{
				Type: framework.TypeString,
				Description: `The Azure environment name. If not provided, AzurePublicCloud is used.
				This value can also be provided with the AZURE_ENVIRONMENT environment variable.`,
			},
			"client_id": &framework.FieldSchema{
				Type: framework.TypeString,
				Description: `The OAuth2 client id to connect to Azure.
				This value can also be provided with the AZURE_CLIENT_ID environment variable.`,
			},
			"client_secret": &framework.FieldSchema{
				Type: framework.TypeString,
				Description: `The OAuth2 client secret to connect to Azure.
				This value can also be provided with the AZURE_CLIENT_SECRET environment variable.`,
			},
			"ttl": {
				Type:        framework.TypeDurationSecond,
				Description: "Default lease for generated credentials. If not set or set to 0, will use system default.",
			},
			"max_ttl": {
				Type:        framework.TypeDurationSecond,
				Description: "Maximum time a service principal. If not set or set to 0, will use system default.",
			},
		},
		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.ReadOperation:   b.pathConfigRead,
			logical.CreateOperation: b.pathConfigWrite,
			logical.UpdateOperation: b.pathConfigWrite,
		},
		ExistenceCheck:  b.pathConfigExistenceCheck,
		HelpSynopsis:    confHelpSyn,
		HelpDescription: confHelpDesc,
	}
}

func (b *azureSecretBackend) pathConfigWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	var merr *multierror.Error

	config, err := b.getConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}

	if config == nil {
		if req.Operation == logical.UpdateOperation {
			return nil, errors.New("config not found during update operation")
		}
		config = new(azureConfig)
	}

	if subscriptionID, ok := data.GetOk("subscription_id"); ok {
		config.SubscriptionID = subscriptionID.(string)
	}

	if tenantID, ok := data.GetOk("tenant_id"); ok {
		config.TenantID = tenantID.(string)
	}

	if environment, ok := data.GetOk("environment"); ok {
		e := environment.(string)
		if _, err := azure.EnvironmentFromName(e); err != nil {
			merr = multierror.Append(merr, err)
		} else {
			config.Environment = e
		}
	}

	if clientID, ok := data.GetOk("client_id"); ok {
		config.ClientID = clientID.(string)
	}

	if clientSecret, ok := data.GetOk("client_secret"); ok {
		config.ClientSecret = clientSecret.(string)
	}

	if ttlRaw, ok := data.GetOk("ttl"); ok {
		config.DefaultTTL = time.Duration(ttlRaw.(int)) * time.Second
	} else if req.Operation == logical.CreateOperation {
		config.DefaultTTL = time.Duration(data.Get("ttl").(int)) * time.Second
	}

	if maxTTLRaw, ok := data.GetOk("max_ttl"); ok {
		config.MaxTTL = time.Duration(maxTTLRaw.(int)) * time.Second
	} else if req.Operation == logical.CreateOperation {
		config.MaxTTL = time.Duration(data.Get("max_ttl").(int)) * time.Second
	}

	// validate ttl constraints
	if config.DefaultTTL < 0 {
		merr = multierror.Append(merr, errors.New("ttl < 0"))
	}
	if config.MaxTTL < 0 {
		merr = multierror.Append(merr, errors.New("max_ttl < 0"))
	}

	if config.DefaultTTL > b.System().DefaultLeaseTTL() {
		merr = multierror.Append(merr, errors.New("ttl > system defined TTL"))
	}

	if config.MaxTTL > b.System().MaxLeaseTTL() {
		merr = multierror.Append(merr, errors.New("max_ttl > system defined max TTL"))
	}

	if config.DefaultTTL > config.MaxTTL && config.MaxTTL != 0 {
		merr = multierror.Append(merr, errors.New("ttl > max_ttl"))
	}

	if merr.ErrorOrNil() != nil {
		return logical.ErrorResponse(merr.Error()), nil
	}

	err = b.saveConfig(ctx, config, req.Storage)

	if err == nil {
		// since credentials might have changed, reset the backend to
		// force a reload of the Azure provider.
		b.reset()
	}

	return nil, err
}

func (b *azureSecretBackend) pathConfigRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	config, err := b.getConfig(ctx, req.Storage)

	if err != nil {
		return nil, err
	}

	if config == nil {
		return nil, nil
	}

	resp := &logical.Response{
		Data: map[string]interface{}{
			"subscription_id": config.SubscriptionID,
			"tenant_id":       config.TenantID,
			"environment":     config.Environment,
			"client_id":       config.ClientID,
			"ttl":             int64(config.DefaultTTL / time.Second),
			"max_ttl":         int64(config.MaxTTL / time.Second),
		},
	}
	return resp, nil
}

func (b *azureSecretBackend) pathConfigExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	config, err := b.getConfig(ctx, req.Storage)
	if err != nil {
		return false, err
	}
	return config != nil, nil
}

func (b *azureSecretBackend) getConfig(ctx context.Context, s logical.Storage) (*azureConfig, error) {
	b.lock.RLock()
	unlockFunc := b.lock.RUnlock
	defer func() { unlockFunc() }()

	if b.config != nil {
		return b.config, nil
	}

	// Upgrade lock
	b.lock.RUnlock()
	b.lock.Lock()
	unlockFunc = b.lock.Unlock

	if b.config != nil {
		return b.config, nil
	}

	entry, err := s.Get(ctx, configStoragePath)

	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, nil
	}

	cfg := new(azureConfig)
	if err := entry.DecodeJSON(cfg); err != nil {
		return nil, err
	}

	b.config = cfg

	return cfg, nil
}

func (b *azureSecretBackend) saveConfig(ctx context.Context, cfg *azureConfig, s logical.Storage) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	entry, err := logical.StorageEntryJSON(configStoragePath, cfg)

	if err != nil {
		return err
	}

	err = s.Put(ctx, entry)
	if err != nil {
		return err
	}

	b.config = cfg

	return nil
}

const confHelpSyn = `Configure the Azure Secret backend.`
const confHelpDesc = `
The Azure secret backend requires credentials for managing applications and
service principals. This endpoint is used to configure those credentials as
well as default values for the backend in general.
`
