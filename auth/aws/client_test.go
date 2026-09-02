// Copyright (c) 2025 OpenBao a Series of LF Projects, LLC
// SPDX-License-Identifier: MPL-2.0

package aws

import (
	"context"
	"fmt"
	"testing"

	"github.com/openbao/openbao/sdk/v2/logical"
)

// TestClientCache verifies that IAM clients for different
// AWS accounts are properly isolated in the cache
func TestClientCache(t *testing.T) {
	config := logical.TestBackendConfig()
	storage := &logical.InmemStorage{}
	config.StorageView = storage

	b, err := Backend(config)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := b.Setup(ctx, config); err != nil {
		t.Fatal(err)
	}

	account1 := "111111111111"
	account2 := "222222222222"

	b.defaultAWSAccountID = account1

	// This should work - same account as default
	stsRole, err := b.stsRoleForAccount(ctx, storage, account1)
	if err != nil {
		t.Fatalf("Expected success for default account, got error: %v", err)
	}
	if stsRole != "" {
		t.Fatalf("Expected empty STS role for default account, got: %v", stsRole)
	}

	// This should fail - different account without STS config
	_, err = b.stsRoleForAccount(ctx, storage, account2)
	if err == nil {
		t.Fatal("Expected error for cross-account access without STS config")
	}

	// Verify the error message contains the expected error
	expectedError := fmt.Sprintf("no STS configuration found for account ID %q", account2)
	if err.Error() != expectedError {
		t.Fatalf("Expected specific error message, got: %v", err)
	}

	stsEntry := &awsStsEntry{
		StsRole: "arn:aws:iam::222222222222:role/cross-account-role",
	}
	err = b.lockedSetAwsStsEntry(ctx, storage, account2, stsEntry)
	if err != nil {
		t.Fatalf("Failed to set STS entry: %v", err)
	}

	stsRole, err = b.stsRoleForAccount(ctx, storage, account2)
	if err != nil {
		t.Fatalf("Expected success for account with STS config, got error: %v", err)
	}
	if stsRole != stsEntry.StsRole {
		t.Fatalf("Expected STS role %v, got: %v", stsEntry.StsRole, stsRole)
	}
}
