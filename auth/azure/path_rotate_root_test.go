// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package azureauth

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
)

// This test performs a rotate root operation and
// swaps the root credentials for the application
func TestRotateRootSuccess(t *testing.T) {
	b, s := getTestBackend(t)

	tenantID, clientID, clientSecret := getAzureEnvironmentSettings()
	if tenantID == "" || clientID == "" || clientSecret == "" {
		t.Skip("environment variables not set, skipping test")
	}

	configData := map[string]interface{}{
		"tenant_id":     tenantID,
		"resource":      "https://management.azure.com/",
		"client_id":     clientID,
		"environment":   "azurepublicCloud",
		"client_secret": clientSecret,
	}
	if err := testConfigCreate(t, b, s, configData); err != nil {
		t.Fatalf("err: %v", err)
	}

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "rotate-root",
		Data:      map[string]interface{}{},
		Storage:   s,
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp != nil && resp.IsError() {
		t.Fatal(resp.Error())
	}

	config, err := b.config(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}

	if config.ClientSecret == "" {
		t.Fatal(fmt.Errorf("root password was empty after rotate root, it shouldn't be"))
	}

	if config.NewClientSecret == config.ClientSecret {
		t.Fatal("old and new password equal after rotate-root, it shouldn't be")
	}

	if config.NewClientSecret == "" {
		t.Fatal("new password is empty, it shouldn't be")
	}

	if config.NewClientSecretKeyID == "" {
		t.Fatal("new password key id is empty, it shouldn't be")
	}

	if !b.updatePassword {
		t.Fatal("update password is false, it shouldn't be")
	}

	config.NewClientSecretCreated = config.NewClientSecretCreated.Add(-(time.Minute * 1))
	err = b.saveConfig(context.Background(), config, s)
	if err != nil {
		t.Fatal(err)
	}

	err = b.periodicFunc(context.Background(), &logical.Request{
		Storage: s,
	})

	if err != nil {
		t.Fatal(err)
	}

	newConfig, err := b.config(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}

	if newConfig.ClientSecret != config.NewClientSecret {
		t.Fatal(fmt.Errorf("old and new password aren't equal after periodic function, they should be"))
	}
}

// This test verifies that the periodicFunc does not remove
// stale credentials until the value for NewClientSecretCreated
// is greater than 1 minute
func TestRotateRootPeriodicFunctionBeforeMinute(t *testing.T) {
	b, s := getTestBackend(t)

	tenantID, clientID, clientSecret := getAzureEnvironmentSettings()
	if tenantID == "" ||
		clientID == "" || clientSecret == "" {
		t.Skip("environment variables not set, skipping test")
	}

	configData := map[string]interface{}{
		"tenant_id":     tenantID,
		"resource":      "https://management.azure.com/",
		"client_id":     clientID,
		"client_secret": clientSecret,
	}
	if err := testConfigCreate(t, b, s, configData); err != nil {
		t.Fatalf("err: %v", err)
	}

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "rotate-root",
		Data:      map[string]interface{}{},
		Storage:   s,
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp != nil && resp.IsError() {
		t.Fatal(resp.Error())
	}

	tests := []struct {
		Name    string
		Created time.Duration
	}{
		{
			Name:    "1 second test:",
			Created: time.Second * 1,
		},
		{
			Name:    "5 seconds test:",
			Created: time.Second * 5,
		},
		{
			Name:    "30 seconds test:",
			Created: time.Second * 30,
		},
		{
			Name:    "50 seconds test:",
			Created: time.Second * 50,
		},
	}

	for _, test := range tests {
		t.Log(test.Name)
		config, err := b.config(context.Background(), s)
		if err != nil {
			t.Fatal(err)
		}

		config.NewClientSecretCreated = time.Now().Add(-(test.Created))
		err = b.saveConfig(context.Background(), config, s)
		if err != nil {
			t.Fatal(test.Name, err)
		}

		err = b.periodicFunc(context.Background(), &logical.Request{
			Storage: s,
		})

		if err != nil {
			t.Fatal(test.Name, err)
		}

		newConfig, err := b.config(context.Background(), s)
		if err != nil {
			t.Fatal(test.Name, err)
		}

		if newConfig.ClientSecret == config.NewClientSecret {
			t.Fatal(test.Name, fmt.Errorf("old and new password are equal after periodic function, they shouldn't be"))
		}
	}
}

func getAzureEnvironmentSettings() (string, string, string) {
	tenantID := os.Getenv("TENANT_ID")
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")

	return tenantID, clientID, clientSecret
}
