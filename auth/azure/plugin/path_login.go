package plugin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2017-12-01/compute"
	"github.com/Azure/go-autorest/autorest"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/helper/strutil"
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
			"resource_id": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: `The resource id for the instance logging in`,
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
	signedJwt := data.Get("jwt").(string)
	if signedJwt == "" {
		return logical.ErrorResponse("jwt is required"), nil
	}
	roleName := data.Get("role").(string)
	if roleName == "" {
		return logical.ErrorResponse("role is required"), nil
	}
	resourceID := data.Get("resource_id").(string)

	config, err := b.config(ctx, req.Storage)
	if err != nil {
		return nil, errwrap.Wrapf("unable to retrieve backend configuration: {{err}}", err)
	}
	if config == nil {
		return logical.ErrorResponse("backend not configured"), nil
	}

	role, err := b.role(ctx, req.Storage, roleName)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return logical.ErrorResponse(fmt.Sprintf("invalid role name %q", roleName)), nil
	}

	// Set the client id for 'aud' claim verification
	client, err := b.getClient(config)
	if err != nil {
		return nil, err
	}

	// The OIDC verifier verifies the signature and checks the 'aud' and 'iss'
	// claims and expiration time
	idToken, err := client.Verifier().Verify(ctx, signedJwt)
	if err != nil {
		return nil, err
	}

	claims := new(additionalClaims)
	if err := idToken.Claims(claims); err != nil {
		return nil, err
	}

	// Check additional claims in token
	if err := verifyClaims(claims, role); err != nil {
		return nil, err
	}

	if err := verifyResourceID(ctx, resourceID, client.Authorizer(), claims, role); err != nil {
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

func verifyClaims(claims *additionalClaims, role *azureRole) error {
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
			if !strutil.StrListContains(role.BoundGroupIDs, group) {
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

func verifyResourceID(ctx context.Context, resourceID string, authorizer autorest.Authorizer, claims *additionalClaims, role *azureRole) error {
	// If not checking anythign with the resource id, exit early
	if len(role.BoundResourceGroups) == 0 && len(role.BoundSubscriptionsIDs) == 0 {
		return nil
	}

	if resourceID == "" {
		return fmt.Errorf("resource_id must be provided for given role")
	}

	parsedResourceID, err := parseAzureResourceID(resourceID)
	if err != nil {
		return err
	}

	if strings.ToLower(parsedResourceID.Provider) != "microsoft.compute" {
		return fmt.Errorf("only Microsoft.Compute providers are supported, got %s", parsedResourceID.Provider)
	}

	vmName, ok := parsedResourceID.Path["virtualMachines"]
	if !ok {
		return fmt.Errorf("virtual machine name not provided")
	}

	client := compute.NewVirtualMachinesClient(parsedResourceID.SubscriptionID)
	client.Authorizer = authorizer
	vm, err := client.Get(ctx, parsedResourceID.ResourceGroup, vmName, compute.InstanceView)
	if err != nil {
		return errwrap.Wrapf("unable to retrieve virtual machine metadata: {{err}}", err)
	}

	if *vm.Identity.PrincipalID != claims.ObjectID {
		return fmt.Errorf("token object id does not match virtual machine principal id")
	}

	if !strutil.StrListContains(role.BoundResourceGroups, parsedResourceID.ResourceGroup) {
		return fmt.Errorf("resource group not authoirzed")
	}

	if !strutil.StrListContains(role.BoundSubscriptionsIDs, parsedResourceID.SubscriptionID) {
		return fmt.Errorf("subscription not authoirzed")
	}

	return nil
}

func (b *azureAuthBackend) pathLoginRenew(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	roleName := req.Auth.InternalData["role"].(string)
	if roleName == "" {
		return nil, fmt.Errorf("failed to fetch role_name during renewal")
	}

	// Ensure that the Role still exists.
	role, err := b.role(ctx, req.Storage, roleName)
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

type additionalClaims struct {
	NotBefore jsonTime `json:"nbf"`
	ObjectID  string   `json:"oid"`
	GroupIDs  []string `json:"groups"`
}
