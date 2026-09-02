// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package azure

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/google/uuid"

	"github.com/openbao/openbao-plugins/secrets/azure/api"
)

// mockProvider is a Provider that provides stubs and simple, deterministic responses.
type mockProvider struct {
	applications               map[string]string
	servicePrincipals          map[string]bool
	deletedObjects             map[string]bool
	passwords                  map[string]string
	failNextCreateApplication  bool
	failUnassignRoles          bool
	unassignRolesFailureParams failureParams
	ctxTimeout                 time.Duration
	lock                       sync.Mutex
}

type failureParams struct {
	statusCode  int
	expectError bool
}

func newMockProvider() AzureProvider {
	return &mockProvider{
		applications: map[string]string{
			// pre-populate applications map with Static obj ID
			// for TestStaticSPRead. In this test, CreateApplication is
			// not called and the test expects an app to exist.
			testStaticSPAppObjID: testStaticSPAppObjID,
		},
		servicePrincipals: make(map[string]bool),
		deletedObjects:    make(map[string]bool),
		passwords:         make(map[string]string),
	}
}

// ListRoles returns a single fake role based on the inbound filter
func (m *mockProvider) ListRoleDefinitions(_ context.Context, _ string, filter string) ([]*armauthorization.RoleDefinition, error) {
	reRoleName := regexp.MustCompile("roleName eq '(.*)'")

	match := reRoleName.FindAllStringSubmatch(filter, -1)
	if len(match) > 0 {
		name := match[0][1]
		if name == "multiple" {
			id1 := fmt.Sprintf("/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-%s-1", name)
			id2 := fmt.Sprintf("/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-%s-2", name)
			return []*armauthorization.RoleDefinition{
				{
					ID: &id1,
					Properties: &armauthorization.RoleDefinitionProperties{
						RoleName: &name,
					},
					Name: &name,
				},
				{
					ID: &id2,
					Properties: &armauthorization.RoleDefinitionProperties{
						RoleName: &name,
					},
					Name: &name,
				},
			}, nil
		}
		id := fmt.Sprintf("/subscriptions/FAKE_SUB_ID/providers/Microsoft.Authorization/roleDefinitions/FAKE_ROLE-%s", name)
		return []*armauthorization.RoleDefinition{
			{
				ID: &id,
				Properties: &armauthorization.RoleDefinitionProperties{
					RoleName: &name,
				},
				Name: &name,
			},
		}, nil
	}

	return []*armauthorization.RoleDefinition{}, nil
}

// GetRoleByID will return a fake role definition from the provided ID
// Assumes an ID format of: .*FAKE_ROLE-{rolename}
func (m *mockProvider) GetRoleDefinitionByID(_ context.Context, roleID string) (armauthorization.RoleDefinitionsClientGetByIDResponse, error) {
	s := strings.Split(roleID, "FAKE_ROLE-")
	roleName := s[1]
	return armauthorization.RoleDefinitionsClientGetByIDResponse{
		RoleDefinition: armauthorization.RoleDefinition{
			Properties: &armauthorization.RoleDefinitionProperties{},
			ID:         &roleID,
			Name:       &roleName,
		},
	}, nil
}

func (m *mockProvider) CreateServicePrincipal(_ context.Context, _ string, _ time.Time, _ time.Time) (spID string, password string, err error) {
	id := generateUUID()
	pass := generateUUID()

	m.lock.Lock()
	defer m.lock.Unlock()

	m.servicePrincipals[id] = true

	return id, pass, nil
}

func (m *mockProvider) CreateApplication(_ context.Context, _ string, _ string, _ []string) (api.Application, error) {
	if m.ctxTimeout != 0 {
		// simulate a context deadline error by sleeping for timeout period
		time.Sleep(m.ctxTimeout)
	}

	if m.failNextCreateApplication {
		m.failNextCreateApplication = false
		return api.Application{}, errors.New("Mock: fail to create application")
	}
	appObjID := generateUUID()
	appID := generateUUID()

	m.lock.Lock()
	defer m.lock.Unlock()

	m.applications[appObjID] = appID

	return api.Application{
		AppID:       appID,
		AppObjectID: appObjID,
	}, nil
}

