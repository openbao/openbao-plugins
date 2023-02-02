// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package azuresecrets

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-01-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/hashicorp/vault-plugin-secrets-azure/api"
	"github.com/hashicorp/vault/sdk/helper/useragent"
)

var _ api.AzureProvider = (*provider)(nil)

// provider is a concrete implementation of AzureProvider. In most cases it is a simple passthrough
// to the appropriate client object. But if the response requires processing that is more practical
// at this layer, the response signature may different from the Azure signature.
type provider struct {
	settings *clientSettings

	appClient    api.ApplicationsClient
	spClient     api.ServicePrincipalClient
	groupsClient api.GroupsClient
	raClient     *authorization.RoleAssignmentsClient
	rdClient     *authorization.RoleDefinitionsClient
}

// newAzureProvider creates an azureProvider, backed by Azure client objects for underlying services.
func newAzureProvider(settings *clientSettings, passwords api.Passwords) (api.AzureProvider, error) {
	// build clients that use the GraphRBAC endpoint
	userAgent := useragent.PluginString(settings.PluginEnv, userAgentPluginName)

	var appClient api.ApplicationsClient
	var groupsClient api.GroupsClient
	var spClient api.ServicePrincipalClient

	graphURI, err := api.GetGraphURI(settings.Environment.Name)
	if err != nil {
		return nil, err
	}

	graphApiAuthorizer, err := getAuthorizer(settings, graphURI)
	if err != nil {
		return nil, err
	}

	msGraphAppClient, err := api.NewMSGraphApplicationClient(settings.SubscriptionID, userAgent, graphURI, graphApiAuthorizer)
	if err != nil {
		return nil, err
	}

	appClient = msGraphAppClient
	groupsClient = msGraphAppClient
	spClient = msGraphAppClient

	// build clients that use the Resource Manager endpoint
	resourceManagerAuthorizer, err := getAuthorizer(settings, settings.Environment.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	raClient := authorization.NewRoleAssignmentsClientWithBaseURI(settings.Environment.ResourceManagerEndpoint, settings.SubscriptionID)
	raClient.Authorizer = resourceManagerAuthorizer
	raClient.AddToUserAgent(userAgent)

	rdClient := authorization.NewRoleDefinitionsClientWithBaseURI(settings.Environment.ResourceManagerEndpoint, settings.SubscriptionID)
	rdClient.Authorizer = resourceManagerAuthorizer
	rdClient.AddToUserAgent(userAgent)

	p := &provider{
		settings: settings,

		appClient:    appClient,
		spClient:     spClient,
		groupsClient: groupsClient,
		raClient:     &raClient,
		rdClient:     &rdClient,
	}

	return p, nil
}

// getAuthorizer attempts to create an authorizer, preferring ClientID/Secret if present,
// and falling back to MSI if not.
func getAuthorizer(settings *clientSettings, resource string) (autorest.Authorizer, error) {
	if settings.ClientID != "" && settings.ClientSecret != "" && settings.TenantID != "" {
		config := auth.NewClientCredentialsConfig(settings.ClientID, settings.ClientSecret, settings.TenantID)
		config.AADEndpoint = settings.Environment.ActiveDirectoryEndpoint
		config.Resource = resource
		return config.Authorizer()
	}

	config := auth.NewMSIConfig()
	config.Resource = resource
	return config.Authorizer()
}

// CreateApplication create a new Azure application object.
func (p *provider) CreateApplication(ctx context.Context, displayName string) (result api.ApplicationResult, err error) {
	return p.appClient.CreateApplication(ctx, displayName)
}

func (p *provider) GetApplication(ctx context.Context, applicationObjectID string) (result api.ApplicationResult, err error) {
	return p.appClient.GetApplication(ctx, applicationObjectID)
}

func (p *provider) ListApplications(ctx context.Context, filter string) ([]api.ApplicationResult, error) {
	return p.appClient.ListApplications(ctx, filter)
}

// DeleteApplication deletes an Azure application object.
// This will in turn remove the service principal (but not the role assignments).
func (p *provider) DeleteApplication(ctx context.Context, applicationObjectID string, permanentlyDelete bool) error {
	return p.appClient.DeleteApplication(ctx, applicationObjectID, permanentlyDelete)
}

func (p *provider) AddApplicationPassword(ctx context.Context, applicationObjectID string, displayName string, endDateTime time.Time) (result api.PasswordCredentialResult, err error) {
	return p.appClient.AddApplicationPassword(ctx, applicationObjectID, displayName, endDateTime)
}

func (p *provider) RemoveApplicationPassword(ctx context.Context, applicationObjectID string, keyID string) (err error) {
	return p.appClient.RemoveApplicationPassword(ctx, applicationObjectID, keyID)
}

// CreateServicePrincipal creates a new Azure service principal.
// An Application must be created prior to calling this and pass in parameters.
func (p *provider) CreateServicePrincipal(ctx context.Context, appID string, startDate time.Time, endDate time.Time) (id string, password string, err error) {
	return p.spClient.CreateServicePrincipal(ctx, appID, startDate, endDate)
}

func (p *provider) DeleteServicePrincipal(ctx context.Context, spObjectID string, permanentlyDelete bool) error {
	return p.spClient.DeleteServicePrincipal(ctx, spObjectID, permanentlyDelete)
}

// ListRoles like all Azure roles with a scope (often subscription).
func (p *provider) ListRoleDefinitions(ctx context.Context, scope string, filter string) (result []authorization.RoleDefinition, err error) {
	page, err := p.rdClient.List(ctx, scope, filter)

	if err != nil {
		return nil, err
	}

	return page.Values(), nil
}

// GetRoleByID fetches the full role definition given a roleID.
func (p *provider) GetRoleDefinitionByID(ctx context.Context, roleID string) (result authorization.RoleDefinition, err error) {
	return p.rdClient.GetByID(ctx, roleID)
}

// CreateRoleAssignment assigns a role to a service principal.
func (p *provider) CreateRoleAssignment(ctx context.Context, scope string, roleAssignmentName string, parameters authorization.RoleAssignmentCreateParameters) (authorization.RoleAssignment, error) {
	return p.raClient.Create(ctx, scope, roleAssignmentName, parameters)
}

// GetRoleAssignmentByID fetches the full role assignment info given a roleAssignmentID.
func (p *provider) GetRoleAssignmentByID(ctx context.Context, roleAssignmentID string) (result authorization.RoleAssignment, err error) {
	return p.raClient.GetByID(ctx, roleAssignmentID)
}

// DeleteRoleAssignmentByID deletes a role assignment.
func (p *provider) DeleteRoleAssignmentByID(ctx context.Context, roleAssignmentID string) (result authorization.RoleAssignment, err error) {
	return p.raClient.DeleteByID(ctx, roleAssignmentID)
}

// ListRoleAssignments lists all role assignments.
// There is no need for paging; the caller only cares about the first match and whether
// there are 0, 1 or >1 items. Unpacking here is a simpler interface.
func (p *provider) ListRoleAssignments(ctx context.Context, filter string) ([]authorization.RoleAssignment, error) {
	page, err := p.raClient.List(ctx, filter)

	if err != nil {
		return nil, err
	}

	return page.Values(), nil
}

// AddGroupMember adds a member to a Group.
func (p *provider) AddGroupMember(ctx context.Context, groupObjectID string, memberObjectID string) (err error) {
	return p.groupsClient.AddGroupMember(ctx, groupObjectID, memberObjectID)
}

// RemoveGroupMember removes a member from a Group.
func (p *provider) RemoveGroupMember(ctx context.Context, groupObjectID, memberObjectID string) (err error) {
	return p.groupsClient.RemoveGroupMember(ctx, groupObjectID, memberObjectID)
}

// GetGroup gets group information from the directory.
func (p *provider) GetGroup(ctx context.Context, objectID string) (result api.Group, err error) {
	return p.groupsClient.GetGroup(ctx, objectID)
}

// ListGroups gets list of groups for the current tenant.
func (p *provider) ListGroups(ctx context.Context, filter string) (result []api.Group, err error) {
	return p.groupsClient.ListGroups(ctx, filter)
}
