package azuresecrets

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-01-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/hashicorp/errwrap"
	multierror "github.com/hashicorp/go-multierror"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/vault/logical"
)

const appNamePrefix = "vault-"

// client offers higher level Azure operations that provide a simpler interface
// for handlers. It in turn relies on a Provider interface to access the lower level
// Azure Client SDK methods.
type client struct {
	provider AzureProvider
	settings *clientSettings
}

func (b *azureSecretBackend) getClient(ctx context.Context, s logical.Storage) (*client, error) {
	b.lock.RLock()
	unlockFunc := b.lock.RUnlock
	defer func() { unlockFunc() }()

	if b.settings == nil {
		// Upgrade lock
		b.lock.RUnlock()
		b.lock.Lock()
		unlockFunc = b.lock.Unlock

		// Create a new client from the stored or empty config
		config, err := b.getConfig(ctx, s)
		if err != nil {
			return nil, err
		}
		if config == nil {
			config = new(azureConfig)
		}

		settings, err := getClientSettings(config)
		if err != nil {
			return nil, err
		}
		b.settings = settings
	}

	p, err := b.getProvider(b.settings)
	if err != nil {
		return nil, err
	}

	c := &client{
		provider: p,
		settings: b.settings,
	}

	return c, nil
}

// createApp creates a new Azure application.
// An Application is a needed to create service principals used by
// the caller for authentication.
func (c *client) createApp(ctx context.Context) (app *graphrbac.Application, err error) {
	name, err := uuid.GenerateUUID()
	if err != nil {
		return nil, err
	}

	name = appNamePrefix + name

	appURL := fmt.Sprintf("https://%s", name)

	result, err := c.provider.CreateApplication(ctx, graphrbac.ApplicationCreateParameters{
		AvailableToOtherTenants: to.BoolPtr(false),
		DisplayName:             to.StringPtr(name),
		Homepage:                to.StringPtr(appURL),
		IdentifierUris:          to.StringSlicePtr([]string{appURL}),
	})

	return &result, err
}

// createSP creates a new service principal.
func (c *client) createSP(
	ctx context.Context,
	app *graphrbac.Application,
	duration time.Duration) (*graphrbac.ServicePrincipal, string, error) {

	// Generate a random key (which must be a UUID) and password
	keyID, err := uuid.GenerateUUID()
	if err != nil {
		return nil, "", err
	}

	password, err := uuid.GenerateUUID()
	if err != nil {
		return nil, "", err
	}

	now := time.Now()
	result, err := c.provider.CreateServicePrincipal(ctx, graphrbac.ServicePrincipalCreateParameters{
		AppID:          app.AppID,
		AccountEnabled: to.BoolPtr(true),
		PasswordCredentials: &[]graphrbac.PasswordCredential{
			graphrbac.PasswordCredential{
				StartDate: &date.Time{Time: now},
				EndDate:   &date.Time{Time: now.Add(duration)},
				KeyID:     to.StringPtr(keyID),
				Value:     to.StringPtr(password),
			},
		},
	})

	return &result, password, err
}

// deleteApp deletes an Azure application.
func (c *client) deleteApp(ctx context.Context, appObjectID string) error {
	resp, err := c.provider.DeleteApplication(ctx, appObjectID)

	// Don't consider it an error if the object wasn't present
	if err != nil && resp.StatusCode == 404 {
		return nil
	}

	return err
}

