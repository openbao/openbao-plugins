package azuresecrets

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/authorization/mgmt/authorization"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/hashicorp/vault-plugin-secrets-azure/api"
)

// mockProvider is a Provider that provides stubs and simple, deterministic responses.
type mockProvider struct {
	subscriptionID            string
	applications              map[string]bool
	passwords                 map[string]api.PasswordCredential
	failNextCreateApplication bool
	lock                      sync.Mutex
}

func newMockProvider() api.AzureProvider {
	return &mockProvider{
		subscriptionID: generateUUID(),
		applications:   make(map[string]bool),
		passwords:      make(map[string]api.PasswordCredential),
	}
}

// ListRoles returns a single fake role based on the inbound filter
func (m *mockProvider) ListRoleDefinitions(_ context.Context, _ string, filter string) (result []authorization.RoleDefinition, err error) {
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
func (m *mockProvider) GetRoleDefinitionByID(_ context.Context, roleID string) (result authorization.RoleDefinition, err error) {
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

func (m *mockProvider) CreateServicePrincipal(_ context.Context, _ string, _ time.Time, _ time.Time) (string, string, error) {
	id := generateUUID()
	pass := generateUUID()
	return id, pass, nil
}

func (m *mockProvider) CreateApplication(_ context.Context, _ string) (api.ApplicationResult, error) {
	if m.failNextCreateApplication {
		m.failNextCreateApplication = false
		return api.ApplicationResult{}, errors.New("Mock: fail to create application")
	}
	appObjID := generateUUID()

	m.lock.Lock()
	defer m.lock.Unlock()

	m.applications[appObjID] = true

	return api.ApplicationResult{
		AppID: to.StringPtr(generateUUID()),
		ID:    &appObjID,
	}, nil
}

func (m *mockProvider) GetApplication(_ context.Context, _ string) (api.ApplicationResult, error) {
	return api.ApplicationResult{
		AppID: to.StringPtr("00000000-0000-0000-0000-000000000000"),
	}, nil
}

func (m *mockProvider) DeleteApplication(_ context.Context, applicationObjectID string) (autorest.Response, error) {
	delete(m.applications, applicationObjectID)
	return autorest.Response{}, nil
}

func (m *mockProvider) AddApplicationPassword(_ context.Context, _ string, displayName string, endDateTime date.Time) (result api.PasswordCredentialResult, err error) {
	keyID := generateUUID()
	cred := api.PasswordCredential{
		DisplayName: to.StringPtr(displayName),
		StartDate:   &date.Time{Time: time.Now()},
		EndDate:     &endDateTime,
		KeyID:       to.StringPtr(keyID),
		SecretText:  to.StringPtr(generateUUID()),
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	m.passwords[keyID] = cred

	return api.PasswordCredentialResult{
		PasswordCredential: cred,
	}, nil
}

func (m *mockProvider) RemoveApplicationPassword(_ context.Context, _ string, keyID string) (result autorest.Response, err error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.passwords, keyID)

	return autorest.Response{}, nil
}

func (m *mockProvider) appExists(s string) bool {
	return m.applications[s]
}

func (m *mockProvider) passwordExists(s string) bool {
	_, ok := m.passwords[s]
	return ok
}

func (m *mockProvider) VMGet(_ context.Context, _ string, _ string, _ compute.InstanceViewTypes) (result compute.VirtualMachine, err error) {
	return compute.VirtualMachine{}, nil
}

func (m *mockProvider) VMUpdate(_ context.Context, _ string, _ string, _ compute.VirtualMachineUpdate) (result compute.VirtualMachinesUpdateFuture, err error) {
	return compute.VirtualMachinesUpdateFuture{}, nil
}

func (m *mockProvider) CreateRoleAssignment(_ context.Context, _ string, _ string, _ authorization.RoleAssignmentCreateParameters) (authorization.RoleAssignment, error) {
	return authorization.RoleAssignment{
		ID: to.StringPtr(generateUUID()),
	}, nil
}

func (m *mockProvider) DeleteRoleAssignmentByID(_ context.Context, _ string) (result authorization.RoleAssignment, err error) {
	return authorization.RoleAssignment{}, nil
}

// AddGroupMember adds a member to a AAD Group.
func (m *mockProvider) AddGroupMember(_ context.Context, _ string, _ string) (err error) {
	return nil
}

// RemoveGroupMember removes a member from a AAD Group.
func (m *mockProvider) RemoveGroupMember(_ context.Context, _ string, _ string) (err error) {
	return nil
}

// GetGroup gets group information from the directory.
func (m *mockProvider) GetGroup(_ context.Context, objectID string) (api.Group, error) {
	g := api.Group{
		ID: objectID,
	}
	s := strings.Split(objectID, "FAKE_GROUP-")
	if len(s) > 1 {
		g.DisplayName = s[1]
	}

	return g, nil
}

// ListGroups gets list of groups for the current tenant.
func (m *mockProvider) ListGroups(_ context.Context, filter string) (result []api.Group, err error) {
	reGroupName := regexp.MustCompile("displayName eq '(.*)'")

	match := reGroupName.FindAllStringSubmatch(filter, -1)
	if len(match) > 0 {
		name := match[0][1]
		if name == "multiple" {
			return []api.Group{
				{
					ID:          fmt.Sprintf("00000000-1111-2222-3333-444444444444FAKE_GROUP-%s-1", name),
					DisplayName: name,
				},
				{
					ID:          fmt.Sprintf("00000000-1111-2222-3333-444444444444FAKE_GROUP-%s-2", name),
					DisplayName: name,
				},
			}, nil
		}

		return []api.Group{
			{
				ID:          fmt.Sprintf("00000000-1111-2222-3333-444444444444FAKE_GROUP-%s", name),
				DisplayName: name,
			},
		}, nil
	}

	return []api.Group{}, nil
}

// errMockProvider simulates a normal provider which fails to associate a role,
// returning an error
type errMockProvider struct {
	*mockProvider
}

func newErrMockProvider() api.AzureProvider {
	return &errMockProvider{
		mockProvider: &mockProvider{
			subscriptionID: generateUUID(),
			applications:   make(map[string]bool),
			passwords:      make(map[string]api.PasswordCredential),
		},
	}
}

// CreateRoleAssignment for the errMockProvider intentionally fails
func (e *errMockProvider) CreateRoleAssignment(ctx context.Context, scope string, roleAssignmentName string, parameters authorization.RoleAssignmentCreateParameters) (authorization.RoleAssignment, error) {
	return authorization.RoleAssignment{}, errors.New("PrincipalNotFound")
}

// GetApplication for the errMockProvider only returns an application if that
// key is found, unlike mockProvider which returns the same application object
// id each time. Existing tests depend on the mockProvider behavior, which is
// why errMockProvider has it's own version.
func (e *errMockProvider) GetApplication(ctx context.Context, applicationObjectID string) (api.ApplicationResult, error) {
	for s := range e.applications {
		if s == applicationObjectID {
			return api.ApplicationResult{
				AppID: to.StringPtr(s),
			}, nil
		}
	}
	return api.ApplicationResult{}, errors.New("not found")
}
