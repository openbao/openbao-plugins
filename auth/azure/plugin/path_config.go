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
		},
		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.ReadOperation:   b.pathConfigRead,
			logical.CreateOperation: b.pathConfigWrite,
			logical.UpdateOperation: b.pathConfigWrite,
		},

		//HelpSynopsis:    confHelpSyn,
		//HelpDescription: confHelpDesc,
	}
}

type azureConfig struct {
	TenantID string `json:"tenant_id"`
	Resource string `json:"resource"`
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

	entry, err := logical.StorageEntryJSON("config", config)
	if err != nil {
		return nil, err
	}
	if err := req.Storage.Put(entry); err != nil {
		return nil, err
	}

	// Reset OIDC Provider
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
			"tenant_id": config.TenantID,
			"resource":  config.Resource,
		},
	}
	return resp, nil
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
