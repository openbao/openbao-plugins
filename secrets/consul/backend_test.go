// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package consul

import (
	"context"
	"fmt"
	"log"
	"os"
	"reflect"
	"testing"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/mitchellh/mapstructure"
	consul "github.com/openbao/openbao-plugins/secrets/consul/testhelpers"
	logicaltest "github.com/openbao/openbao/helper/testhelpers/logical"
	"github.com/openbao/openbao/sdk/v2/logical"
)

func testOldestAndLatestSupported(t *testing.T, f func(t *testing.T, versionId string)) {
	t.Run("latest-supported", func(t *testing.T) {
		f(t, "latest-supported")
	})

	t.Run("oldest-supported", func(t *testing.T) {
		f(t, "oldest-supported")
	})
}

func TestBackend_Config_Access(t *testing.T) {
	t.Parallel()

	t.Run("no automatic bootstrap", func(t *testing.T) {
		t.Parallel()
		testOldestAndLatestSupported(t, func(t *testing.T, versionId string) {
			t.Parallel()
			testBackendConfigAccess(t, versionId, false)
		})
	})

	t.Run("automatic bootstrap", func(t *testing.T) {
		t.Parallel()
		testOldestAndLatestSupported(t, func(t *testing.T, versionId string) {
			t.Parallel()
			testBackendConfigAccess(t, versionId, true)
		})
	})
}

func testBackendConfigAccess(t *testing.T, version string, autoBootstrap bool) {
	config := logical.TestBackendConfig()
	config.StorageView = &logical.InmemStorage{}
	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}

	cleanup, consulConfig := consul.PrepareTestContainer(t, version, false, !autoBootstrap)
	defer cleanup()

	connData := map[string]any{
		"address": consulConfig.Address(),
	}
	if autoBootstrap {
		if consulConfig.Token != "" {
			t.Fatal("expected bootstrap not to have happened yet")
		}
	} else {
		connData["token"] = consulConfig.Token
	}

	confReq := &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "config/access",
		Storage:   config.StorageView,
		Data:      connData,
	}

	resp, err := b.HandleRequest(context.Background(), confReq)
	if err != nil || (resp != nil && resp.IsError()) || resp != nil {
		t.Fatalf("failed to write configuration: resp:%#v err:%s", resp, err)
	}

	confReq.Operation = logical.ReadOperation
	resp, err = b.HandleRequest(context.Background(), confReq)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("failed to write configuration: resp:%#v err:%s", resp, err)
	}

	expected := map[string]any{
		"address": connData["address"].(string),
		"scheme":  "http",
	}
	if !reflect.DeepEqual(expected, resp.Data) {
		t.Fatalf("bad: expected:%#v\nactual:%#v\n", expected, resp.Data)
	}
	if resp.Data["token"] != nil {
		t.Fatalf("token should not be set in the response")
	}
}

func TestBackend_Renew_Revoke(t *testing.T) {
	t.Parallel()
	testOldestAndLatestSupported(t, func(t *testing.T, versionId string) {
		t.Parallel()
		testBackendRenewRevoke(t, versionId)
	})
}

func testBackendRenewRevoke(t *testing.T, version string) {
	config := logical.TestBackendConfig()
	config.StorageView = &logical.InmemStorage{}
	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}

	cleanup, consulConfig := consul.PrepareTestContainer(t, version, false, true)
	defer cleanup()

	connData := map[string]any{
		"address": consulConfig.Address(),
		"token":   consulConfig.Token,
	}

	req := &logical.Request{
		Storage:   config.StorageView,
		Operation: logical.UpdateOperation,
		Path:      "config/access",
		Data:      connData,
	}
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	req.Path = "roles/test"
	req.Data = map[string]any{
		"consul_policies": []string{"test"},
		"lease":           "6h",
	}
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	req.Operation = logical.ReadOperation
	req.Path = "creds/test"
	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("resp nil")
	}
	if resp.IsError() {
		t.Fatalf("resp is error: %v", resp.Error())
	}

	generatedSecret := resp.Secret
	generatedSecret.TTL = 6 * time.Hour

	var d struct {
		Token string `mapstructure:"token"`
	}
	if err := mapstructure.Decode(resp.Data, &d); err != nil {
		t.Fatal(err)
	}

	// Build a client and verify that the credentials work
	consulapiConfig := consulapi.DefaultConfig()
	consulapiConfig.Address = connData["address"].(string)
	consulapiConfig.Token = d.Token
	client, err := consulapi.NewClient(consulapiConfig)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.KV().Put(&consulapi.KVPair{
		Key:   "foo",
		Value: []byte("bar"),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Operation = logical.RenewOperation
	req.Secret = generatedSecret
	resp, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("got nil response from renew")
	}

	req.Operation = logical.RevokeOperation
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.KV().Put(&consulapi.KVPair{
		Key:   "foo",
		Value: []byte("bar"),
	}, nil)
	if err == nil {
		t.Fatal("err: expected error")
	}
}

