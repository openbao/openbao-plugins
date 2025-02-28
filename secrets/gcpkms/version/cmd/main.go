// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/openbao/openbao-plugins/secrets/gcpkms/version"
)

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		log.Fatal("missing argument")
	}

	switch args[0] {
	case "name":
		fmt.Printf("%s", version.Name)
	case "version":
		fmt.Printf("%s", version.Version)
	default:
		log.Fatalf("unknown arg %q", args[0])
	}
}