func (m *mockProvider) GetApplication(_ context.Context, applicationObjectID string) (api.Application, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	appID := m.applications[applicationObjectID]

	return api.Application{
		AppID:       appID,
		AppObjectID: applicationObjectID,
	}, nil
}

func (m *mockProvider) ListApplications(_ context.Context, _ string) ([]api.Application, error) {
	return nil, nil
}

func (m *mockProvider) DeleteApplication(_ context.Context, applicationObjectID string, permanentlyDelete bool) error {
	delete(m.applications, applicationObjectID)
	m.deletedObjects[applicationObjectID] = true

	if permanentlyDelete {
		delete(m.deletedObjects, applicationObjectID)
	}
	return nil
}

func (m *mockProvider) DeleteServicePrincipal(_ context.Context, spObjectID string, permanentlyDelete bool) error {
	delete(m.servicePrincipals, spObjectID)
	m.deletedObjects[spObjectID] = true

	if permanentlyDelete {
		delete(m.deletedObjects, spObjectID)
	}
	return nil
}

func (m *mockProvider) AddApplicationPassword(_ context.Context, _ string, _ string, _ time.Time) (result api.PasswordCredential, err error) {
	keyID := uuid.New().String()
	pass := uuid.New().String()

	m.lock.Lock()
	defer m.lock.Unlock()
	m.passwords[keyID] = pass

	return api.PasswordCredential{
		KeyID:      keyID,
		SecretText: pass,
	}, nil
}

func (m *mockProvider) RemoveApplicationPassword(_ context.Context, _ string, keyID string) (err error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.passwords, keyID)

	return nil
}

func (m *mockProvider) deletedObjectExists(s string) bool {
	return m.deletedObjects[s]
}

func (m *mockProvider) appExists(s string) bool {
	_, ok := m.applications[s]
	return ok
}

func (m *mockProvider) passwordExists(s string) bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	_, ok := m.passwords[s]
	return ok
}

func (m *mockProvider) CreateRoleAssignment(_ context.Context, scope string, name string, params armauthorization.RoleAssignmentCreateParameters) (armauthorization.RoleAssignmentsClientCreateResponse, error) {
	return armauthorization.RoleAssignmentsClientCreateResponse{
		RoleAssignment: armauthorization.RoleAssignment{
			Properties: &armauthorization.RoleAssignmentProperties{
				Scope: &scope,
			},
			Name: &name,
			ID:   params.Properties.RoleDefinitionID,
		},
	}, nil
}

func (m *mockProvider) DeleteRoleAssignmentByID(_ context.Context, _ string) (armauthorization.RoleAssignmentsClientDeleteByIDResponse, error) {
	if m.failUnassignRoles {
		if m.unassignRolesFailureParams.expectError {
			// return empty response and no 200 status codes to throw error
			return armauthorization.RoleAssignmentsClientDeleteByIDResponse{}, &azcore.ResponseError{
				ErrorCode: "mock: fail to delete role assignment",
			}
		}

		// return empty response and with status code; will ignore error and assume role
		// assignment was manually deleted based on status code
		return armauthorization.RoleAssignmentsClientDeleteByIDResponse{}, &azcore.ResponseError{
			StatusCode: m.unassignRolesFailureParams.statusCode,
			ErrorCode:  "mock: fail to delete role assignment",
		}
	}
	return armauthorization.RoleAssignmentsClientDeleteByIDResponse{}, nil
}

// AddGroupMember adds a member to a Group.
func (m *mockProvider) AddGroupMember(_ context.Context, _ string, _ string) error {
	return nil
}

// RemoveGroupMember removes a member from a Group.
func (m *mockProvider) RemoveGroupMember(_ context.Context, _ string, _ string) error {
	return nil
}

// GetGroup gets group information from the directory.
func (m *mockProvider) GetGroup(_ context.Context, objectID string) (api.Group, error) {
	var groupName string
	s := strings.Split(objectID, "FAKE_GROUP-")
	if len(s) > 1 {
		groupName = s[1]
	}

	return api.Group{
		ID:          objectID,
		DisplayName: groupName,
	}, nil
}

// ListGroups gets list of groups for the current tenant.
func (m *mockProvider) ListGroups(_ context.Context, filter string) ([]api.Group, error) {
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
