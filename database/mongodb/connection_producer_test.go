// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package mongodb

import (
	"context"
	"fmt"
	"os"
	paths "path"
	"reflect"
	"sort"
	"testing"
	"time"

	mongodb "github.com/openbao/openbao-plugins/database/mongodb/testhelpers"
	dbplugin "github.com/openbao/openbao/sdk/v2/database/dbplugin/v5"
	"github.com/tsaarni/certyaml"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func TestInit_clientTLS(t *testing.T) {
	confDir := t.TempDir()

	// Create certificates for Mongo authentication
	TRUE := true
	caCert := &certyaml.Certificate{
		Subject: "CN=test certificate authority",
		IsCA:    &TRUE,
	}
	serverCert := &certyaml.Certificate{
		Subject:         "CN=server",
		SubjectAltNames: []string{"DNS:localhost", "IP:127.0.0.1"},
		Issuer:          caCert,
	}
	clientCert := &certyaml.Certificate{
		Subject: "CN=client",
		Issuer:  caCert,
	}

	writeFile(t, paths.Join(confDir, "ca.pem"), caCert.CertPEM(), 0o644)
	writeFile(t, paths.Join(confDir, "server.pem"), fmt.Append(serverCert.KeyPEM(), "\n", string(serverCert.CertPEM())), 0o644)
	writeFile(t, paths.Join(confDir, "client.pem"), fmt.Append(clientCert.KeyPEM(), "\n", string(clientCert.CertPEM())), 0o644)

	// //////////////////////////////////////////////////////
	// Set up Mongo config file
	rawConf := `
net:
   tls:
      mode: preferTLS
      certificateKeyFile: /etc/mongo/server.pem
      CAFile: /etc/mongo/ca.pem
      allowInvalidHostnames: true`

	writeFile(t, paths.Join(confDir, "mongod.conf"), []byte(rawConf), 0o644)

	// //////////////////////////////////////////////////////
	// Start Mongo container
	cleanup, retURL := mongodb.PrepareTestContainer(t, "latest", confDir)
	defer cleanup()

	// //////////////////////////////////////////////////////
	// Set up x509 user
	mClient := connect(t, retURL)

	setUpX509User(t, mClient, clientCert)

	// //////////////////////////////////////////////////////
	// Test
	mongo := new()

	initReq := dbplugin.InitializeRequest{
		Config: map[string]any{
			"connection_url":      retURL,
			"allowed_roles":       "*",
			"tls_certificate_key": fmt.Appendln(clientCert.KeyPEM(), string(clientCert.CertPEM())),
			"tls_ca":              caCert.CertPEM(),
		},
		VerifyConnection: true,
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	_, err := mongo.Initialize(ctx, initReq)
	if err != nil {
		t.Fatalf("Unable to initialize mongo engine: %s", err)
	}

	// Initialization complete. The connection was established, but we need to ensure
	// that we're connected as the right user
	whoamiCmd := map[string]any{
		"connectionStatus": 1,
	}

	client, err := mongo.Connection(ctx)
	if err != nil {
		t.Fatalf("Unable to make connection to Mongo: %s", err)
	}
	result := client.Database("test").RunCommand(ctx, whoamiCmd)
	if result.Err() != nil {
		t.Fatalf("Unable to connect to Mongo: %s", err)
	}

	expected := connStatus{
		AuthInfo: authInfo{
			AuthenticatedUsers: []user{
				{
					User: clientCert.Subject,
					DB:   "$external",
				},
			},
			AuthenticatedUserRoles: []role{
				{
					Role: "readWrite",
					DB:   "test",
				},
				{
					Role: "userAdminAnyDatabase",
					DB:   "admin",
				},
			},
		},
		Ok: 1,
	}
	// Sort the AuthenticatedUserRoles because Mongo doesn't return them in the same order every time
	// Thanks Mongo! /tableflip
	sort.Sort(expected.AuthInfo.AuthenticatedUserRoles)

	actual := connStatus{}
	err = result.Decode(&actual)
	if err != nil {
		t.Fatalf("Unable to decode connection status: %s", err)
	}

	sort.Sort(actual.AuthInfo.AuthenticatedUserRoles)

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("Actual:%#v\nExpected:\n%#v", actual, expected)
	}
}

func connect(t *testing.T, uri string) (client *mongo.Client) {
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("Unable to make connection to Mongo: %s", err)
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		t.Fatalf("Failed to ping Mongo server: %s", err)
	}

	return client
}

func setUpX509User(t *testing.T, client *mongo.Client, cert *certyaml.Certificate) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	cmd := &createUserCommand{
		Username: cert.Subject,
		Roles: []any{
			mongodbRole{
				Role: "readWrite",
				DB:   "test",
			},
			mongodbRole{
				Role: "userAdminAnyDatabase",
				DB:   "admin",
			},
		},
	}

	result := client.Database("$external").RunCommand(ctx, cmd)
	err := result.Err()
	if err != nil {
		t.Fatalf("Failed to create x509 user in database: %s", err)
	}
}

type connStatus struct {
	AuthInfo authInfo `bson:"authInfo"`
	Ok       int      `bson:"ok"`
}

type authInfo struct {
	AuthenticatedUsers     []user `bson:"authenticatedUsers"`
	AuthenticatedUserRoles roles  `bson:"authenticatedUserRoles"`
}

type user struct {
	User string `bson:"user"`
	DB   string `bson:"db"`
}

type role struct {
	Role string `bson:"role"`
	DB   string `bson:"db"`
}

type roles []role

func (r roles) Len() int           { return len(r) }
func (r roles) Less(i, j int) bool { return r[i].Role < r[j].Role }
func (r roles) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }

// ////////////////////////////////////////////////////////////////////////////
// Writing to file
// ////////////////////////////////////////////////////////////////////////////
func writeFile(t *testing.T, filename string, data []byte, perms os.FileMode) {
	t.Helper()

	err := os.WriteFile(filename, data, perms)
	if err != nil {
		t.Fatalf("Unable to write to file [%s]: %s", filename, err)
	}
}