func testBackendRenewRevoke14(t *testing.T, version string, policiesParam string) {
	config := logical.TestBackendConfig()
	config.StorageView = &logical.InmemStorage{}
	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}

	cleanup, consulConfig := consul.PrepareTestContainer(t, version, false, true)
	defer cleanup()

	connData := map[string]any{
		"address": consulConfig.Address(),
		"token":   consulConfig.Token,
	}

	req := &logical.Request{
		Storage:   config.StorageView,
		Operation: logical.UpdateOperation,
		Path:      "config/access",
		Data:      connData,
	}
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	req.Path = "roles/test"
	req.Data = map[string]any{
		"lease": "6h",
	}
	if policiesParam == "both" {
		req.Data["policies"] = []string{"wrong-name"}
		req.Data["consul_policies"] = []string{"test"}
	} else {
		req.Data[policiesParam] = []string{"test"}
	}

	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	req.Operation = logical.ReadOperation
	req.Path = "creds/test"
	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("resp nil")
	}
	if resp.IsError() {
		t.Fatalf("resp is error: %v", resp.Error())
	}

	generatedSecret := resp.Secret
	generatedSecret.TTL = 6 * time.Hour

	var d struct {
		Token    string `mapstructure:"token"`
		Accessor string `mapstructure:"accessor"`
	}
	if err := mapstructure.Decode(resp.Data, &d); err != nil {
		t.Fatal(err)
	}

	// Build a client and verify that the credentials work
	consulapiConfig := consulapi.DefaultNonPooledConfig()
	consulapiConfig.Address = connData["address"].(string)
	consulapiConfig.Token = d.Token
	client, err := consulapi.NewClient(consulapiConfig)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Catalog(), nil
	if err != nil {
		t.Fatal(err)
	}

	req.Operation = logical.RenewOperation
	req.Secret = generatedSecret
	resp, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("got nil response from renew")
	}

	req.Operation = logical.RevokeOperation
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	// Build a management client and verify that the token does not exist anymore
	consulmgmtConfig := consulapi.DefaultNonPooledConfig()
	consulmgmtConfig.Address = connData["address"].(string)
	consulmgmtConfig.Token = connData["token"].(string)
	mgmtclient, err := consulapi.NewClient(consulmgmtConfig)
	if err != nil {
		t.Fatal(err)
	}
	q := &consulapi.QueryOptions{
		Datacenter: "DC1",
	}

	_, _, err = mgmtclient.ACL().TokenRead(d.Accessor, q)
	if err == nil {
		t.Fatal("err: expected error")
	}

	t.Run("revoking missing token", func(t *testing.T) {
		// read new token
		req.Operation = logical.ReadOperation
		req.Path = "creds/test"
		resp, err = b.HandleRequest(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if resp == nil {
			t.Fatal("resp nil")
		}
		if resp.IsError() {
			t.Fatalf("resp is error: %v", resp.Error())
		}

		if err := mapstructure.Decode(resp.Data, &d); err != nil {
			t.Fatal(err)
		}

		// Delete token using consul api
		consulapiConfig.Token = consulConfig.Token
		client, err = consulapi.NewClient(consulapiConfig)
		if err != nil {
			t.Fatal(err)
		}

		_, err = client.ACL().TokenDelete(d.Accessor, &consulapi.WriteOptions{})
		if err != nil {
			t.Fatal(err)
		}

		// revoke
		req.Operation = logical.RevokeOperation
		req.Secret = resp.Secret
		_, err = b.HandleRequest(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if resp == nil {
			t.Fatal("resp nil")
		}
		if resp.IsError() {
			t.Fatalf("resp is error: %v", resp.Error())
		}
	})
}

func TestBackend_LocalToken(t *testing.T) {
	config := logical.TestBackendConfig()
	config.StorageView = &logical.InmemStorage{}
	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}

	cleanup, consulConfig := consul.PrepareTestContainer(t, "latest-supported", false, true)
	defer cleanup()

	connData := map[string]any{
		"address": consulConfig.Address(),
		"token":   consulConfig.Token,
	}

	req := &logical.Request{
		Storage:   config.StorageView,
		Operation: logical.UpdateOperation,
		Path:      "config/access",
		Data:      connData,
	}
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	req.Path = "roles/test"
	req.Data = map[string]any{
		"consul_policies": []string{"test"},
		"ttl":             "6h",
		"local":           false,
	}
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	req.Path = "roles/test_local"
	req.Data = map[string]any{
		"consul_policies": []string{"test"},
		"ttl":             "6h",
		"local":           true,
	}
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	req.Operation = logical.ReadOperation
	req.Path = "creds/test"
	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("resp nil")
	}
	if resp.IsError() {
		t.Fatalf("resp is error: %v", resp.Error())
	}

	var d struct {
		Token    string `mapstructure:"token"`
		Accessor string `mapstructure:"accessor"`
		Local    bool   `mapstructure:"local"`
	}
	if err := mapstructure.Decode(resp.Data, &d); err != nil {
		t.Fatal(err)
	}

	if d.Local {
		t.Fatalf("requested global token, got local one")
	}

	// Build a client and verify that the credentials work
	consulapiConfig := consulapi.DefaultNonPooledConfig()
	consulapiConfig.Address = connData["address"].(string)
	consulapiConfig.Token = d.Token
	client, err := consulapi.NewClient(consulapiConfig)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Catalog(), nil
	if err != nil {
		t.Fatal(err)
	}

	req.Operation = logical.ReadOperation
	req.Path = "creds/test_local"
	resp, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("resp nil")
	}
	if resp.IsError() {
		t.Fatalf("resp is error: %v", resp.Error())
	}

	if err := mapstructure.Decode(resp.Data, &d); err != nil {
		t.Fatal(err)
	}

	if !d.Local {
		t.Fatalf("requested local token, got global one")
	}

	// Build a client and verify that the credentials work
	consulapiConfig = consulapi.DefaultNonPooledConfig()
	consulapiConfig.Address = connData["address"].(string)
	consulapiConfig.Token = d.Token
	client, err = consulapi.NewClient(consulapiConfig)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Catalog(), nil
	if err != nil {
		t.Fatal(err)
	}
}

