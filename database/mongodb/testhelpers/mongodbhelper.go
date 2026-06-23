// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package mongodb

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/openbao/openbao/sdk/v2/helper/docker"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func PrepareTestContainer(t *testing.T, version, configDir string) (func(), string) {
	if os.Getenv("MONGODB_URL") != "" {
		return func() {}, os.Getenv("MONGODB_URL")
	}

	runOptions := docker.RunOptions{
		ContainerName: "mongo",
		ImageRepo:     "mongo",
		ImageTag:      version,
		Ports:         []string{"27017/tcp"},

		OmitLogTimestamps: true, // mongodb server logs are already timestamped
		LogConsumer: func(msg string) {
			t.Log("[mongod]", msg)
		},
	}

	if configDir != "" {
		runOptions.Cmd = []string{"mongod", "--config", "/etc/mongo/mongod.conf"}
		runOptions.CopyFromTo = map[string]string{
			configDir: "/etc/mongo",
		}
	}

	runner, err := docker.NewServiceRunner(runOptions)
	if err != nil {
		t.Fatalf("could not start docker mongo: %s", err)
	}

	svc, err := runner.StartService(t.Context(), func(ctx context.Context, host string, port int) (docker.ServiceConfig, error) {
		connURL := fmt.Sprintf("mongodb://%s:%d", host, port)

		client, err := mongo.Connect(ctx, options.Client().ApplyURI(connURL))
		if err != nil {
			return nil, err
		}

		if err = client.Ping(ctx, readpref.Primary()); err != nil {
			t.Fatal(err)
		}

		if err = client.Disconnect(ctx); err != nil {
			t.Fatal(err)
		}

		return docker.NewServiceURLParse(connURL)
	})
	if err != nil {
		t.Fatalf("could not start docker mongo: %s", err)
	}

	return svc.Cleanup, svc.Config.URL().String()
}
