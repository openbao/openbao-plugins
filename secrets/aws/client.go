// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	awsv1 "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-hclog"
	awsutilv1 "github.com/hashicorp/go-secure-stdlib/awsutil"
	"github.com/hashicorp/go-secure-stdlib/awsutil/v2"
	"github.com/openbao/openbao/sdk/v2/logical"
)

// NOTE: The caller is required to ensure that b.clientMutex is at least read locked
func getRootConfig(ctx context.Context, s logical.Storage, clientType string, logger hclog.Logger) (*aws.Config, error) {
	credsConfig := &awsutil.CredentialsConfig{}
	var endpoint *string

	entry, err := s.Get(ctx, "config/root")
	if err != nil {
		return nil, err
	}
	if entry != nil {
		var config rootConfig
		if err := entry.DecodeJSON(&config); err != nil {
			return nil, fmt.Errorf("error reading root configuration: %w", err)
		}

		credsConfig.AccessKey = config.AccessKey
		credsConfig.SecretKey = config.SecretKey
		credsConfig.Region = config.Region
		credsConfig.MaxRetries = aws.Int(config.MaxRetries)
		switch {
		case clientType == "iam" && config.IAMEndpoint != "":
			endpoint = aws.String(config.IAMEndpoint)
		case clientType == "sts" && config.STSEndpoint != "":
			endpoint = aws.String(config.STSEndpoint)
		}

		if clientType == "sts" && config.STSRegion != "" {
			credsConfig.Region = config.STSRegion
		}
	}

	if credsConfig.Region == "" {
		credsConfig.Region = os.Getenv("AWS_REGION")
		if credsConfig.Region == "" {
			credsConfig.Region = os.Getenv("AWS_DEFAULT_REGION")
			if credsConfig.Region == "" {
				credsConfig.Region = "us-east-1"
			}
		}
	}

	credsConfig.HTTPClient = cleanhttp.DefaultClient()

	credsConfig.Logger = logger

	creds, err := credsConfig.GenerateCredentialChain(ctx)
	if err != nil {
		return nil, err
	}

	creds.BaseEndpoint = endpoint

	return creds, nil
}

func nonCachedClientIAM(ctx context.Context, s logical.Storage, logger hclog.Logger) (*iam.IAM, error) {
	awsConfig, err := getRootConfigV1(ctx, s, "iam", logger)
	if err != nil {
		return nil, err
	}
	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, err
	}
	client := iam.New(sess)
	if client == nil {
		return nil, fmt.Errorf("could not obtain iam client")
	}
	return client, nil
}

func nonCachedClientSTS(ctx context.Context, s logical.Storage, logger hclog.Logger) (*sts.Client, error) {
	awsConfig, err := getRootConfig(ctx, s, "sts", logger)
	if err != nil {
		return nil, err
	}
	client := sts.NewFromConfig(*awsConfig)
	if client == nil {
		return nil, fmt.Errorf("could not obtain sts client")
	}
	return client, nil
}

// NOTE: The caller is required to ensure that b.clientMutex is at least read locked
// This is a copy of the old getRootConfig and will be removed in a later commit
func getRootConfigV1(ctx context.Context, s logical.Storage, clientType string, logger hclog.Logger) (*awsv1.Config, error) {
	credsConfig := &awsutilv1.CredentialsConfig{}
	var endpoint string
	var maxRetries int = awsv1.UseServiceDefaultRetries

	entry, err := s.Get(ctx, "config/root")
	if err != nil {
		return nil, err
	}
	if entry != nil {
		var config rootConfig
		if err := entry.DecodeJSON(&config); err != nil {
			return nil, fmt.Errorf("error reading root configuration: %w", err)
		}

		credsConfig.AccessKey = config.AccessKey
		credsConfig.SecretKey = config.SecretKey
		credsConfig.Region = config.Region
		maxRetries = config.MaxRetries
		switch {
		case clientType == "iam" && config.IAMEndpoint != "":
			endpoint = *awsv1.String(config.IAMEndpoint)
		case clientType == "sts" && config.STSEndpoint != "":
			endpoint = *awsv1.String(config.STSEndpoint)
		}

		if clientType == "sts" && config.STSRegion != "" {
			credsConfig.Region = config.STSRegion
		}
	}

	if credsConfig.Region == "" {
		credsConfig.Region = os.Getenv("AWS_REGION")
		if credsConfig.Region == "" {
			credsConfig.Region = os.Getenv("AWS_DEFAULT_REGION")
			if credsConfig.Region == "" {
				credsConfig.Region = "us-east-1"
			}
		}
	}

	credsConfig.HTTPClient = cleanhttp.DefaultClient()

	credsConfig.Logger = logger

	creds, err := credsConfig.GenerateCredentialChain()
	if err != nil {
		return nil, err
	}

	return &awsv1.Config{
		Credentials: creds,
		Region:      awsv1.String(credsConfig.Region),
		Endpoint:    &endpoint,
		HTTPClient:  cleanhttp.DefaultClient(),
		MaxRetries:  awsv1.Int(maxRetries),
	}, nil
}
