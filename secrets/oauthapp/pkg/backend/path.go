package backend

import (
	"fmt"

	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
)

// nameRegex allows any character other than a : followed by a /, which allows
// us to specially reserve a small subset of possible names for derived
// credentials (STS).
func nameRegex(name string) string {
	return fmt.Sprintf(`(?P<%s>(?:[^:]|:[^/])+)`, name)
}

func pathsSpecial() *logical.Paths {
	return &logical.Paths{
		SealWrapStorage: []string{
			CredsPathPrefix,
			SelfPathPrefix,
			ServersPathPrefix,
			STSPathPrefix,
		},
	}
}

func paths(b *backend) []*framework.Path {
	return []*framework.Path{
		pathAuthCodeURL(b),
		pathConfig(b),
		pathCreds(b),
		pathSelf(b),
		pathServersList(b),
		pathServers(b),
		pathSTS(b),
	}
}
