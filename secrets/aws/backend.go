// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package aws

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
	"github.com/openbao/openbao/sdk/v2/queue"
)

var pluginVersion string

const (
	rootConfigPath        = "config/root"
	minAwsUserRollbackAge = 5 * time.Minute
	operationPrefixAWS    = "aws"
	operationPrefixAWSASD = "aws-config"
)

type STSAPI interface {
	AssumeRole(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
	GetFederationToken(ctx context.Context, params *sts.GetFederationTokenInput, optFns ...func(*sts.Options)) (*sts.GetFederationTokenOutput, error)
}

type IAMAPI interface {
	AddUserToGroup(ctx context.Context, params *iam.AddUserToGroupInput, optFns ...func(*iam.Options)) (*iam.AddUserToGroupOutput, error)
	AttachUserPolicy(ctx context.Context, params *iam.AttachUserPolicyInput, optFns ...func(*iam.Options)) (*iam.AttachUserPolicyOutput, error)
	CreateAccessKey(ctx context.Context, params *iam.CreateAccessKeyInput, optFns ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error)
	CreateUser(ctx context.Context, params *iam.CreateUserInput, optFns ...func(*iam.Options)) (*iam.CreateUserOutput, error)
	DeleteAccessKey(ctx context.Context, params *iam.DeleteAccessKeyInput, optFns ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error)
	DeleteUser(ctx context.Context, params *iam.DeleteUserInput, optFns ...func(*iam.Options)) (*iam.DeleteUserOutput, error)
	DeleteUserPolicy(ctx context.Context, params *iam.DeleteUserPolicyInput, optFns ...func(*iam.Options)) (*iam.DeleteUserPolicyOutput, error)
	DetachUserPolicy(ctx context.Context, params *iam.DetachUserPolicyInput, optFns ...func(*iam.Options)) (*iam.DetachUserPolicyOutput, error)
	GetGroupPolicy(ctx context.Context, params *iam.GetGroupPolicyInput, optFns ...func(*iam.Options)) (*iam.GetGroupPolicyOutput, error)
	GetUser(ctx context.Context, params *iam.GetUserInput, optFns ...func(*iam.Options)) (*iam.GetUserOutput, error)
	ListAccessKeys(ctx context.Context, params *iam.ListAccessKeysInput, optFns ...func(*iam.Options)) (*iam.ListAccessKeysOutput, error)
	ListAttachedGroupPolicies(ctx context.Context, params *iam.ListAttachedGroupPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListAttachedGroupPoliciesOutput, error)
	ListAttachedUserPolicies(ctx context.Context, params *iam.ListAttachedUserPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListAttachedUserPoliciesOutput, error)
	ListGroupPolicies(ctx context.Context, params *iam.ListGroupPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListGroupPoliciesOutput, error)
	ListGroupsForUser(ctx context.Context, params *iam.ListGroupsForUserInput, optFns ...func(*iam.Options)) (*iam.ListGroupsForUserOutput, error)
	ListUserPolicies(ctx context.Context, params *iam.ListUserPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListUserPoliciesOutput, error)
	PutUserPolicy(ctx context.Context, params *iam.PutUserPolicyInput, optFns ...func(*iam.Options)) (*iam.PutUserPolicyOutput, error)
	RemoveUserFromGroup(ctx context.Context, params *iam.RemoveUserFromGroupInput, optFns ...func(*iam.Options)) (*iam.RemoveUserFromGroupOutput, error)
	TagUser(ctx context.Context, params *iam.TagUserInput, optFns ...func(*iam.Options)) (*iam.TagUserOutput, error)
}

func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b := Backend(conf)
	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}
	return b, nil
}

func Backend(_ *logical.BackendConfig) *backend {
	var b backend
	b.credRotationQueue = queue.New()
	b.Backend = &framework.Backend{
		Help: strings.TrimSpace(backendHelp),

		PathsSpecial: &logical.Paths{
			LocalStorage: []string{
				framework.WALPrefix,
			},
			SealWrapStorage: []string{
				rootConfigPath,
				pathStaticCreds + "/",
			},
		},

		Paths: []*framework.Path{
			pathConfigRoot(&b),
			pathConfigRotateRoot(&b),
			pathConfigLease(&b),
			pathRoles(&b),
			pathListRoles(&b),
			pathStaticRoles(&b),
			pathStaticCredentials(&b),
			pathUser(&b),
		},

		Secrets: []*framework.Secret{
			secretAccessKeys(&b),
		},

		Invalidate:        b.invalidate,
		WALRollback:       b.walRollback,
		WALRollbackMinAge: minAwsUserRollbackAge,
		PeriodicFunc: func(ctx context.Context, req *logical.Request) error {
			return b.rotateExpiredStaticCreds(ctx, req)
		},
		BackendType:    logical.TypeLogical,
		RunningVersion: pluginVersion,
	}

	return &b
}

type backend struct {
	*framework.Backend

	// Mutex to protect access to reading and writing policies
	roleMutex sync.RWMutex

	// Mutex to protect access to iam/sts clients and client configs
	clientMutex sync.RWMutex

	// iamClient and stsClient hold configured iam and sts clients for reuse, and
	// to enable mocking with AWS iface for tests
	iamClient IAMAPI
	stsClient STSAPI

	// the age of a static role's credential is tracked by a priority queue and handled
	// by the PeriodicFunc
	credRotationQueue *queue.PriorityQueue
}

const backendHelp = `
The AWS backend dynamically generates AWS access keys for a set of
IAM policies. The AWS access keys have a configurable lease set and
are automatically revoked at the end of the lease.

After mounting this backend, credentials to generate IAM keys must
be configured with the "root" path and policies must be written using
the "roles/" endpoints before any access keys can be generated.
`

func (b *backend) invalidate(ctx context.Context, key string) {
	switch {
	case key == rootConfigPath:
		b.clearClients()
	}
}

// clearClients clears the backend's IAM and STS clients
func (b *backend) clearClients() {
	b.clientMutex.Lock()
	defer b.clientMutex.Unlock()
	b.iamClient = nil
	b.stsClient = nil
}

// clientIAM returns the configured IAM client. If nil, it constructs a new one
// and returns it, setting it the internal variable
func (b *backend) clientIAM(ctx context.Context, s logical.Storage) (IAMAPI, error) {
	b.clientMutex.RLock()
	if b.iamClient != nil {
		b.clientMutex.RUnlock()
		return b.iamClient, nil
	}

	// Upgrade the lock for writing
	b.clientMutex.RUnlock()
	b.clientMutex.Lock()
	defer b.clientMutex.Unlock()

	// check client again, in the event that a client was being created while we
	// waited for Lock()
	if b.iamClient != nil {
		return b.iamClient, nil
	}

	iamClient, err := nonCachedClientIAM(ctx, s, b.Logger())
	if err != nil {
		return nil, err
	}
	b.iamClient = iamClient

	return b.iamClient, nil
}

func (b *backend) clientSTS(ctx context.Context, s logical.Storage) (STSAPI, error) {
	b.clientMutex.RLock()
	if b.stsClient != nil {
		b.clientMutex.RUnlock()
		return b.stsClient, nil
	}

	// Upgrade the lock for writing
	b.clientMutex.RUnlock()
	b.clientMutex.Lock()
	defer b.clientMutex.Unlock()

	// check client again, in the event that a client was being created while we
	// waited for Lock()
	if b.stsClient != nil {
		return b.stsClient, nil
	}

	stsClient, err := nonCachedClientSTS(ctx, s, b.Logger())
	if err != nil {
		return nil, err
	}
	b.stsClient = stsClient

	return b.stsClient, nil
}