// assignRoles assigns Azure roles to a service principal.
func (c *client) assignRoles(ctx context.Context, sp *graphrbac.ServicePrincipal, roles []*azureRole) ([]string, error) {
	var ids []string

	for _, role := range roles {
		assignmentID, err := uuid.GenerateUUID()
		if err != nil {
			return nil, err
		}

		// retries are essential for this operation as there can be a significant delay between
		// when a service principal is created and when it is visible for other operations. If
		// it isn't visible yet, "PrincipalNotFound" is the error received and is not treated
		// as an error here. The retry logic (5 seconds sleeps, max 36 tries) is straight from
		// the az cli behavior and was confirmed as acceptable by Microsoft.
		assigned := false
		for i := 0; i < 36; i++ {
			ra, err := c.provider.CreateRoleAssignment(ctx, role.Scope, assignmentID,
				authorization.RoleAssignmentCreateParameters{
					RoleAssignmentProperties: &authorization.RoleAssignmentProperties{
						RoleDefinitionID: to.StringPtr(role.RoleID),
						PrincipalID:      sp.ObjectID,
					},
				})

			if err == nil {
				ids = append(ids, to.String(ra.ID))
				assigned = true
				break
			}

			if !strings.Contains(err.Error(), "PrincipalNotFound") {
				return nil, errwrap.Wrapf("error while assigning role: {{err}}", err)
			}

			t := time.NewTimer(5 * time.Second)
			select {
			case <-t.C:
			case <-ctx.Done():
				return nil, errors.New("role assignment operation cancelled")
			}
		}

		if !assigned {
			return nil, fmt.Errorf("unable to assign roles. Principal ID not found: %s", to.String(sp.ObjectID))
		}
	}

	return ids, nil
}

// unassignRoles deletes role assignments, if they existed.
// This is a clean-up operation that isn't essential to revocation. As such, an
// attempt is made to remove all assignments, and not return immediately if there
// is an error.
func (c *client) unassignRoles(ctx context.Context, roleIDs []string) error {
	var merr *multierror.Error

	for _, id := range roleIDs {
		if _, err := c.provider.DeleteRoleAssignmentByID(ctx, id); err != nil {
			merr = multierror.Append(merr, errwrap.Wrapf("error unassigning role: {{err}}", err))
		}
	}

	return merr.ErrorOrNil()
}

// search for roles by name
func (c *client) findRoles(ctx context.Context, roleName string) ([]authorization.RoleDefinition, error) {
	return c.provider.ListRoles(ctx, fmt.Sprintf("subscriptions/%s", c.settings.SubscriptionID), fmt.Sprintf("roleName eq '%s'", roleName))
}

// clientSettings is used by a client to configure the connections to Azure.
// It is created from a combination of Vault config settings and environment variables.
type clientSettings struct {
	SubscriptionID string
	TenantID       string
	ClientID       string
	ClientSecret   string
	Environment    azure.Environment
}

// getClientSettings creates a new clientSettings object.
// Environment variables have higher precedence than stored configuration.
func getClientSettings(config *azureConfig) (*clientSettings, error) {
	firstAvailable := func(opts ...string) string {
		for _, s := range opts {
			if s != "" {
				return s
			}
		}
		return ""
	}

	settings := new(clientSettings)

	settings.ClientID = firstAvailable(os.Getenv("AZURE_CLIENT_ID"), config.ClientID)
	settings.ClientSecret = firstAvailable(os.Getenv("AZURE_CLIENT_SECRET"), config.ClientSecret)
	settings.SubscriptionID = firstAvailable(os.Getenv("AZURE_SUBSCRIPTION_ID"))

	settings.SubscriptionID = firstAvailable(os.Getenv("AZURE_SUBSCRIPTION_ID"), config.SubscriptionID)
	if settings.SubscriptionID == "" {
		return nil, errors.New("subscription_id is required")
	}

	settings.TenantID = firstAvailable(os.Getenv("AZURE_TENANT_ID"), config.TenantID)
	if settings.TenantID == "" {
		return nil, errors.New("tenant_id is required")
	}

	envName := firstAvailable(os.Getenv("AZURE_ENVIRONMENT"), config.Environment, "AZUREPUBLICCLOUD")
	env, err := azure.EnvironmentFromName(envName)
	if err != nil {
		return nil, err
	}
	settings.Environment = env

	return settings, nil
}
