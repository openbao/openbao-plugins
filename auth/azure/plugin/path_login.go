package plugin

import (
	"context"
	"fmt"
	"time"

	oidc "github.com/coreos/go-oidc"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

func pathLogin(b *azureAuthBackend) *framework.Path {
	return &framework.Path{
		Pattern: "login$",
		Fields: map[string]*framework.FieldSchema{
			"role": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: `The token role to login in use.`,
			},
			"jwt": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: `A signed JWT`,
			},
		},

		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.UpdateOperation:         b.pathLogin,
			logical.AliasLookaheadOperation: b.pathLogin,
		},

		//HelpSynopsis:    pathLoginHelpSyn,
		//HelpDescription: pathLoginHelpDesc,
	}
}

func (b *azureAuthBackend) pathLogin(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	signedJwt := data.Get("jwt")
	if signedJwt == "" {
		return logical.ErrorResponse("jwt is required"), nil
	}
	roleName := data.Get("role").(string)
	if roleName == "" {
		return logical.ErrorResponse("role is required"), nil
	}

	config, err := b.config(req.Storage)
	if err != nil {
		return nil, errwrap.Wrapf("unable to retrieve backend configuration: {{err}}", err)
	}
	if config == nil {
		return logical.ErrorResponse("backend not configured"), nil
	}

	role, err := b.role(req.Storage, roleName)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return logical.ErrorResponse(fmt.Sprintf("invalid role name %q", roleName)), nil
	}

	// Set the client id for aud claim verification
	verifierConfig := &oidc.Config{
		ClientID: config.Resource,
	}
	verifier, err := b.getOIDCVerifier(verifierConfig, config)
	if err != nil {
		return nil, err
	}

	// The OIDC verifier verifies the signature and checks the 'aud' and 'iss'
	//claims and expiration time
	idToken, err := verifier.Verify(ctx, signedJwt.(string))
	if err != nil {
		return nil, err
	}

	if err := verifyClaims(verifierConfig, idToken); err != nil {
		return nil, err
	}

	resp := &logical.Response{
		Auth: &logical.Auth{
			Policies:    role.Policies,
			DisplayName: idToken.Subject,
			Period:      role.Period,
			NumUses:     role.NumUses,
			Alias: &logical.Alias{
				Name: idToken.Subject,
			},
			InternalData: map[string]interface{}{
				"role": roleName,
			},
			Metadata: map[string]string{
				"role": roleName,
			},
			LeaseOptions: logical.LeaseOptions{
				Renewable: true,
				TTL:       role.TTL,
			},
		},
	}
	return resp, nil
}

func verifyClaims(config *oidc.Config, idToken *oidc.IDToken) error {
	var claims struct {
		NotBefore jsonTime `json:"nbf"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return err
	}

	now := time.Now
	if config.Now != nil {
		now = config.Now
	}

	notBefore := time.Time(claims.NotBefore)
	if notBefore.After(now()) {
		return fmt.Errorf("token is not yet valid (Token Not Before: %v)", notBefore)
	}
	return nil
}
