// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package azure

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestJsonTime(t *testing.T) {
	type a struct {
		Time jsonTime `json:"time"`
	}

	now := time.Now().Round(time.Second)
	test1Raw := fmt.Sprintf(`{"time":%d}`, now.Unix())
	test1 := new(a)
	if err := json.Unmarshal([]byte(test1Raw), test1); err != nil {
		t.Fatalf("err: %v", err)
	}
	if !now.Equal(time.Time(test1.Time)) {
		t.Fatalf("expected: %s, got: %s", now, time.Time(test1.Time))
	}

	test2Raw := fmt.Sprintf(`{"time":%f}`, float64(now.Unix()))
	test2 := new(a)
	if err := json.Unmarshal([]byte(test2Raw), test2); err != nil {
		t.Fatalf("err: %v", err)
	}
	if !now.Equal(time.Time(test2.Time)) {
		t.Fatalf("expected: %s, got: %s", now, time.Time(test2.Time))
	}
}
