// Copyright (c) 2026 OpenBao a Series of LF Projects, LLC
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"github.com/openbao/go-kms-wrapping/kms/securosyshsm/v2"

	"github.com/openbao/go-kms-wrapping/plugin/v2"
	wrapping "github.com/openbao/go-kms-wrapping/v2"
	"github.com/openbao/go-kms-wrapping/v2/kms"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		KMSFactoryFunc: func() kms.KMS {
			return securosyshsm.New()
		},
		WrapperFactoryFunc: func() wrapping.Wrapper {
			return securosyshsm.NewWrapper()
		},
		Metadata: plugin.Metadata{},
	})
}
