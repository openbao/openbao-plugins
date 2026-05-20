// Copyright (c) 2026 OpenBao a Series of LF Projects, LLC
// Copyright (c) 2026 Floatplane Media
// SPDX-License-Identifier: MPL-2.0

package typesense

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	dbplugin "github.com/openbao/openbao/sdk/v2/database/dbplugin/v5"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/typesense"
)

type TypesenseTestConfig struct {
	Host    string
	Port    string
	APIKey  string
	Address string
}

func SetupEphemeralTypesense(t *testing.T) (TypesenseTestConfig, func()) {
	pool, err := dockertest.NewPool("")
	require.NoError(t, err, "Failed to construct Docker pool")

	err = pool.Client.Ping()
	require.NoError(t, err, "Failed to establish communication with Docker daemon")

	const (
		typesenseImage   = "typesense/typesense"
		typesenseVersion = "29.1"
		adminAPIKey      = "test-integration-key"
		internalDataDir  = "/data"
		internalPort     = "8108/tcp"
	)

	runOptions := &dockertest.RunOptions{
		Repository: typesenseImage,
		Tag:        typesenseVersion,
		Env: []string{
			fmt.Sprintf("TYPESENSE_API_KEY=%s", adminAPIKey),
			fmt.Sprintf("TYPESENSE_DATA_DIR=%s", internalDataDir),
		},
		ExposedPorts: []string{internalPort},
	}

	resource, err := pool.RunWithOptions(runOptions, func(config *docker.HostConfig) {
		config.Tmpfs = map[string]string{
			internalDataDir: "",
		}
	})
	require.NoError(t, err, "Failed to dispatch Typesense container")

	cleanup := func() {
		if err := pool.Purge(resource); err != nil {
			t.Errorf("Failed to purge Typesense container during cleanup: %v", err)
		}
	}

	hostPort := resource.GetPort(internalPort)
	hostAddress := fmt.Sprintf("http://localhost:%s", hostPort)

	client := typesense.NewClient(
		typesense.WithServer(hostAddress),
		typesense.WithAPIKey(adminAPIKey),
		typesense.WithConnectionTimeout(2*time.Second),
	)

	pool.MaxWait = 60 * time.Second
	err = pool.Retry(func() error {
		ok, err := client.Health(context.Background(), 2*time.Second)
		if err != nil {
			return fmt.Errorf("failed to check Typesense health: %w", err)
		}
		if !ok {
			return errors.New("typesense health check returned false")
		}
		return nil
	})

	if err != nil {
		cleanup()
		t.Fatalf("Typesense container failed to reach operational readiness: %v", err)
	}

	config := TypesenseTestConfig{
		Host:    "localhost",
		Port:    hostPort,
		APIKey:  adminAPIKey,
		Address: hostAddress,
	}

	return config, cleanup
}

func TestTypesensePlugin_Integration(t *testing.T) {
	config, cleanup := SetupEphemeralTypesense(t)
	defer cleanup()

	// 1. Initialize Plugin
	dbRaw, err := New()
	require.NoError(t, err)

	db, ok := dbRaw.(dbplugin.Database)
	require.True(t, ok)

	ctx := context.Background()

	initReq := dbplugin.InitializeRequest{
		Config: map[string]interface{}{
			"api_url": config.Address,
			"api_key": config.APIKey,
		},
		VerifyConnection: true,
	}

	_, err = db.Initialize(ctx, initReq)
	require.NoError(t, err, "Initialize should succeed with running Typesense instance")

	// 2. Create User
	newUserReq := dbplugin.NewUserRequest{
		UsernameConfig: dbplugin.UsernameMetadata{
			DisplayName: "test",
			RoleName:    "admin",
		},
		Statements: dbplugin.Statements{
			Commands: []string{`{"actions": ["*"], "collections": ["*"]}`},
		},
		Password:   "some-secure-secret-value",
		Expiration: time.Now().Add(1 * time.Hour),
	}

	newUserResp, err := db.NewUser(ctx, newUserReq)
	require.NoError(t, err, "NewUser should successfully create an API key")
	assert.NotEmpty(t, newUserResp.Username, "NewUser should return the generated username description")

	// 2b. Verify key validity directly against Typesense
	userClient := typesense.NewClient(
		typesense.WithServer(config.Address),
		typesense.WithAPIKey(newUserReq.Password),
		typesense.WithConnectionTimeout(2*time.Second),
	)

	_, err = userClient.Collections().Retrieve(ctx)
	require.NoError(t, err, "Should be able to retrieve collections using the generated key")

	// 3. Delete User
	deleteUserReq := dbplugin.DeleteUserRequest{
		Username: newUserResp.Username,
	}

	_, err = db.DeleteUser(ctx, deleteUserReq)
	require.NoError(t, err, "DeleteUser should successfully remove the created API key")

	// 3b. Verify key invalidity directly against Typesense
	_, err = userClient.Collections().Retrieve(ctx)
	require.Error(t, err, "Should fail to retrieve collections after the key is deleted")
}
