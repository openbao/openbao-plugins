// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package azure

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-uuid"
	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
)

func pathRotateRoot(b *azureSecretBackend) *framework.Path {
	return &framework.Path{
		Pattern: "rotate-root",
		DisplayAttrs: &framework.DisplayAttributes{
			OperationPrefix: operationPrefixAzure,
			OperationVerb:   "rotate",
			OperationSuffix: "root",
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback:                    b.pathRotateRoot,
				ForwardPerformanceSecondary: true,
				ForwardPerformanceStandby:   true,
			},
		},

		HelpSynopsis: "Attempt to rotate the root credentials used to communicate with Azure.",
		HelpDescription: "This path will attempt to generate new root credentials for the user used to access and manipulate Azure.\n" +
			"The new credentials will not be returned from this endpoint, nor the read config endpoint.",
	}
}

func (b *azureSecretBackend) pathRotateRoot(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	err := b.rotateRootCredential(ctx, req)

	// this replicates the old case where a wal error returned the multierror and a blank response instead of nil.
	var error *multierror.Error
	if errors.As(err, &error) {
		return &logical.Response{}, err
	}

	return nil, err
}

func (b *azureSecretBackend) rotateRootCredential(ctx context.Context, req *logical.Request) error {

	config, err := b.getConfig(ctx, req.Storage)
	if err != nil {
		return err
	}

	if config == nil {
		return fmt.Errorf("config is nil")
	}

	expDur := config.RootPasswordTTL
	if expDur == 0 {
		expDur = defaultRootPasswordTTL
	}
	expiration := time.Now().Add(expDur)

	client, err := b.getClient(ctx, req.Storage)
	if err != nil {
		return err
	}

	// We need to use List instead of Get here because we don't have the Object ID
	// (which is different from the Application/Client ID)
	apps, err := client.provider.ListApplications(ctx, fmt.Sprintf("appId eq '%s'", config.ClientID))
	if err != nil {
		return err
	}

	if len(apps) == 0 {
		return fmt.Errorf("no application found")
	}
	if len(apps) > 1 {
		return fmt.Errorf("multiple applications found - double check your client_id")
	}

	app := apps[0]

	uniqueID, err := uuid.GenerateUUID()
	if err != nil {
		return fmt.Errorf("failed to generate UUID: %w", err)
	}

	// This could have the same username customization logic put on it if we really wanted it here
	passwordDisplayName := fmt.Sprintf("vault-%s", uniqueID)
	newPasswordResp, err := client.provider.AddApplicationPassword(ctx, app.AppObjectID, passwordDisplayName, expiration)
	if err != nil {
		return fmt.Errorf("failed to add new password: %w", err)
	}

	var wal walRotateRoot
	walID, walErr := framework.PutWAL(ctx, req.Storage, walRotateRootCreds, wal)
	if walErr != nil {
		err = client.provider.RemoveApplicationPassword(ctx, app.AppObjectID, newPasswordResp.KeyID)
		merr := multierror.Append(err, err)
		return merr
	}

	config.NewClientSecret = newPasswordResp.SecretText
	config.NewClientSecretCreated = time.Now()
	config.NewClientSecretExpirationDate = newPasswordResp.EndDate
	config.NewClientSecretKeyID = newPasswordResp.KeyID

	err = b.saveConfig(ctx, config, req.Storage)
	if err != nil {
		return fmt.Errorf("failed to save new configuration: %w", err)
	}

	b.updatePassword = true

	err = framework.DeleteWAL(ctx, req.Storage, walID)
	if err != nil {
		b.Logger().Error("rotate root", "delete wal", err)
	}

	return err
}

type passwordRemover interface {
	RemoveApplicationPassword(ctx context.Context, applicationObjectID string, keyID string) error
}

func removeApplicationPasswords(ctx context.Context, passRemover passwordRemover, appID string, passwordKeyIDs ...string) (err error) {
	merr := new(multierror.Error)
	for _, keyID := range passwordKeyIDs {
		// Attempt to remove all of them, don't fail early
		err := passRemover.RemoveApplicationPassword(ctx, appID, keyID)
		if err != nil {
			merr = multierror.Append(merr, err)
		}
	}

	return merr.ErrorOrNil()
}