func TestBackend_Basic(t *testing.T) {
	t.Parallel()
	testOldestAndLatestSupported(t, func(t *testing.T, versionId string) {
		t.Parallel()
		testBackendBasic(t, versionId)
	})
}

func testBackendBasic(t *testing.T, version string) {
	config := logical.TestBackendConfig()
	config.StorageView = &logical.InmemStorage{}
	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}

	cleanup, consulConfig := consul.PrepareTestContainer(t, version, false, true)
	defer cleanup()

	connData := map[string]any{
		"address": consulConfig.Address(),
		"token":   consulConfig.Token,
	}

	logicaltest.Test(t, logicaltest.TestCase{
		LogicalBackend: b,
		Steps: []logicaltest.TestStep{
			testAccStepConfig(t, connData),
			testAccStepWriteRole(t, "test", "test", ""),
			testAccStepReadToken(t, "test", connData),
		},
	})
}

func TestBackend_crud(t *testing.T) {
	b, _ := Factory(context.Background(), logical.TestBackendConfig())
	logicaltest.Test(t, logicaltest.TestCase{
		LogicalBackend: b,
		Steps: []logicaltest.TestStep{
			testAccStepWriteRole(t, "test", "write", ""),
			testAccStepWriteRole(t, "test2", "write", ""),
			testAccStepWriteRole(t, "test3", "write", ""),
			testAccStepReadRole(t, "test", "write", 0),
			testAccStepListRole(t, []string{"test", "test2", "test3"}),
			testAccStepDeleteRole(t, "test"),
		},
	})
}

func TestBackend_role_ttl(t *testing.T) {
	b, _ := Factory(context.Background(), logical.TestBackendConfig())
	logicaltest.Test(t, logicaltest.TestCase{
		LogicalBackend: b,
		Steps: []logicaltest.TestStep{
			testAccStepWriteRole(t, "test", "write", "6h"),
			testAccStepReadRole(t, "test", "write", 6*time.Hour),
			testAccStepDeleteRole(t, "test"),
		},
	})
}

func testAccStepConfig(t *testing.T, config map[string]any) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.UpdateOperation,
		Path:      "config/access",
		Data:      config,
	}
}

