package azuresecrets

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-01-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/helper/logging"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	defaultLeaseTTLHr = 1 * time.Hour
	maxLeaseTTLHr     = 12 * time.Hour
	defaultTestTTL    = 300
	defaultTestMaxTTL = 3600
)

func getTestBackend(t *testing.T, initConfig bool) (*azureSecretBackend, logical.Storage) {
	b := backend()

	config := &logical.BackendConfig{
		Logger: logging.NewVaultLogger(log.Trace),
		System: &logical.StaticSystemView{
			DefaultLeaseTTLVal: defaultLeaseTTLHr,
			MaxLeaseTTLVal:     maxLeaseTTLHr,
		},
		StorageView: &logical.InmemStorage{},
	}
	err := b.Setup(context.Background(), config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}

	b.settings = new(clientSettings)
	mockProvider := newMockProvider()
	b.getProvider = func(s *clientSettings) (AzureProvider, error) {
		return mockProvider, nil
	}

	if initConfig {
		cfg := map[string]interface{}{
			"subscription_id": generateUUID(),
			"tenant_id":       generateUUID(),
			"client_id":       "testClientId",
			"client_secret":   "testClientSecret",
			"environment":     "AZURECHINACLOUD",
			"ttl":             defaultTestTTL,
			"max_ttl":         defaultTestMaxTTL,
		}

		testConfigCreate(t, b, config.StorageView, cfg)
	}

	return b, config.StorageView
}

// mockProvider is a Provider that provides stubs and simple, deterministic responses.
type mockProvider struct {
	subscriptionID            string
	applications              map[string]bool
	passwords                 map[string]bool
	failNextCreateApplication bool
}

// errMockProvider simulates a normal provider which fails to associate a role,
// returning an error
type errMockProvider struct {
	*mockProvider
}

// CreateRoleAssignment for the errMockProvider intentionally fails
func (e *errMockProvider) CreateRoleAssignment(ctx context.Context, scope string, roleAssignmentName string, parameters authorization.RoleAssignmentCreateParameters) (authorization.RoleAssignment, error) {
	return authorization.RoleAssignment{}, errors.New("PrincipalNotFound")
}

// GetApplication for the errMockProvider only returns an application if that
// key is found, unlike mockProvider which returns the same application object
// id each time. Existing tests depend on the mockProvider behavior, which is
// why errMockProvider has it's own version.
func (e *errMockProvider) GetApplication(ctx context.Context, applicationObjectID string) (graphrbac.Application, error) {
	for s := range e.applications {
		if s == applicationObjectID {
			return graphrbac.Application{
				AppID: to.StringPtr(s),
			}, nil
		}
	}
	return graphrbac.Application{}, errors.New("not found")
}

func newErrMockProvider() AzureProvider {
	return &errMockProvider{
		mockProvider: &mockProvider{
			subscriptionID: generateUUID(),
			applications:   make(map[string]bool),
			passwords:      make(map[string]bool),
		},
	}
}

func newMockProvider() AzureProvider {
	return &mockProvider{
		subscriptionID: generateUUID(),
		applications:   make(map[string]bool),
		passwords:      make(map[string]bool),
	}
}

// ListRoles returns a single fake role based on the inbound filter
func (m *mockProvider) ListRoles(ctx context.Context, scope string, filter string) (result []authorization.RoleDefinition, err error) {
	reRoleName := regexp.MustCompile("roleName eq '(.*)'")

	match := reRoleName.FindAllStringSubmatch(filter, -1)
	if len(match) > 0 {
		name := match[0][1]
		if name == "multiple" {
			return []authorization.RoleDefinition{
				{
					ID: to.StringPtr(fmt.Sprintf("/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-%s-1", name)),
					RoleDefinitionProperties: &authorization.RoleDefinitionProperties{
						RoleName: to.StringPtr(name),
					},
				},
				{
					ID: to.StringPtr(fmt.Sprintf("/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-%s-2", name)),
					RoleDefinitionProperties: &authorization.RoleDefinitionProperties{
						RoleName: to.StringPtr(name),
					},
				},
			}, nil
		}
		return []authorization.RoleDefinition{
			{
				ID: to.StringPtr(fmt.Sprintf("/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-%s", name)),
				RoleDefinitionProperties: &authorization.RoleDefinitionProperties{
					RoleName: to.StringPtr(name),
				},
			},
		}, nil
	}

	return []authorization.RoleDefinition{}, nil
}

// GetRoleByID will returns a fake role definition from the povided ID
// Assumes an ID format of: .*FAKE_ROLE-{rolename}
func (m *mockProvider) GetRoleByID(ctx context.Context, roleID string) (result authorization.RoleDefinition, err error) {
	d := authorization.RoleDefinition{}
	s := strings.Split(roleID, "FAKE_ROLE-")
	if len(s) > 1 {
		d.ID = to.StringPtr(roleID)
		d.RoleDefinitionProperties = &authorization.RoleDefinitionProperties{
			RoleName: to.StringPtr(s[1]),
		}
	}

	return d, nil
}

func (m *mockProvider) CreateServicePrincipal(ctx context.Context, parameters graphrbac.ServicePrincipalCreateParameters) (graphrbac.ServicePrincipal, error) {
	return graphrbac.ServicePrincipal{
		ObjectID: to.StringPtr(generateUUID()),
	}, nil
}

