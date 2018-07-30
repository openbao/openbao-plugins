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
	"github.com/hashicorp/vault/helper/logging"
	"github.com/hashicorp/vault/logical"
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
	failNextCreateApplication bool
}

func newMockProvider() AzureProvider {
	return &mockProvider{
		subscriptionID: generateUUID(),
		applications:   make(map[string]bool),
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
	return graphrbac.ServicePrincipal{}, nil
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

func (m *mockProvider) DeleteApplication(ctx context.Context, applicationObjectID string) (autorest.Response, error) {
	delete(m.applications, applicationObjectID)
	return autorest.Response{}, nil
}

func (m *mockProvider) appExists(s string) bool {
	return m.applications[s]
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
