// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package version

import "fmt"

const (
	// Name is the name of the plugin.
	Name = "vault-plugin-secrets-gcpkms"

	// Version is the version of the release.
	Version = "0.0.1"
)

var (
	// GitCommit is the specific git commit of the plugin. This is completed by
	// the compiler.
	GitCommit string

	// HumanVersion is the human-formatted version of the plugin.
	HumanVersion = fmt.Sprintf("%s v%s (%s)", Name, Version, GitCommit)
)