func testAccStepReadToken(t *testing.T, name string, conf map[string]any) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ReadOperation,
		Path:      "creds/" + name,
		Check: func(resp *logical.Response) error {
			var d struct {
				Token string `mapstructure:"token"`
			}
			if err := mapstructure.Decode(resp.Data, &d); err != nil {
				return err
			}
			log.Printf("[WARN] Generated token: %s", d.Token)

			// Build a client and verify that the credentials work
			config := consulapi.DefaultConfig()
			config.Address = conf["address"].(string)
			config.Token = d.Token
			client, err := consulapi.NewClient(config)
			if err != nil {
				return err
			}

			log.Printf("[WARN] Verifying that the generated token works...")
			_, err = client.KV().Put(&consulapi.KVPair{
				Key:   "foo",
				Value: []byte("bar"),
			}, nil)
			if err != nil {
				return err
			}

			return nil
		},
	}
}

func testAccStepWriteRole(t *testing.T, name string, policy string, ttl string) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.UpdateOperation,
		Path:      "roles/" + name,
		Data: map[string]any{
			"consul_policies": []string{policy},
			"ttl":             ttl,
		},
	}
}

func testAccStepReadRole(t *testing.T, name string, policy string, ttl time.Duration) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ReadOperation,
		Path:      "roles/" + name,
		Check: func(resp *logical.Response) error {
			policies := resp.Data["consul_policies"].([]string)
			if len(policies) != 1 || policies[0] != policy {
				return fmt.Errorf("mismatch: %q %v", policy, policies)
			}

			l := resp.Data["ttl"].(int64)
			if ttl != time.Second*time.Duration(l) {
				return fmt.Errorf("mismatch: %v %v", l, ttl)
			}
			return nil
		},
	}
}

func testAccStepListRole(t *testing.T, names []string) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ListOperation,
		Path:      "roles/",
		Check: func(resp *logical.Response) error {
			respKeys := resp.Data["keys"].([]string)
			if !reflect.DeepEqual(respKeys, names) {
				return fmt.Errorf("mismatch: %#v %#v", respKeys, names)
			}
			return nil
		},
	}
}

func testAccStepDeleteRole(t *testing.T, name string) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.DeleteOperation,
		Path:      "roles/" + name,
	}
}

func TestBackend_Roles(t *testing.T) {
	config := logical.TestBackendConfig()
	config.StorageView = &logical.InmemStorage{}
	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}

	cleanup, consulConfig := consul.PrepareTestContainer(t, "latest-supported", false, true)
	defer cleanup()

	connData := map[string]any{
		"address": consulConfig.Address(),
		"token":   consulConfig.Token,
	}

	req := &logical.Request{
		Storage:   config.StorageView,
		Operation: logical.UpdateOperation,
		Path:      "config/access",
		Data:      connData,
	}
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	// Create the consul_roles role
	req.Path = "roles/test-consul-roles"
	req.Data = map[string]any{
		"consul_roles": []string{"role-test"},
		"ttl":          "6h",
	}
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	req.Operation = logical.ReadOperation
	req.Path = "creds/test-consul-roles"
	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("resp nil")
	}
	if resp.IsError() {
		t.Fatalf("resp is error: %v", resp.Error())
	}

	generatedSecret := resp.Secret
	generatedSecret.TTL = 6 * time.Hour

	var d struct {
		Token    string `mapstructure:"token"`
		Accessor string `mapstructure:"accessor"`
	}
	if err := mapstructure.Decode(resp.Data, &d); err != nil {
		t.Fatal(err)
	}

	// Build a client and verify that the credentials work
	consulapiConfig := consulapi.DefaultNonPooledConfig()
	consulapiConfig.Address = connData["address"].(string)
	consulapiConfig.Token = d.Token
	client, err := consulapi.NewClient(consulapiConfig)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Catalog(), nil
	if err != nil {
		t.Fatal(err)
	}

	req.Operation = logical.RenewOperation
	req.Secret = generatedSecret
	resp, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("got nil response from renew")
	}

	req.Operation = logical.RevokeOperation
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	// Build a management client and verify that the token does not exist anymore
	consulmgmtConfig := consulapi.DefaultNonPooledConfig()
	consulmgmtConfig.Address = connData["address"].(string)
	consulmgmtConfig.Token = connData["token"].(string)
	mgmtclient, err := consulapi.NewClient(consulmgmtConfig)
	if err != nil {
		t.Fatal(err)
	}
	q := &consulapi.QueryOptions{
		Datacenter: "DC1",
	}

	_, _, err = mgmtclient.ACL().TokenRead(d.Accessor, q)
	if err == nil {
		t.Fatal("err: expected error")
	}
}

func TestBackend_Enterprise_Diff_Namespace_Revocation(t *testing.T) {
	if _, hasLicense := os.LookupEnv("CONSUL_LICENSE"); !hasLicense {
		t.Skip("Skipping: No enterprise license found")
	}

	testBackendEntDiffNamespaceRevocation(t)
}

