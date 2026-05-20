// Copyright (c) 2026 OpenBao a Series of LF Projects, LLC
// Copyright (c) 2026 Floatplane Media
// SPDX-License-Identifier: MPL-2.0

package typesense

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	dbplugin "github.com/openbao/openbao/sdk/v2/database/dbplugin/v5"
	"github.com/openbao/openbao/sdk/v2/helper/template"
	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"
)

const defaultUsernameTemplate = `{{ printf "v-%s-%s-%s" (.DisplayName | truncate 20) (.RoleName | truncate 20) (unix_time) }}`


// New returns a new Typesense database implementation.
func New() (interface{}, error) {
	db := &typesenseDB{}
	// Use the middleware to prevent leaking the Typesense admin API key in error messages
	return dbplugin.NewDatabaseErrorSanitizerMiddleware(db, db.secretValues), nil
}

// typesenseDB implements the dbplugin.Database interface
type typesenseDB struct {
	client *typesense.Client
	// apiKey is retained to allow the error sanitizer middleware to redact it from errors/logs
	apiKey string
}

// secretValues tells the sanitizer which strings to redact from logs/errors
func (db *typesenseDB) secretValues() map[string]string {
	return map[string]string{
		db.apiKey: "[typesense-api-key]",
	}
}

// Initialize configures the plugin and verifies the connection
func (db *typesenseDB) Initialize(ctx context.Context, req dbplugin.InitializeRequest) (dbplugin.InitializeResponse, error) {
	apiURL, ok := req.Config["api_url"].(string)
	if !ok || apiURL == "" {
		return dbplugin.InitializeResponse{}, errors.New("api_url is required")
	}
	apiKey, ok := req.Config["api_key"].(string)
	if !ok || apiKey == "" {
		return dbplugin.InitializeResponse{}, errors.New("api_key is required")
	}

	db.apiKey = apiKey
	db.client = typesense.NewClient(
		typesense.WithServer(apiURL),
		typesense.WithAPIKey(db.apiKey),
		typesense.WithConnectionTimeout(10*time.Second),
	)

	if req.VerifyConnection {
		// Ping the Typesense health endpoint
		ok, err := db.client.Health(ctx, 5*time.Second)
		if err != nil {
			return dbplugin.InitializeResponse{}, fmt.Errorf("failed to connect to Typesense: %w", err)
		}
		if !ok {
			return dbplugin.InitializeResponse{}, errors.New("typesense health check returned false")
		}
	}

	return dbplugin.InitializeResponse{
		Config: req.Config,
	}, nil
}

// NewUser creates a new API key in Typesense
func (db *typesenseDB) NewUser(ctx context.Context, req dbplugin.NewUserRequest) (dbplugin.NewUserResponse, error) {
	// 1. Generate the username dynamically using OpenBao's contextual metadata
	up, err := template.NewTemplate(template.Template(defaultUsernameTemplate))
	if err != nil {
		return dbplugin.NewUserResponse{}, fmt.Errorf("unable to initialize username template: %w", err)
	}

	generatedUsername, err := up.Generate(req.UsernameConfig)
	if err != nil {
		return dbplugin.NewUserResponse{}, fmt.Errorf("failed to generate username: %w", err)
	}


	// 2. Prepare the Typesense Key payload
	payloadStr := `{"actions": ["*"], "collections": ["*"]}`
	if len(req.Statements.Commands) > 0 {
		payloadStr = req.Statements.Commands[0]
	}

	var payload api.ApiKeySchema
	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		return dbplugin.NewUserResponse{}, fmt.Errorf("failed to parse creation_statements as JSON: %w", err)
	}

	// 3. Inject the dynamically generated identity and password
	payload.Description = generatedUsername
	payload.Value = &req.Password
	if !req.Expiration.IsZero() {
		exp := req.Expiration.Unix()
		payload.ExpiresAt = &exp
	}

	// 4. Send request to Typesense
	createdKey, err := db.client.Keys().Create(ctx, &payload)
	if err != nil {
		return dbplugin.NewUserResponse{}, fmt.Errorf("failed to create Typesense key: %w", err)
	}
	if createdKey == nil || createdKey.Id == nil {
		return dbplugin.NewUserResponse{}, fmt.Errorf("failed to retrieve created key ID from Typesense")
	}

	// 5. Return the generated username back to OpenBao so it can track it
	// Format: <id>-<generatedUsername>
	return dbplugin.NewUserResponse{
		Username: fmt.Sprintf("%d-%s", *createdKey.Id, generatedUsername),
	}, nil
}

// DeleteUser revokes the API key from Typesense
func (db *typesenseDB) DeleteUser(ctx context.Context, req dbplugin.DeleteUserRequest) (dbplugin.DeleteUserResponse, error) {
	var keyIDToDelete int64 = -1

	// Try to parse the key ID from the username (format: "<id>-<description>")
	parts := strings.SplitN(req.Username, "-", 2)
	if len(parts) == 2 {
		if id, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
			keyIDToDelete = id
		}
	}

	if keyIDToDelete == -1 {
		// Fallback: linear search for the key by description (for legacy leases)
		keys, err := db.client.Keys().Retrieve(ctx)
		if err != nil {
			return dbplugin.DeleteUserResponse{}, err
		}
		for _, k := range keys {
			if k.Description == req.Username && k.Id != nil {
				keyIDToDelete = *k.Id
				break
			}
		}
	}

	if keyIDToDelete == -1 {
		return dbplugin.DeleteUserResponse{}, nil // Key already gone
	}

	_, err := db.client.Key(keyIDToDelete).Delete(ctx)
	if err != nil {
		var httpErr *typesense.HTTPError
		if errors.As(err, &httpErr) && httpErr.Status == 404 {
			return dbplugin.DeleteUserResponse{}, nil // Key already gone
		}
		return dbplugin.DeleteUserResponse{}, fmt.Errorf("failed to delete key: %w", err)
	}

	return dbplugin.DeleteUserResponse{}, nil
}

// UpdateUser is not supported for Typesense keys.
func (db *typesenseDB) UpdateUser(ctx context.Context, req dbplugin.UpdateUserRequest) (dbplugin.UpdateUserResponse, error) {
	return dbplugin.UpdateUserResponse{}, errors.New("update/rotation is not supported for typesense keys")
}

// Type returns the type of the database.
func (db *typesenseDB) Type() (string, error) {
	return "typesense", nil
}

// Close is a no-op for Typesense.
func (db *typesenseDB) Close() error {
	return nil
}
