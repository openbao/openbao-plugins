package plugin

import (
	"context"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

func pathConfig(b *azureAuthBackend) *framework.Path {
	return &framework.Path{
		Pattern: "config",
		Fields: map[string]*framework.FieldSchema{
			"tenant_id": &framework.FieldSchema{
				Type: framework.TypeString,
				//Description: "",
			},
			"resource": &framework.FieldSchema{
				Type: framework.TypeString,
				//Description: "",
			},
			"client_id": &framework.FieldSchema{
				Type: framework.TypeString,
				//Description: "",
			},
			"client_secret": &framework.FieldSchema{
				Type: framework.TypeString,
				//Description: "",
			},
		},
		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.ReadOperation:   b.pathConfigRead,
			logical.CreateOperation: b.pathConfigWrite,
			logical.UpdateOperation: b.pathConfigWrite,
		},
		ExistenceCheck: b.pathConfigExistenceCheck,

		//HelpSynopsis:    confHelpSyn,
		//HelpDescription: confHelpDesc,
	}
}

type azureConfig struct {
	TenantID     string `json:"tenant_id"`
	Resource     string `json:"resource"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func (b *azureAuthBackend) config(s logical.Storage) (*azureConfig, error) {
	entry, err := s.Get("config")
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

func (b *azureAuthBackend) pathConfigExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	config, err := b.config(req.Storage)
	if err != nil {
		return false, err
	}
	return config != nil, nil
}

func (b *azureAuthBackend) pathConfigWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	config, err := b.config(req.Storage)
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
	} else if req.Operation == logical.CreateOperation {
		return logical.ErrorResponse("resource is required"), logical.ErrInvalidRequest
	}

	clientID, ok := data.GetOk("client_id")
	if ok {
		config.ClientID = clientID.(string)
	}

	clientSecret, ok := data.GetOk("client_secret")
	if ok {
		config.ClientSecret = clientSecret.(string)
	}

	entry, err := logical.StorageEntryJSON("config", config)
	if err != nil {
		return nil, err
	}
	if err := req.Storage.Put(entry); err != nil {
		return nil, err
	}

	// Reset backend
	b.reset()

	return nil, nil
}

func (b *azureAuthBackend) pathConfigRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	config, err := b.config(req.Storage)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, nil
	}

	resp := &logical.Response{
		Data: map[string]interface{}{
			"tenant_id":     config.TenantID,
			"resource":      config.Resource,
			"client_id":     config.ClientID,
			"client_secret": config.ClientSecret,
		},
	}
	return resp, nil
}