func TestBackend_Enterprise_Diff_Partition_Revocation(t *testing.T) {
	if _, hasLicense := os.LookupEnv("CONSUL_LICENSE"); !hasLicense {
		t.Skip("Skipping: No enterprise license found")
	}

	testBackendEntDiffPartitionRevocation(t)
}

func TestBackend_Enterprise_Namespace(t *testing.T) {
	if _, hasLicense := os.LookupEnv("CONSUL_LICENSE"); !hasLicense {
		t.Skip("Skipping: No enterprise license found")
	}

	testBackendEntNamespace(t)
}

func TestBackend_Enterprise_Partition(t *testing.T) {
	if _, hasLicense := os.LookupEnv("CONSUL_LICENSE"); !hasLicense {
		t.Skip("Skipping: No enterprise license found")
	}

	testBackendEntPartition(t)
}

func testBackendEntDiffNamespaceRevocation(t *testing.T) {
	config := logical.TestBackendConfig()
	config.StorageView = &logical.InmemStorage{}
	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}

	cleanup, consulConfig := consul.PrepareTestContainer(t, "latest-supported", true, true)
	defer cleanup()

	// Perform additional Consul configuration
	consulapiConfig := consulapi.DefaultNonPooledConfig()
	consulapiConfig.Address = consulConfig.Address()
	consulapiConfig.Token = consulConfig.Token
	client, err := consulapi.NewClient(consulapiConfig)
	if err != nil {
		t.Fatal(err)
	}

	// Create Policy in default namespace to manage ACLs in a different
	// namespace
	nsPol := &consulapi.ACLPolicy{
		Name:        "diff-ns-test",
		Description: "policy to test management of ACLs in one ns from another",
		Rules: `namespace "ns1" {
			acl="write"
		}
		`,
	}
	pol, _, err := client.ACL().PolicyCreate(nsPol, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create new Token in default namespace with new ACL
	cToken, _, err := client.ACL().TokenCreate(
		&consulapi.ACLToken{
			Policies: []*consulapi.ACLLink{{ID: pol.ID}},
		}, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Write backend config
	connData := map[string]any{
		"address": consulConfig.Address(),
		"token":   cToken.SecretID,
	}

	req := &logical.Request{
		Storage:   config.StorageView,
		Operation: logical.UpdateOperation,
		Path:      "config/access",
		Data:      connData,
	}
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	// Create the role in namespace "ns1"
	req.Path = "roles/test-ns"
	req.Data = map[string]any{
		"consul_policies":  []string{"ns-test"},
		"ttl":              "6h",
		"consul_namespace": "ns1",
	}
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	// Get Token
	req.Operation = logical.ReadOperation
	req.Path = "creds/test-ns"
	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("resp nil")
	}
	if resp.IsError() {
		t.Fatalf("resp is error: %v", resp.Error())
	}

	generatedSecret := resp.Secret
	generatedSecret.TTL = 6 * time.Hour

	// Verify Secret
	var d struct {
		Token           string `mapstructure:"token"`
		Accessor        string `mapstructure:"accessor"`
		ConsulNamespace string `mapstructure:"consul_namespace"`
	}
	if err := mapstructure.Decode(resp.Data, &d); err != nil {
		t.Fatal(err)
	}

	if d.ConsulNamespace != "ns1" {
		t.Fatalf("Failed to access namespace")
	}

	// Revoke the credential
	req.Operation = logical.RevokeOperation
	req.Secret = generatedSecret
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("Revocation failed: %v", err)
	}

	// Build a management client and verify that the token does not exist anymore
	consulmgmtConfig := consulapi.DefaultNonPooledConfig()
	consulmgmtConfig.Address = connData["address"].(string)
	consulmgmtConfig.Token = connData["token"].(string)
	mgmtclient, err := consulapi.NewClient(consulmgmtConfig)
	if err != nil {
		t.Fatal(err)
	}
	q := &consulapi.QueryOptions{
		Datacenter: "DC1",
		Namespace:  "ns1",
	}

	_, _, err = mgmtclient.ACL().TokenRead(d.Accessor, q)
	if err == nil {
		t.Fatal("err: expected error")
	}
}

