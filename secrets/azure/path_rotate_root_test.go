// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package azure

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/openbao/openbao/sdk/v2/logical"
)

func TestRotateRootSuccess(t *testing.T) {
	b, s := getTestBackend(t)

	skipIfMissingEnvVars(t,
		"AZURE_CLIENT_ID",
		"AZURE_CLIENT_SECRET",
		"AZURE_TENANT_ID",
	)

	configData := map[string]interface{}{
		"tenant_id":     os.Getenv("AZURE_TENANT_ID"),
		"client_id":     os.Getenv("AZURE_CLIENT_ID"),
		"client_secret": os.Getenv("AZURE_CLIENT_SECRET"),
	}
	testConfigCreate(t, b, s, configData, false)

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

	config, err := b.getConfig(context.Background(), s)
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

	newConfig, err := b.getConfig(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}

	if newConfig.ClientSecret != config.NewClientSecret {
		t.Fatal(fmt.Errorf("old and new password aren't equal after periodic function, they should be"))
	}
}

func TestRotateRootPeriodicFunctionBeforeMinute(t *testing.T) {
	b, s := getTestBackend(t)

	skipIfMissingEnvVars(t,
		"AZURE_CLIENT_ID",
		"AZURE_CLIENT_SECRET",
		"AZURE_TENANT_ID",
	)

	configData := map[string]interface{}{
		"tenant_id":     os.Getenv("AZURE_TENANT_ID"),
		"client_id":     os.Getenv("AZURE_CLIENT_ID"),
		"client_secret": os.Getenv("AZURE_CLIENT_SECRET"),
	}
	testConfigCreate(t, b, s, configData, false)

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
		config, err := b.getConfig(context.Background(), s)
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

		newConfig, err := b.getConfig(context.Background(), s)
		if err != nil {
			t.Fatal(test.Name, err)
		}

		if newConfig.ClientSecret == config.NewClientSecret {
			t.Fatal(test.Name, fmt.Errorf("old and new password are equal after periodic function, they shouldn't be"))
		}
	}
}

func assertNotNil(t *testing.T, val interface{}) {
	t.Helper()
	if val == nil {
		t.Fatalf("expected not nil, but was nil")
	}
}

func assertNotEmptyString(t *testing.T, str string) {
	t.Helper()
	if str == "" {
		t.Fatalf("string is empty")
	}
}

func assertStrSliceIsNotEmpty(t *testing.T, strs []string) {
	t.Helper()
	if strs == nil || len(strs) == 0 {
		t.Fatalf("string slice is empty")
	}
}
