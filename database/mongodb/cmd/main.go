// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"github.com/openbao/openbao-plugins/database/mongodb"
	"github.com/openbao/openbao/sdk/v2/database/dbplugin/v5"
)

func main() {
	// instantiates a MongoDB object, and runs the RPC server for the plugin
	dbplugin.ServeMultiplex(mongodb.New)
}
