// Copyright (c) 2025 OpenBao a Series of LF Projects, LLC
// SPDX-License-Identifier: MPL-2.0

package partitions

//go:generate ./gen.sh

func GetGlobalRegionForPartition(partition string) (string, bool) {
	region, ok := data[partition]
	return region, ok
}