func (m *mockProvider) CreateApplication(ctx context.Context, parameters graphrbac.ApplicationCreateParameters) (graphrbac.Application, error) {
	if m.failNextCreateApplication {
		m.failNextCreateApplication = false
		return graphrbac.Application{}, errors.New("Mock: fail to create application")
	}
	appObjID := generateUUID()
	m.applications[appObjID] = true

	return graphrbac.Application{
		AppID:    to.StringPtr(generateUUID()),
		ObjectID: &appObjID,
	}, nil
}

func (m *mockProvider) GetApplication(ctx context.Context, applicationObjectID string) (graphrbac.Application, error) {
	return graphrbac.Application{
		AppID: to.StringPtr("00000000-0000-0000-0000-000000000000"),
	}, nil
}

func (m *mockProvider) DeleteApplication(ctx context.Context, applicationObjectID string) (autorest.Response, error) {
	delete(m.applications, applicationObjectID)
	return autorest.Response{}, nil
}

func (m *mockProvider) UpdateApplicationPasswordCredentials(ctx context.Context, applicationObjectID string, parameters graphrbac.PasswordCredentialsUpdateParameters) (result autorest.Response, err error) {
	m.passwords = make(map[string]bool)
	for _, v := range *parameters.Value {
		m.passwords[*v.KeyID] = true
	}

	return autorest.Response{}, nil
}

func (m *mockProvider) ListApplicationPasswordCredentials(ctx context.Context, applicationObjectID string) (result graphrbac.PasswordCredentialListResult, err error) {
	var creds []graphrbac.PasswordCredential
	for keyID := range m.passwords {
		creds = append(creds, graphrbac.PasswordCredential{KeyID: &keyID})
	}

	return graphrbac.PasswordCredentialListResult{
		Value: &creds,
	}, nil
}

func (m *mockProvider) appExists(s string) bool {
	return m.applications[s]
}

func (m *mockProvider) passwordExists(s string) bool {
	return m.passwords[s]
}

func (m *mockProvider) VMGet(ctx context.Context, resourceGroupName string, VMName string, expand compute.InstanceViewTypes) (result compute.VirtualMachine, err error) {
	return compute.VirtualMachine{}, nil
}

func (m *mockProvider) VMUpdate(ctx context.Context, resourceGroupName string, VMName string, parameters compute.VirtualMachineUpdate) (result compute.VirtualMachinesUpdateFuture, err error) {
	return compute.VirtualMachinesUpdateFuture{}, nil
}

func (m *mockProvider) CreateRoleAssignment(ctx context.Context, scope string, roleAssignmentName string, parameters authorization.RoleAssignmentCreateParameters) (authorization.RoleAssignment, error) {
	return authorization.RoleAssignment{
		ID: to.StringPtr(generateUUID()),
	}, nil
}

func (m *mockProvider) DeleteRoleAssignmentByID(ctx context.Context, roleID string) (result authorization.RoleAssignment, err error) {
	return authorization.RoleAssignment{}, nil
}

// AddGroupMember adds a member to a AAD Group.
func (m *mockProvider) AddGroupMember(ctx context.Context, groupObjectID string, parameters graphrbac.GroupAddMemberParameters) (result autorest.Response, err error) {
	return autorest.Response{}, nil
}

// RemoveGroupMember removes a member from a AAD Group.
func (m *mockProvider) RemoveGroupMember(ctx context.Context, groupObjectID, memberObjectID string) (result autorest.Response, err error) {
	return autorest.Response{}, nil
}

// GetGroup gets group information from the directory.
func (m *mockProvider) GetGroup(ctx context.Context, objectID string) (result graphrbac.ADGroup, err error) {
	g := graphrbac.ADGroup{
		ObjectID: to.StringPtr(objectID),
	}
	s := strings.Split(objectID, "FAKE_GROUP-")
	if len(s) > 1 {
		g.DisplayName = to.StringPtr(s[1])
	}

	return g, nil
}

// ListGroups gets list of groups for the current tenant.
func (m *mockProvider) ListGroups(ctx context.Context, filter string) (result []graphrbac.ADGroup, err error) {
	reGroupName := regexp.MustCompile("displayName eq '(.*)'")

	match := reGroupName.FindAllStringSubmatch(filter, -1)
	if len(match) > 0 {
		name := match[0][1]
		if name == "multiple" {
			return []graphrbac.ADGroup{
				{
					ObjectID:    to.StringPtr(fmt.Sprintf("00000000-1111-2222-3333-444444444444FAKE_GROUP-%s-1", name)),
					DisplayName: to.StringPtr(name),
				},
				{
					ObjectID:    to.StringPtr(fmt.Sprintf("00000000-1111-2222-3333-444444444444FAKE_GROUP-%s-2", name)),
					DisplayName: to.StringPtr(name),
				},
			}, nil
		}

		return []graphrbac.ADGroup{
			{
				ObjectID:    to.StringPtr(fmt.Sprintf("00000000-1111-2222-3333-444444444444FAKE_GROUP-%s", name)),
				DisplayName: to.StringPtr(name),
			},
		}, nil
	}

	return []graphrbac.ADGroup{}, nil
}
