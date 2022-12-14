package azureauth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/cidrutil"
	"github.com/hashicorp/vault/sdk/logical"
)

// resourceClientAPIVersion is the API version to use for the operation. This
// is not well documented but we are setting it to the current value defined
// here: https://learn.microsoft.com/en-us/rest/api/resources/resources/get-by-id#uri-parameters
var resourceClientAPIVersion = "2021-04-01"

func pathLogin(b *azureAuthBackend) *framework.Path {
	return &framework.Path{
		Pattern: "login$",
		Fields: map[string]*framework.FieldSchema{
			"role": {
				Type:        framework.TypeString,
				Description: `The token role.`,
			},
			"jwt": {
				Type:        framework.TypeString,
				Description: `A signed JWT`,
			},
			"subscription_id": {
				Type:        framework.TypeString,
				Description: `The subscription id for the instance.`,
			},
			"resource_group_name": {
				Type:        framework.TypeString,
				Description: `The resource group from the instance.`,
			},
			"vm_name": {
				Type:        framework.TypeString,
				Description: `The name of the virtual machine. This value is ignored if vmss_name is specified.`,
			},
			"vmss_name": {
				Type:        framework.TypeString,
				Description: `The name of the virtual machine scale set the instance is in.`,
			},
			"resource_id": {
				Type: framework.TypeString,
				Description: `The fully qualified ID of the resource, including` +
					`the resource name and resource type. Use the format, ` +
					`/subscriptions/{guid}/resourceGroups/{resource-group-name}/{resource-provider-namespace}/{resource-type}/{resource-name}. ` +
					`This value is ignored if vm_name or vmss_name is specified.`,
			},
		},

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathLogin,
			},
			logical.AliasLookaheadOperation: &framework.PathOperation{
				Callback: b.pathLogin,
			},
			logical.ResolveRoleOperation: &framework.PathOperation{
				Callback: b.pathResolveRole,
			},
		},

		HelpSynopsis:    pathLoginHelpSyn,
		HelpDescription: pathLoginHelpDesc,
	}
}

func (b *azureAuthBackend) pathResolveRole(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	roleName := data.Get("role").(string)
	if roleName == "" {
		return logical.ErrorResponse("role is required"), nil
	}

	role, err := b.role(ctx, req.Storage, roleName)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return logical.ErrorResponse(fmt.Sprintf("invalid role name %q", roleName)), nil
	}

	return logical.ResolveRoleResponse(roleName)
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

	role, err := b.role(ctx, req.Storage, roleName)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return logical.ErrorResponse(fmt.Sprintf("invalid role name %q", roleName)), nil
	}

	if len(role.TokenBoundCIDRs) > 0 {
		if req.Connection == nil {
			b.Logger().Warn("token bound CIDRs found but no connection information available for validation")
			return nil, logical.ErrPermissionDenied
		}
		if !cidrutil.RemoteAddrIsOk(req.Connection.RemoteAddr, role.TokenBoundCIDRs) {
			return nil, logical.ErrPermissionDenied
		}
	}

	subscriptionID := data.Get("subscription_id").(string)
	resourceGroupName := data.Get("resource_group_name").(string)
	vmssName := data.Get("vmss_name").(string)
	vmName := data.Get("vm_name").(string)
	resourceID := data.Get("resource_id").(string)

	config, err := b.config(ctx, req.Storage)
	if err != nil {
		return nil, errwrap.Wrapf("unable to retrieve backend configuration: {{err}}", err)
	}
	if config == nil {
		config = new(azureConfig)
	}

	provider, err := b.getProvider(ctx, config)
	if err != nil {
		return nil, err
	}

	// The OIDC verifier verifies the signature and checks the 'aud' and 'iss'
	// claims and expiration time
	idToken, err := provider.Verifier().Verify(ctx, signedJwt)
	if err != nil {
		return nil, err
	}

	claims := new(additionalClaims)
	if err := idToken.Claims(claims); err != nil {
		return nil, err
	}

	// Check additional claims in token
	if err := b.verifyClaims(claims, role); err != nil {
		return nil, err
	}

	if err := b.verifyResource(ctx, subscriptionID, resourceGroupName, vmName, vmssName, resourceID, claims, role); err != nil {
		return nil, err
	}

	auth := &logical.Auth{
		DisplayName: claims.ObjectID,
		Alias: &logical.Alias{
			Name: claims.ObjectID,
			Metadata: map[string]string{
				"resource_group_name": resourceGroupName,
				"subscription_id":     subscriptionID,
			},
		},
		InternalData: map[string]interface{}{
			"role": roleName,
		},
		Metadata: map[string]string{
			"role":                roleName,
			"resource_group_name": resourceGroupName,
			"subscription_id":     subscriptionID,
		},
	}

	if vmName != "" {
		auth.Alias.Metadata["vm_name"] = vmName
		auth.Metadata["vm_name"] = vmName
	}
	if vmssName != "" {
		auth.Alias.Metadata["vmss_name"] = vmssName
		auth.Metadata["vmss_name"] = vmssName
	}
	if resourceID != "" {
		auth.Alias.Metadata["resource_id"] = resourceID
		auth.Metadata["resource_id"] = resourceID
	}

	role.PopulateTokenAuth(auth)

	resp := &logical.Response{
		Auth: auth,
	}

	// Add groups to group aliases
	for _, groupID := range claims.GroupIDs {
		if groupID == "" {
			continue
		}
		resp.Auth.GroupAliases = append(resp.Auth.GroupAliases, &logical.Alias{
			Name: groupID,
		})
	}

	return resp, nil
}

