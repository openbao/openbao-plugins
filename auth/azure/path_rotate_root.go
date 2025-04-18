// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package azure

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"

	"github.com/openbao/openbao-plugins/auth/azure/client"
)

func pathRotateRoot(b *azureAuthBackend) *framework.Path {
	return &framework.Path{
		Pattern: "rotate-root",
		DisplayAttrs: &framework.DisplayAttributes{
			OperationPrefix: operationPrefixAzure,
			OperationVerb:   "rotate",
			OperationSuffix: "root-credentials",
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

func (b *azureAuthBackend) pathRotateRoot(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	err := b.rotateRootCredential(ctx, req)
	var error *multierror.Error
	if errors.As(err, &error) {
		return &logical.Response{}, err // match multierror return from old rotate call
	}

	return nil, err
}

func (b *azureAuthBackend) rotateRootCredential(ctx context.Context, req *logical.Request) error {
	config, err := b.config(ctx, req.Storage)
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

	provider, err := b.getProvider(ctx, config)
	if err != nil {
		return err
	}

	client, err := provider.MSGraphClient()
	if err != nil {
		return err
	}

	app, err := client.GetApplication(ctx, config.ClientID)
	if err != nil {
		return err
	}

	// This could have the same username customization logic put on it if we really wanted it here
	passwordDisplayName := fmt.Sprintf("vault-%s", uuid.New())
	newPasswordResp, err := client.AddApplicationPassword(ctx, *app.GetId(), passwordDisplayName, expiration)
	if err != nil {
		return fmt.Errorf("failed to add new password: %w", err)
	}

	var wal walRotateRoot
	walID, walErr := framework.PutWAL(ctx, req.Storage, walRotateRootCreds, wal)
	if walErr != nil {
		err = client.RemoveApplicationPassword(ctx, *app.GetId(), newPasswordResp.GetKeyId())
		merr := multierror.Append(err, err)
		return merr
	}

	config.NewClientSecret = *newPasswordResp.GetSecretText()
	config.NewClientSecretCreated = time.Now()
	config.NewClientSecretExpirationDate = *newPasswordResp.GetEndDateTime()
	config.NewClientSecretKeyID = newPasswordResp.GetKeyId().String()

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

func removeApplicationPasswords(ctx context.Context, c client.MSGraphClient, appID string, passwordKeyIDs ...*uuid.UUID) (err error) {
	merr := new(multierror.Error)
	for _, keyID := range passwordKeyIDs {
		// Attempt to remove all of them, don't fail early
		err := c.RemoveApplicationPassword(ctx, appID, keyID)
		if err != nil {
			merr = multierror.Append(merr, err)
		}
	}

	return merr.ErrorOrNil()
}
