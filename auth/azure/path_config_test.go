// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package azureauth

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
)

func TestConfig_Write(t *testing.T) {
	b, s := getTestBackend(t)

	configData := map[string]interface{}{}
	if err := testConfigCreate(t, b, s, configData); err == nil {
		t.Fatal("expected error")
	}

	configData = map[string]interface{}{
		"tenant_id": "tid",
		"resource":  "resource",
	}
	if err := testConfigCreate(t, b, s, configData); err != nil {
		t.Fatalf("err: %v", err)
	}

	configData["environment"] = "AzureNotRealCloud"
	if err := testConfigCreate(t, b, s, configData); err == nil {
		t.Fatal("expected error")
	}

	configData["environment"] = "AzurePublicCloud"
	if err := testConfigCreate(t, b, s, configData); err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestConfigDelete(t *testing.T) {
	b, s := getTestBackend(t)

	configData := map[string]interface{}{
		"tenant_id": "tid",
		"resource":  "resource",
	}
	if err := testConfigCreate(t, b, s, configData); err != nil {
		t.Fatalf("err: %v", err)
	}

	_, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.DeleteOperation,
		Path:      "config",
		Storage:   s,
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "config",
		Storage:   s,
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp != nil {
		t.Fatal("expected nil config after delete")
	}
}

func testConfigCreate(t *testing.T, b *azureAuthBackend, s logical.Storage, d map[string]interface{}) error {
	t.Helper()
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "config",
		Data:      d,
		Storage:   s,
	})
	if err != nil {
		return err
	}
	if resp != nil && resp.IsError() {
		return resp.Error()
	}
	return nil
}

func testConfigRead(t *testing.T, b *azureAuthBackend, s logical.Storage) (*logical.Response, error) {
	t.Helper()
	return b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "config",
		Storage:   s,
	})
}

func TestConfig_RetryDefaults(t *testing.T) {
	b, s := getTestBackend(t)

	configData := map[string]interface{}{
		"tenant_id": "tid",
		"resource":  "resource",
	}

	if err := testConfigCreate(t, b, s, configData); err != nil {
		t.Fatalf("err: %v", err)
	}

	resp, err := testConfigRead(t, b, s)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if resp.Data["max_retries"] != defaultMaxRetries {
		t.Fatalf("wrong max_retries default: expected %v, got %v", defaultMaxRetries, resp.Data["max_retries"])
	}

	if resp.Data["max_retry_delay"] != defaultMaxRetryDelay {
		t.Fatalf("wrong 'max_retry_delay' default: expected %v, got %v", defaultMaxRetryDelay, resp.Data["max_retry_delay"])
	}

	if resp.Data["retry_delay"] != defaultRetryDelay {
		t.Fatalf("wrong 'retry_delay' default: expected %v, got %v", defaultRetryDelay, resp.Data["retry_delay"])
	}

	config, err := b.config(context.Background(), s)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	azureSettings, err := b.getAzureSettings(context.Background(), config)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if azureSettings.MaxRetries != defaultMaxRetries {
		t.Fatalf("wrong 'max_retries' default azure settings value: expected %v, got %v", defaultMaxRetries, azureSettings.MaxRetries)
	}

	if azureSettings.MaxRetryDelay != defaultMaxRetryDelay {
		t.Fatalf("wrong 'max_retry_delay' default azure settings value: expected %v, got %v", defaultMaxRetryDelay, azureSettings.MaxRetryDelay)
	}

	if azureSettings.RetryDelay != defaultRetryDelay {
		t.Fatalf("wrong 'retry_delay' default azure settings value: expected %v, got %v", defaultRetryDelay, azureSettings.RetryDelay)
	}
}

func TestConfig_RetryCustom(t *testing.T) {
	b, s := getTestBackend(t)
	maxRetries := int32(60)
	maxRetryDelay := time.Second * 120
	retryDelay := time.Second * 10

	configData := map[string]interface{}{
		"tenant_id":       "tid",
		"resource":        "resource",
		"max_retries":     maxRetries,
		"max_retry_delay": maxRetryDelay,
		"retry_delay":     retryDelay,
	}

	if err := testConfigCreate(t, b, s, configData); err != nil {
		t.Fatalf("err: %v", err)
	}

	resp, err := testConfigRead(t, b, s)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if resp.Data["max_retries"] != maxRetries {
		t.Fatalf("wrong max_retries value: expected %v, got %v", maxRetries, resp.Data["max_retries"])
	}

	if resp.Data["max_retry_delay"] != maxRetryDelay {
		t.Fatalf("wrong 'max_retry_delay' value: expected %v, got %v", maxRetryDelay, resp.Data["max_retry_delay"])
	}

	if resp.Data["retry_delay"] != retryDelay {
		t.Fatalf("wrong 'retry_delay' value: expected %v, got %v", retryDelay, resp.Data["retry_delay"])
	}

	config, err := b.config(context.Background(), s)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	azureSettings, err := b.getAzureSettings(context.Background(), config)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if azureSettings.MaxRetries != maxRetries {
		t.Fatalf("wrong 'max_retries' azure settings value: expected %v, got %v", maxRetries, azureSettings.MaxRetries)
	}

	if azureSettings.MaxRetryDelay != maxRetryDelay {
		t.Fatalf("wrong 'max_retry_delay' azure settings value: expected %v, got %v", maxRetryDelay, azureSettings.MaxRetryDelay)
	}

	if azureSettings.RetryDelay != retryDelay {
		t.Fatalf("wrong 'retry_delay' azure settings value: expected %v, got %v", retryDelay, azureSettings.RetryDelay)
	}
}