func (b *azureAuthBackend) verifyClaims(claims *additionalClaims, role *azureRole) error {
	notBefore := time.Time(claims.NotBefore)
	if notBefore.After(time.Now()) {
		return fmt.Errorf("token is not yet valid (Token Not Before: %v)", notBefore)
	}

	if (len(role.BoundServicePrincipalIDs) == 1 && role.BoundServicePrincipalIDs[0] == "*") &&
		(len(role.BoundGroupIDs) == 1 && role.BoundGroupIDs[0] == "*") {
		return fmt.Errorf("expected specific bound_group_ids or bound_service_principal_ids; both cannot be '*'")
	}
	switch {
	case len(role.BoundServicePrincipalIDs) == 1 && role.BoundServicePrincipalIDs[0] == "*":
		// Globbing on PrincipalIDs; can skip Service Principal ID check
	case len(role.BoundServicePrincipalIDs) > 0:
		if !strListContains(role.BoundServicePrincipalIDs, claims.ObjectID) {
			return fmt.Errorf("service principal not authorized: %s", claims.ObjectID)
		}
	}

	switch {
	case len(role.BoundGroupIDs) == 1 && role.BoundGroupIDs[0] == "*":
		// Globbing on GroupIDs; can skip group ID check
	case len(role.BoundGroupIDs) > 0:
		var found bool
		for _, group := range claims.GroupIDs {
			if strListContains(role.BoundGroupIDs, group) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("groups not authorized: %v", claims.GroupIDs)
		}
	}

	return nil
}

