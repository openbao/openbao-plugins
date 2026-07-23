// Copyright (c) 2026 OpenBao a Series of LF Projects, LLC
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"github.com/openbao/go-kms-wrapping/kms/pkcs11/v2"
	"github.com/openbao/go-kms-wrapping/plugin/v2"
	wrapping "github.com/openbao/go-kms-wrapping/v2"
	"github.com/openbao/go-kms-wrapping/v2/kms"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		KMSFactoryFunc: func() kms.KMS {
			return pkcs11.New(pkcs11.WithAliasesFromEnv())
		},
		WrapperFactoryFunc: func() wrapping.Wrapper {
			return pkcs11.NewWrapper(pkcs11.WithAliasesFromEnv())
		},
		Metadata: plugin.Metadata{
			SensitiveKMSFields: pkcs11.SensitiveKMSFields,
		},
	})
}
