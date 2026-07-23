// Copyright (c) 2026 OpenBao a Series of LF Projects, LLC
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"errors"
	"flag"
	"strings"

	"github.com/openbao/go-kms-wrapping/kms/pkcs11/v2"
	"github.com/openbao/go-kms-wrapping/plugin/v2"
	wrapping "github.com/openbao/go-kms-wrapping/v2"
	"github.com/openbao/go-kms-wrapping/v2/kms"
)

func main() {
	aliases := make(map[string]string)

	flag.Func(
		"alias",
		"Register a PKCS#11 library path alias. This option is repeatable.",
		func(s string) error {
			parts := strings.SplitN(s, "=", 2)
			if len(parts) != 2 {
				return errors.New(`aliases are passed as "<name>=<path>"`)
			}
			aliases[parts[0]] = parts[1]
			return nil
		},
	)

	flag.Parse()

	opts := []pkcs11.Option{
		pkcs11.WithAliasesFromEnv(),
		pkcs11.WithAliases(aliases),
	}

	plugin.Serve(&plugin.ServeOpts{
		KMSFactoryFunc: func() kms.KMS {
			return pkcs11.New(opts...)
		},
		WrapperFactoryFunc: func() wrapping.Wrapper {
			return pkcs11.NewWrapper(opts...)
		},
		Metadata: plugin.Metadata{
			SensitiveKMSFields: pkcs11.SensitiveKMSFields,
		},
	})
}
