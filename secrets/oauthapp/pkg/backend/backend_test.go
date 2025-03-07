package backend

import (
	"context"
	"testing"
	"time"

	"github.com/openbao/openbao/sdk/v2/logical"
	"github.com/stretchr/testify/require"
)

func TestBackendNew(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	b, err := Factory(ctx, &logical.BackendConfig{StorageView: &logical.InmemStorage{}})
	require.NoError(t, err)
	require.NotNil(t, b)
}