func testBackendEntDiffPartitionRevocation(t *testing.T) {
	config := logical.TestBackendConfig()
	config.StorageView = &logical.InmemStorage{}
	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}

	cleanup, consulConfig := consul.PrepareTestContainer(t, "latest-supported", true, true)
	defer cleanup()

	// Perform additional Consul configuration
	consulapiConfig := consulapi.DefaultNonPooledConfig()
	consulapiConfig.Address = consulConfig.Address()
	consulapiConfig.Token = consulConfig.Token
	client, err := consulapi.NewClient(consulapiConfig)
	if err != nil {
		t.Fatal(err)
	}

	// Create Policy in default partition to manage ACLs in a different
	// partition
	partPol := &consulapi.ACLPolicy{
		Name:        "diff-part-test",
		Description: "policy to test management of ACLs in one part from another",
		Rules: `partition "part1" {
			acl="write"
		}
		`,
	}
	pol, _, err := client.ACL().PolicyCreate(partPol, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create new Token in default partition with new ACL
	cToken, _, err := client.ACL().TokenCreate(
		&consulapi.ACLToken{
			Policies: []*consulapi.ACLLink{{ID: pol.ID}},
		}, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Write backend config
	connData := map[string]any{
		"address": consulConfig.Address(),
		"token":   cToken.SecretID,
	}

	req := &logical.Request{
		Storage:   config.StorageView,
		Operation: logical.UpdateOperation,
		Path:      "config/access",
		Data:      connData,
	}
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	// Create the role in partition "part1"
	req.Path = "roles/test-part"
	req.Data = map[string]any{
		"consul_policies": []string{"part-test"},
		"ttl":             "6h",
		"partition":       "part1",
	}
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	// Get Token
	req.Operation = logical.ReadOperation
	req.Path = "creds/test-part"
	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("resp nil")
	}
	if resp.IsError() {
		t.Fatalf("resp is error: %v", resp.Error())
	}

	generatedSecret := resp.Secret
	generatedSecret.TTL = 6 * time.Hour

	// Verify Secret
	var d struct {
		Token     string `mapstructure:"token"`
		Accessor  string `mapstructure:"accessor"`
		Partition string `mapstructure:"partition"`
	}
	if err := mapstructure.Decode(resp.Data, &d); err != nil {
		t.Fatal(err)
	}

	if d.Partition != "part1" {
		t.Fatalf("Failed to access partition")
	}

	// Revoke the credential
	req.Operation = logical.RevokeOperation
	req.Secret = generatedSecret
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("Revocation failed: %v", err)
	}

	// Build a management client and verify that the token does not exist anymore
	consulmgmtConfig := consulapi.DefaultNonPooledConfig()
	consulmgmtConfig.Address = connData["address"].(string)
	consulmgmtConfig.Token = connData["token"].(string)
	mgmtclient, err := consulapi.NewClient(consulmgmtConfig)
	if err != nil {
		t.Fatal(err)
	}
	q := &consulapi.QueryOptions{
		Datacenter: "DC1",
		Partition:  "part1",
	}

	_, _, err = mgmtclient.ACL().TokenRead(d.Accessor, q)
	if err == nil {
		t.Fatal("err: expected error")
	}
}

func testBackendEntNamespace(t *testing.T) {
	config := logical.TestBackendConfig()
	config.StorageView = &logical.InmemStorage{}
	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}

	cleanup, consulConfig := consul.PrepareTestContainer(t, "latest-supported", true, true)
	defer cleanup()

	connData := map[string]any{
		"address": consulConfig.Address(),
		"token":   consulConfig.Token,
	}

	req := &logical.Request{
		Storage:   config.StorageView,
		Operation: logical.UpdateOperation,
		Path:      "config/access",
		Data:      connData,
	}
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	// Create the role in namespace "ns1"
	req.Path = "roles/test-ns"
	req.Data = map[string]any{
		"consul_policies":  []string{"ns-test"},
		"ttl":              "6h",
		"consul_namespace": "ns1",
	}
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	req.Operation = logical.ReadOperation
	req.Path = "creds/test-ns"
	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("resp nil")
	}
	if resp.IsError() {
		t.Fatalf("resp is error: %v", resp.Error())
	}

	generatedSecret := resp.Secret
	generatedSecret.TTL = 6 * time.Hour

	var d struct {
		Token           string `mapstructure:"token"`
		Accessor        string `mapstructure:"accessor"`
		ConsulNamespace string `mapstructure:"consul_namespace"`
	}
	if err := mapstructure.Decode(resp.Data, &d); err != nil {
		t.Fatal(err)
	}

	if d.ConsulNamespace != "ns1" {
		t.Fatalf("Failed to access namespace")
	}

	// Build a client and verify that the credentials work
	consulapiConfig := consulapi.DefaultNonPooledConfig()
	consulapiConfig.Address = connData["address"].(string)
	consulapiConfig.Token = d.Token
	client, err := consulapi.NewClient(consulapiConfig)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Catalog(), nil
	if err != nil {
		t.Fatal(err)
	}

	req.Operation = logical.RenewOperation
	req.Secret = generatedSecret
	resp, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("got nil response from renew")
	}

	req.Operation = logical.RevokeOperation
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	// Build a management client and verify that the token does not exist anymore
	consulmgmtConfig := consulapi.DefaultNonPooledConfig()
	consulmgmtConfig.Address = connData["address"].(string)
	consulmgmtConfig.Token = connData["token"].(string)
	mgmtclient, err := consulapi.NewClient(consulmgmtConfig)
	if err != nil {
		t.Fatal(err)
	}
	q := &consulapi.QueryOptions{
		Datacenter: "DC1",
		Namespace:  "ns1",
	}

	_, _, err = mgmtclient.ACL().TokenRead(d.Accessor, q)
	if err == nil {
		t.Fatal("err: expected error")
	}
}

