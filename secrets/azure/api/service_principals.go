// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package api

import (
	"context"
	"time"
)

type ServicePrincipalClient interface {
	// CreateServicePrincipal in Azure. The password returned is the actual password that the appID was created with
	CreateServicePrincipal(ctx context.Context, appID string, startDate time.Time, endDate time.Time) (id string, password string, err error)
	DeleteServicePrincipal(ctx context.Context, spObjectID string, permanentlyDelete bool) error
}

type ServicePrincipal struct {
	ObjectID string
	AppID    string
}
