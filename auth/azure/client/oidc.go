package client

import (
	"context"

	"github.com/coreos/go-oidc"
)

type TokenVerifier interface {
	Verify(ctx context.Context, token string) (*oidc.IDToken, error)
}
