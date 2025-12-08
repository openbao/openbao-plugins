// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package partitions_test

import (
	"testing"

	endpoints "github.com/openbao/openbao-plugins/auth/aws/partitions"
)

func TestGeneratePartitionToRegionMap(t *testing.T) {
	res, ok := endpoints.GetGlobalRegionForPartition("aws")
	if !ok {
		t.Fatal("expected aws to be available")
	}
	if res != "us-east-1" {
		t.Fatal("expected us-east-1 but received " + res)
	}

	res, ok = endpoints.GetGlobalRegionForPartition("aws-us-gov")
	if !ok {
		t.Fatal("expected us-gov-west-1 to be available")
	}
	if res != "us-gov-west-1" {
		t.Fatal("expected us-gov-west-1 but received " + res)
	}

	res, ok = endpoints.GetGlobalRegionForPartition("gcp")
	if ok {
		t.Fatal("expected gcp not to be available, but got" + res)
	}

}