func testBackendEntPartition(t *testing.T) {
	config := logical.TestBackendConfig()
	config.StorageView = &logical.InmemStorage{}
	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}

	cleanup, consulConfig := consul.PrepareTestContainer(t, "latest-supported", true, true)
	defer cleanup()

	connData := map[string]any{
		"address": consulConfig.Address(),
		"token":   consulConfig.Token,
	}

	req := &logical.Request{
		Storage:   config.StorageView,
		Operation: logical.UpdateOperation,
		Path:      "config/access",
		Data:      connData,
	}
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	// Create the role in partition "part1"
	req.Path = "roles/test-part"
	req.Data = map[string]any{
		"consul_policies": []string{"part-test"},
		"ttl":             "6h",
		"partition":       "part1",
	}
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	req.Operation = logical.ReadOperation
	req.Path = "creds/test-part"
	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("resp nil")
	}
	if resp.IsError() {
		t.Fatalf("resp is error: %v", resp.Error())
	}

	generatedSecret := resp.Secret
	generatedSecret.TTL = 6 * time.Hour

	var d struct {
		Token     string `mapstructure:"token"`
		Accessor  string `mapstructure:"accessor"`
		Partition string `mapstructure:"partition"`
	}
	if err := mapstructure.Decode(resp.Data, &d); err != nil {
		t.Fatal(err)
	}

	if d.Partition != "part1" {
		t.Fatalf("Failed to access partition")
	}

	// Build a client and verify that the credentials work
	consulapiConfig := consulapi.DefaultNonPooledConfig()
	consulapiConfig.Address = connData["address"].(string)
	consulapiConfig.Token = d.Token
	client, err := consulapi.NewClient(consulapiConfig)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Catalog(), nil
	if err != nil {
		t.Fatal(err)
	}

	req.Operation = logical.RenewOperation
	req.Secret = generatedSecret
	resp, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("got nil response from renew")
	}

	req.Operation = logical.RevokeOperation
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	// Build a management client and verify that the token does not exist anymore
	consulmgmtConfig := consulapi.DefaultNonPooledConfig()
	consulmgmtConfig.Address = connData["address"].(string)
	consulmgmtConfig.Token = connData["token"].(string)
	mgmtclient, err := consulapi.NewClient(consulmgmtConfig)
	if err != nil {
		t.Fatal(err)
	}
	q := &consulapi.QueryOptions{
		Datacenter: "DC1",
		Partition:  "test1",
	}

	_, _, err = mgmtclient.ACL().TokenRead(d.Accessor, q)
	if err == nil {
		t.Fatal("err: expected error")
	}
}

