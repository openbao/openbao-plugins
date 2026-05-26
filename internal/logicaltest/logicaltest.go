// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package logicaltest

import (
	"errors"
	"testing"

	"github.com/openbao/openbao/api/v2"
	"github.com/openbao/openbao/sdk/v2/logical"
)

// TestEnvVar must be set to a non-empty value for acceptance tests to run.
const TestEnvVar = "BAO_ACC"

// TestCase is a single set of tests to run for a backend. A TestCase should
// generally map 1:1 to each test method for your acceptance tests.
type TestCase struct {
	// Precheck, if non-nil, will be called once before the test case runs at
	// all. This can be used for some validation prior to the test running.
	PreCheck func()

	// Backend is the backend that will be tested against.
	Backend logical.Backend

	// Steps are the set of operations that are run for this test case.
	Steps []TestStep

	// Teardown will be called before the test case is over regardless of if the
	// test succeeded or failed. This should return an error in the case that
	// the test can't guarantee all resources were properly cleaned up.
	Teardown func() error

	// AcceptanceTest, if set, the test case will be run only if TestEnvVar is
	// set. If not, this test case will be run as a unit test.
	AcceptanceTest bool
}

// TestStep is a single step within a TestCase.
type TestStep struct {
	// Operation is the operation to execute.
	Operation logical.Operation

	// Path is the request path.
	Path string

	// Data to pass to the request handler.
	Data map[string]any

	// Check is called after this step is executed in order to test that the
	// step executed successfully. If not set, the next step will be called.
	Check func(*logical.Response) error

	// PreFlight is called directly before execution of the request, allowing
	// modification of the request parameters (e.g., Path) with dynamic values.
	PreFlight func(*logical.Request) error

	// ErrorOk, if true, will not fail the test if the request handler responds
	// with an error.
	ErrorOk bool
}

// Test performs a series of tests on a backend with the given test case.
func Test(t *testing.T, c TestCase) {
	// We only run acceptance tests if an env var is set because they're
	// slow and generally require some outside configuration.
	if c.AcceptanceTest && api.ReadBaoVariable(TestEnvVar) == "" {
		t.Skipf(
			"Acceptance tests skipped unless env %q set",
			TestEnvVar,
		)
	}

	// Run the PreCheck if we have it.
	if c.PreCheck != nil {
		c.PreCheck()
	}

	// Defer Teardown, regardless of pass/fail at this point.
	if c.Teardown != nil {
		defer c.Teardown()
	}

	ctx := t.Context()
	storage := &logical.InmemStorage{}

	// Make requests.
	var revoke []*logical.Request
	for i, s := range c.Steps {
		req := &logical.Request{
			Operation: s.Operation,
			Path:      s.Path,
			Data:      s.Data,
			Storage:   storage,
		}

		// Run the PreFlight if we have it.
		if s.PreFlight != nil {
			if err := s.PreFlight(req); err != nil {
				t.Errorf("Failed preflight for step %d: %s", i+1, err)
				break
			}
		}

		// Make the actual request.
		resp, err := c.Backend.HandleRequest(ctx, req)

		// Revoke this secret later.
		if resp != nil && resp.Secret != nil {
			revoke = append(revoke, &logical.Request{
				Operation: logical.RevokeOperation,
				Path:      req.Path,
				Secret:    resp.Secret,
				Storage:   req.Storage,
			})
		}

		// Direct error from request handler.
		if err != nil && !s.ErrorOk {
			t.Errorf("Failed step %d: %s", i+1, err)
			break
		}

		// Error in response.
		if resp.IsError() && !s.ErrorOk {
			t.Errorf("Response contained error at step %d: %s", i+1, resp.Error())
			break
		}
	}

	// Revoke any secrets we might have.
	var failedRevokes []*logical.Secret
	for _, req := range revoke {
		t.Logf("Revoking secret at %q: %#v", req.Path, req.Secret)
		resp, err := c.Backend.HandleRequest(ctx, req)
		if err := errors.Join(err, resp.Error()); err != nil {
			failedRevokes = append(failedRevokes, req.Secret)
			t.Errorf("Revoke error: %s", err)
		}
	}

	// If we have any failed revokes, log it.
	if len(failedRevokes) > 0 {
		for _, s := range failedRevokes {
			t.Errorf(
				"WARNING: Revoking the following secret failed. It may\n"+
					"still exist. Please verify:\n\n%#v",
				s,
			)
		}
	}
}