func (b *azureAuthBackend) verifyResource(ctx context.Context, subscriptionID, resourceGroupName, vmName, vmssName, resourceID string, claims *additionalClaims, role *azureRole) error {
	// If not checking anything with the resource id, exit early
	if len(role.BoundResourceGroups) == 0 && len(role.BoundSubscriptionsIDs) == 0 && len(role.BoundLocations) == 0 && len(role.BoundScaleSets) == 0 {
		return nil
	}

	if subscriptionID == "" || resourceGroupName == "" {
		return errors.New("subscription_id and resource_group_name are required")
	}

	var location *string
	principalIDs := map[string]struct{}{}

	switch {
	// If vmss name is specified, the vm name will be ignored and only the scale set
	// will be verified since vm names are generated automatically for scale sets
	case vmssName != "":
		client, err := b.provider.VMSSClient(subscriptionID)
		if err != nil {
			return errwrap.Wrapf("unable to create vmss client: {{err}}", err)
		}

		// Omit armcompute.ExpandTypesForGetVMScaleSetsUserData since we do not need that information for purpose of authenticating an instance
		vmss, err := client.Get(ctx, resourceGroupName, vmssName, nil)
		if err != nil {
			return errwrap.Wrapf("unable to retrieve virtual machine scale set metadata: {{err}}", err)
		}

		// Check bound scale sets
		if len(role.BoundScaleSets) > 0 && !strListContains(role.BoundScaleSets, vmssName) {
			return errors.New("scale set not authorized")
		}

		location = vmss.Location

		if vmss.Identity == nil {
			return errors.New("vmss client did not return identity information")
		}
		// if system-assigned identity's principal id is available
		if vmss.Identity.PrincipalID != nil {
			principalIDs[convertPtrToString(vmss.Identity.PrincipalID)] = struct{}{}
		}
		// if not, look for user-assigned identities
		for userIdentityID, userIdentity := range vmss.Identity.UserAssignedIdentities {
			// Principal ID is not nil for VMSS uniform orchestration mode
			if userIdentity.PrincipalID != nil {
				principalIDs[convertPtrToString(userIdentity.PrincipalID)] = struct{}{}
				continue
			}

			elements := strings.Split(userIdentityID, "/")
			if len(elements) < 9 {
				return fmt.Errorf("unable to parse the user-assigned identity resource ID: %s", userIdentityID)
			}
			msiSubscriptionID := elements[2]
			msiResourceGroupName := elements[4]
			msiResourceName := elements[8]

			// Principal ID is nil for VMSS flex orchestration mode, so we
			// must look up the user-assigned identity using the MSI client
			msiClient, err := b.provider.MSIClient(msiSubscriptionID)
			if err != nil {
				return fmt.Errorf("unable to create msi client: %w", err)
			}
			userIdentityResponse, err := msiClient.Get(ctx, msiResourceGroupName, msiResourceName, nil)
			if err != nil {
				return fmt.Errorf("unable to retrieve user assigned identity metadata: %w", err)
			}

			if userIdentityResponse.Properties != nil && userIdentityResponse.Properties.PrincipalID != nil {
				principalIDs[*userIdentityResponse.Properties.PrincipalID] = struct{}{}
			}
		}
	case vmName != "":
		client, err := b.provider.ComputeClient(subscriptionID)
		if err != nil {
			return errwrap.Wrapf("unable to create compute client: {{err}}", err)
		}

		instanceView := armcompute.InstanceViewTypesInstanceView
		options := armcompute.VirtualMachinesClientGetOptions{
			Expand: &instanceView,
		}

		vm, err := client.Get(ctx, resourceGroupName, vmName, &options)
		if err != nil {
			return errwrap.Wrapf("unable to retrieve virtual machine metadata: {{err}}", err)
		}

		location = vm.Location

		if vm.Identity == nil {
			return errors.New("vm client did not return identity information")
		}
		// Check bound scale sets
		if len(role.BoundScaleSets) > 0 {
			return errors.New("bound scale set defined but this vm isn't in a scale set")
		}
		// if system-assigned identity's principal id is available
		if vm.Identity.PrincipalID != nil {
			principalIDs[convertPtrToString(vm.Identity.PrincipalID)] = struct{}{}
		}
		// if not, look for user-assigned identities
		for _, userIdentity := range vm.Identity.UserAssignedIdentities {
			principalIDs[convertPtrToString(userIdentity.PrincipalID)] = struct{}{}
		}
	default:
		// this is the generic case that should enable Azure services that
		// support managed identities to authenticate to Vault
		if resourceID == "" {
			return errors.New("resource_id is required")
		}
		if len(role.BoundScaleSets) > 0 {
			return errors.New("scale set requires the vmss_name field to be set")
		}

		client, err := b.provider.ResourceClient(subscriptionID)
		if err != nil {
			return fmt.Errorf("unable to create resource client: %w", err)
		}
		resp, err := client.GetByID(ctx, resourceID, resourceClientAPIVersion, nil)
		if err != nil {
			return fmt.Errorf("unable to retrieve user assigned identity metadata: %w", err)
		}
		if resp.Identity == nil {
			return errors.New("client did not return identity information")
		}
		// if system-assigned identity's principal id is available
		if resp.Identity.PrincipalID != nil {
			principalIDs[convertPtrToString(resp.Identity.PrincipalID)] = struct{}{}
		}
		// if not, look for user-assigned identities
		for _, userIdentity := range resp.Identity.UserAssignedIdentities {
			principalIDs[convertPtrToString(userIdentity.PrincipalID)] = struct{}{}
		}
	}

	// Ensure the token OID is the principal id of the system-assigned identity
	// or one of the user-assigned identities
	if _, ok := principalIDs[claims.ObjectID]; !ok {
		return errors.New("token object id does not match expected identities")
	}

	// Check bound subscriptions
	if len(role.BoundSubscriptionsIDs) > 0 && !strListContains(role.BoundSubscriptionsIDs, subscriptionID) {
		return errors.New("subscription not authorized")
	}

	// Check bound resource groups
	if len(role.BoundResourceGroups) > 0 && !strListContains(role.BoundResourceGroups, resourceGroupName) {
		return errors.New("resource group not authorized")
	}

	// Check bound locations
	if len(role.BoundLocations) > 0 {
		if location == nil {
			return errors.New("location is empty")
		}
		if !strListContains(role.BoundLocations, convertPtrToString(location)) {
			return errors.New("location not authorized")
		}
	}

	return nil
}

func (b *azureAuthBackend) pathLoginRenew(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	roleName := req.Auth.InternalData["role"].(string)
	if roleName == "" {
		return nil, errors.New("failed to fetch role_name during renewal")
	}

	// Ensure that the Role still exists.
	role, err := b.role(ctx, req.Storage, roleName)
	if err != nil {
		return nil, errwrap.Wrapf(fmt.Sprintf("failed to validate role %s during renewal: {{err}}", roleName), err)
	}
	if role == nil {
		return nil, fmt.Errorf("role %s does not exist during renewal", roleName)
	}

	resp := &logical.Response{Auth: req.Auth}
	resp.Auth.TTL = role.TokenTTL
	resp.Auth.MaxTTL = role.TokenMaxTTL
	resp.Auth.Period = role.TokenPeriod
	return resp, nil
}

type additionalClaims struct {
	NotBefore jsonTime `json:"nbf"`
	ObjectID  string   `json:"oid"`
	GroupIDs  []string `json:"groups"`
}

const (
	pathLoginHelpSyn  = `Authenticates Azure Managed Service Identities with Vault.`
	pathLoginHelpDesc = `
Authenticate Azure Managed Service Identities.
`
)

func convertPtrToString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}
