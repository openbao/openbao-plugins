package plugin

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/vault/helper/strutil"

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
				Description: `The token role.`,
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

	// Set the client id for 'aud' claim verification
	verifier, err := b.getOIDCVerifier(config)
	if err != nil {
		return nil, err
	}

	// The OIDC verifier verifies the signature and checks the 'aud' and 'iss'
	// claims and expiration time
	idToken, err := verifier.Verify(ctx, signedJwt.(string))
	if err != nil {
		return nil, err
	}

	// Check additional claims in token
	if err := verifyClaims(idToken, role); err != nil {
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

func verifyClaims(idToken *oidc.IDToken, role *azureRole) error {
	var claims struct {
		NotBefore jsonTime `json:"nbf"`
		ObjectID  string   `json:"oid"`
		GroupIDs  []string `json:"groups"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return err
	}

	notBefore := time.Time(claims.NotBefore)
	if notBefore.After(time.Now()) {
		return fmt.Errorf("token is not yet valid (Token Not Before: %v)", notBefore)
	}

	if len(role.BoundServicePrincipalIDs) > 0 {
		if !strutil.StrListContains(role.BoundServicePrincipalIDs, claims.ObjectID) {
			return fmt.Errorf("service principal not authorized: %s", claims.ObjectID)
		}
	}

	if len(role.BoundServicePrincipalIDs) > 0 {
		var found bool
		for _, group := range claims.GroupIDs {
			if !strutil.StrListContains(role.BoundServicePrincipalIDs, group) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("group not authorized: %v", claims.GroupIDs)
		}
	}

	return nil
}

func (b *azureAuthBackend) pathLoginRenew(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	roleName := req.Auth.InternalData["role"].(string)
	if roleName == "" {
		return nil, fmt.Errorf("failed to fetch role_name during renewal")
	}

	// Ensure that the Role still exists.
	role, err := b.role(req.Storage, roleName)
	if err != nil {
		return nil, fmt.Errorf("failed to validate role %s during renewal:%s", roleName, err)
	}
	if role == nil {
		return nil, fmt.Errorf("role %s does not exist during renewal", roleName)
	}

	// If 'Period' is set on the Role, the token should never expire.
	// Replenish the TTL with 'Period's value.
	if role.Period > time.Duration(0) {
		// If 'Period' was updated after the token was issued,
		// token will bear the updated 'Period' value as its TTL.
		req.Auth.TTL = role.Period
		return &logical.Response{Auth: req.Auth}, nil
	}

	return framework.LeaseExtend(role.TTL, role.MaxTTL, b.System())(ctx, req, data)
}
