module github.com/hashicorp/vault-plugin-secrets-gcpkms

go 1.12

require (
	cloud.google.com/go/kms v1.4.0
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/gammazero/deque v0.0.0-20190130191400-2afb3858e9c7 // indirect
	github.com/gammazero/workerpool v0.0.0-20190406235159-88d534f22b56
	github.com/golang/protobuf v1.5.2
	github.com/hashicorp/errwrap v1.0.0
	github.com/hashicorp/go-hclog v0.12.0
	github.com/hashicorp/go-multierror v1.0.0
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.1
	github.com/hashicorp/go-version v1.2.0 // indirect
	github.com/hashicorp/vault/api v1.0.5-0.20200215224050-f6547fa8e820
	github.com/hashicorp/vault/sdk v0.1.14-0.20200215224050-f6547fa8e820
	github.com/hashicorp/yamux v0.0.0-20181012175058-2f1d1f20f75d // indirect
	github.com/jeffchao/backoff v0.0.0-20140404060208-9d7fd7aa17f2
	github.com/satori/go.uuid v1.2.0
	golang.org/x/oauth2 v0.0.0-20220524215830-622c5d57e401
	google.golang.org/api v0.83.0
	google.golang.org/genproto v0.0.0-20220602131408-e326c6e8e9c8
	google.golang.org/grpc v1.47.0
)