func TestBackendRenewRevokeRolesAndIdentities(t *testing.T) {
	config := logical.TestBackendConfig()
	config.StorageView = &logical.InmemStorage{}
	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}

	cleanup, consulConfig := consul.PrepareTestContainer(t, "latest-supported", false, true)
	defer cleanup()

	connData := map[string]any{
		"address": consulConfig.Address(),
		"token":   consulConfig.Token,
	}

	req := &logical.Request{
		Storage:   config.StorageView,
		Operation: logical.UpdateOperation,
		Path:      "config/access",
		Data:      connData,
	}
	_, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string]struct {
		RoleName string
		RoleData map[string]any
	}{
		"just role": {
			"r",
			map[string]any{
				"consul_roles": []string{"role-test"},
				"ttl":          "6h",
			},
		},
		"role and policies": {
			"rp",
			map[string]any{
				"consul_policies": []string{"test"},
				"consul_roles":    []string{"role-test"},
				"ttl":             "6h",
			},
		},
		"service identity": {
			"si",
			map[string]any{
				"service_identities": "service1",
				"ttl":                "6h",
			},
		},
		"service identity and policies": {
			"sip",
			map[string]any{
				"consul_policies":    []string{"test"},
				"service_identities": "service1",
				"ttl":                "6h",
			},
		},
		"service identity and role": {
			"sir",
			map[string]any{
				"consul_roles":       []string{"role-test"},
				"service_identities": "service1",
				"ttl":                "6h",
			},
		},
		"service identity and role and policies": {
			"sirp",
			map[string]any{
				"consul_policies":    []string{"test"},
				"consul_roles":       []string{"role-test"},
				"service_identities": "service1",
				"ttl":                "6h",
			},
		},
		"node identity": {
			"ni",
			map[string]any{
				"node_identities": []string{"node1:dc1"},
				"ttl":             "6h",
			},
		},
		"node identity and policies": {
			"nip",
			map[string]any{
				"consul_policies": []string{"test"},
				"node_identities": []string{"node1:dc1"},
				"ttl":             "6h",
			},
		},
		"node identity and role": {
			"nir",
			map[string]any{
				"consul_roles":    []string{"role-test"},
				"node_identities": []string{"node1:dc1"},
				"ttl":             "6h",
			},
		},
		"node identity and role and policies": {
			"nirp",
			map[string]any{
				"consul_policies": []string{"test"},
				"consul_roles":    []string{"role-test"},
				"node_identities": []string{"node1:dc1"},
				"ttl":             "6h",
			},
		},
		"node identity and service identity": {
			"nisi",
			map[string]any{
				"service_identities": "service1",
				"node_identities":    []string{"node1:dc1"},
				"ttl":                "6h",
			},
		},
		"node identity and service identity and policies": {
			"nisip",
			map[string]any{
				"consul_policies":    []string{"test"},
				"service_identities": "service1",
				"node_identities":    []string{"node1:dc1"},
				"ttl":                "6h",
			},
		},
		"node identity and service identity and role": {
			"nisir",
			map[string]any{
				"consul_roles":       []string{"role-test"},
				"service_identities": "service1",
				"node_identities":    []string{"node1:dc1"},
				"ttl":                "6h",
			},
		},
		"node identity and service identity and role and policies": {
			"nisirp",
			map[string]any{
				"consul_policies":    []string{"test"},
				"consul_roles":       []string{"role-test"},
				"service_identities": "service1",
				"node_identities":    []string{"node1:dc1"},
				"ttl":                "6h",
			},
		},
	}

	for description, tc := range cases {
		t.Logf("Testing: %s", description)

		req.Operation = logical.UpdateOperation
		req.Path = fmt.Sprintf("roles/%s", tc.RoleName)
		req.Data = tc.RoleData
		_, err = b.HandleRequest(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}

		req.Operation = logical.ReadOperation
		req.Path = fmt.Sprintf("creds/%s", tc.RoleName)
		resp, err := b.HandleRequest(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if resp == nil {
			t.Fatal("resp nil")
		}
		if resp.IsError() {
			t.Fatalf("resp is error: %v", resp.Error())
		}

		generatedSecret := resp.Secret
		generatedSecret.TTL = 6 * time.Hour

		var d struct {
			Token    string `mapstructure:"token"`
			Accessor string `mapstructure:"accessor"`
		}
		if err := mapstructure.Decode(resp.Data, &d); err != nil {
			t.Fatal(err)
		}

		// Build a client and verify that the credentials work
		consulapiConfig := consulapi.DefaultNonPooledConfig()
		consulapiConfig.Address = connData["address"].(string)
		consulapiConfig.Token = d.Token
		client, err := consulapi.NewClient(consulapiConfig)
		if err != nil {
			t.Fatal(err)
		}

		_, err = client.Catalog(), nil
		if err != nil {
			t.Fatal(err)
		}

		req.Operation = logical.RenewOperation
		req.Secret = generatedSecret
		resp, err = b.HandleRequest(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if resp == nil {
			t.Fatal("got nil response from renew")
		}

		req.Operation = logical.RevokeOperation
		_, err = b.HandleRequest(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}

		// Build a management client and verify that the token does not exist anymore
		consulmgmtConfig := consulapi.DefaultNonPooledConfig()
		consulmgmtConfig.Address = connData["address"].(string)
		consulmgmtConfig.Token = connData["token"].(string)
		mgmtclient, err := consulapi.NewClient(consulmgmtConfig)
		if err != nil {
			t.Fatal(err)
		}

		q := &consulapi.QueryOptions{
			Datacenter: "DC1",
		}

		_, _, err = mgmtclient.ACL().TokenRead(d.Accessor, q)
		if err == nil {
			t.Fatal("err: expected error")
		}
	}
}
