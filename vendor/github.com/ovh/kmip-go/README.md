# kmip-go
[![Go Reference](https://pkg.go.dev/badge/github.com/ovh/kmip-go.svg)](https://pkg.go.dev/github.com/ovh/kmip-go) [![license](https://img.shields.io/badge/license-Apache%202.0-red.svg?style=flat)](https://raw.githubusercontent.com/ovh/kmip-go/master/LICENSE) [![test](https://github.com/ovh/kmip-go/actions/workflows/test.yaml/badge.svg)](https://github.com/ovh/kmip-go/actions/workflows/test.yaml) [![Go Report Card](https://goreportcard.com/badge/github.com/ovh/kmip-go)](https://goreportcard.com/report/github.com/ovh/kmip-go)

A go implementation of the KMIP protocol and client, supporting KMIP v1.0 to v1.4.
See [KMIP v1.4 protocole specification](https://docs.oasis-open.org/kmip/spec/v1.4/os/kmip-spec-v1.4-os.pdf)

This library is developped for and tested against [OVHcloud KMS](https://help.ovhcloud.com/csm/en-ie-kms-quick-start?id=kb_article_view&sysparm_article=KB0063362).

> **NOTE:** THIS PROJECT IS CURRENTLY UNDER DEVELOPMENT AND SUBJECT TO BREAKING CHANGES.

## Usage

Add it to your project by running
```bash
go get github.com/ovh/kmip-go@latest
```
and import required packages
```go
import (
	"github.com/ovh/kmip-go"
	"github.com/ovh/kmip-go/kmipclient"
	"github.com/ovh/kmip-go/payloads"
	"github.com/ovh/kmip-go/ttlv"
)
```

Then you can connect to your KMS service:
```go
const (
	ADDR = "eu-west-rbx.okms.ovh.net:5696"
	CA   = "ca.pem"
	CERT = "cert.pem"
	KEY  = "key.pem"
)

client, err := kmipclient.Dial(
	ADDR,
	// Optional if server's CA is known by the system
	// kmipclient.WithRootCAFile(CA),
	kmipclient.WithClientCertFiles(CERT, KEY),
	kmipclient.WithMiddlewares(
		kmipclient.CorrelationValueMiddleware(uuid.NewString),
		kmipclient.DebugMiddleware(os.Stdout, ttlv.MarshalXML),
	),
	// kmipclient.EnforceVersion(kmip.V1_4),
)
if err != nil {
	panic(err)
}
defer client.Close()
fmt.Println("Connected using KMIP version", client.Version())
```

You can then use the high level client helper methods to create and send requests
to the server:
```go
resp := client.Create().
	AES(256, kmip.CryptographicUsageEncrypt|kmip.CryptographicUsageDecrypt).
	WithName("my-key").
	MustExec()
fmt.Println("Created AES key with ID", resp.UniqueIdentifier)
```

Or alternatively if more flexibility is required, craft your kmip requests payloads:
```go
request := payloads.CreateRequestPayload{
	ObjectType: kmip.ObjectTypeSymmetricKey,
	TemplateAttribute: kmip.TemplateAttribute{
		Attribute: []kmip.Attribute{
			{
				AttributeName:  kmip.AttributeNameCryptographicAlgorithm,
				AttributeValue: kmip.CryptographicAlgorithmAES,
			}, {
				AttributeName:  kmip.AttributeNameCryptographicLength,
				AttributeValue: int32(256),
			}, {
				AttributeName: kmip.AttributeNameName,
				AttributeValue: kmip.Name{
					NameType:  kmip.NameTypeUninterpretedTextString,
					NameValue: "another-key",
				},
			}, {
				AttributeName:  kmip.AttributeNameCryptographicUsageMask,
				AttributeValue: kmip.CryptographicUsageEncrypt | kmip.CryptographicUsageDecrypt,
			},
		},
	},
}

response, err := client.Request(context.Background(), &request)
if err != nil {
	panic(err)
}
id := response.(*payloads.CreateResponsePayload).UniqueIdentifier
fmt.Println("Created an AES key with ID", id)
```

You can also send batches of requests:
```go
batchResponse, err := client.Batch(context.Background(), &request, &request)
if err != nil {
	panic(err)
}
id1 := batchResponse[0].ResponsePayload.(*payloads.CreateResponsePayload).UniqueIdentifier
id2 := batchResponse[1].ResponsePayload.(*payloads.CreateResponsePayload).UniqueIdentifier
fmt.Println("Created 2 AES keys with IDs", id1, id2)
```

And directly craft your request message with one or more payloads batched together:
```go
msg := kmip.NewRequestMessage(client.Version(), &request, &request)
rMsg, err := client.Roundtrip(context.Background(), &msg)
if err != nil {
	panic(err)
}
id1 := rMsg.BatchItem[0].ResponsePayload.(*payloads.CreateResponsePayload).UniqueIdentifier
id2 := rMsg.BatchItem[1].ResponsePayload.(*payloads.CreateResponsePayload).UniqueIdentifier
fmt.Println("Created a 5th and 6th AES keys with IDs", id1, id2)
```
}

See [examples](./examples) for more possibilities.

## Implementation status

> **Legend:**
> * N/A : Not Applicable 
> * ✅ : Fully compatible
> * ❌ : Not implemented or reviewed
> * 🚧 : Work in progress / Partially compatible
> * 💀 : Deprecated

### Messages
|                      | v1.0 | v1.1 | v1.2 | v1.3 | v1.4 |
| -------------------- | ---- | ---- | ---- | ---- | ---- |
| Request Message      |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Response Message     |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |

### Operations
| Operation            | v1.0 | v1.1 | v1.2 | v1.3 | v1.4 |
| -------------------- | ---- | ---- | ---- | ---- | ---- |
| Create               |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Create Key Pair      |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Register             |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Re-key               |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| DeriveKey            |  ❌  |  ❌  |  ❌  |  ❌  |  ❌  |
| Certify              |  ❌  |  ❌  |  ❌  |  ❌  |  ❌  |
| Re-certify           |  ❌  |  ❌  |  ❌  |  ❌  |  ❌  |
| Locate               |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Check                |  ❌  |  ❌  |  ❌  |  ❌  |  ❌  |
| Get                  |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Get Attributes       |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Get Attribute List   |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Add Attribute        |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Modify Attribute     |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Delete Attribute     |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Obtain Lease         |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Get Usage Allocation |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Activate             |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Revoke               |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Destroy              |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Archive              |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Recover              |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Validate             |  ❌  |  ❌  |  ❌  |  ❌  |  ❌  |
| Query                |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Cancel               |  ❌  |  ❌  |  ❌  |  ❌  |  ❌  |
| Poll                 |  ❌  |  ❌  |  ❌  |  ❌  |  ❌  |
| Notify               |  ❌  |  ❌  |  ❌  |  ❌  |  ❌  |
| Put                  |  ❌  |  ❌  |  ❌  |  ❌  |  ❌  |
| Discover             |  N/A |  ✅  |  ✅  |  ✅  |  ✅  |
| Re-key Key Pair      |  N/A |  ❌  |  ❌  |  ❌  |  ❌  |
| Encrypt              |  N/A |  N/A |  ✅  |  ✅  |  ✅  |
| Decrypt              |  N/A |  N/A |  ✅  |  ✅  |  ✅  |
| Sign                 |  N/A |  N/A |  ✅  |  ✅  |  ✅  |
| Signature Verify     |  N/A |  N/A |  ✅  |  ✅  |  ✅  |
| MAC                  |  N/A |  N/A |  ❌  |  ❌  |  ❌  |
| MAC Verify           |  N/A |  N/A |  ❌  |  ❌  |  ❌  |
| RNG Retrieve         |  N/A |  N/A |  ❌  |  ❌  |  ❌  |
| RNG Seed             |  N/A |  N/A |  ❌  |  ❌  |  ❌  |
| Hash                 |  N/A |  N/A |  ❌  |  ❌  |  ❌  |
| Create Split Key     |  N/A |  N/A |  ❌  |  ❌  |  ❌  |
| Join Split Key       |  N/A |  N/A |  ❌  |  ❌  |  ❌  |
| Export               |  N/A |  N/A |  N/A |  N/A |  ❌  |
| Import               |  N/A |  N/A |  N/A |  N/A |  ❌  |

### Managed Objects
| Object        | v1.0 | v1.1 | v1.2 | v1.3 | v1.4 |
| ------------- | ---- | ---- | ---- | ---- | ---- |
| Certificate   |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Symmetric Key |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Public Key    |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Private Key   |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Split Key     |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Template      |  ✅  |  ✅  |  ✅  |  💀  |  💀  |
| Secret Data   |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Opaque Object |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| PGP Key       |  N/A |  N/A |  ✅  |  ✅  |  ✅  |

### Base Objects
| Object                                   | v1.0 | v1.1 | v1.2 | v1.3 | v1.4 |
| ---------------------------------------- | ---- | ---- | ---- | ---- | ---- |
| Attribute                                |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Credential                               |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Key Block                                |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Key Value                                |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Key Wrapping Data                        |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Key Wrapping Specification               |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Transparent Key Structures               |  🚧  |  🚧  |  🚧  |  🚧  |  🚧  |
| Template-Attribute Structures            |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Extension Information                    |  N/A |  ✅  |  ✅  |  ✅  |  ✅  |
| Data                                     |  N/A |  N/A |  ✅  |  ✅  |  ✅  |
| Data Length                              |  N/A |  N/A |  ❌  |  ❌  |  ❌  |
| Signature Data                           |  N/A |  N/A |  ✅  |  ✅  |  ✅  |
| MAC Data                                 |  N/A |  N/A |  ❌  |  ❌  |  ❌  |
| Nonce                                    |  N/A |  N/A |  ✅  |  ✅  |  ✅  |
| Correlation Value                        |  N/A |  N/A |  N/A |  ✅  |  ✅  |
| Init Indicator                           |  N/A |  N/A |  N/A |  ✅  |  ✅  |
| Final Indicator                          |  N/A |  N/A |  N/A |  ✅  |  ✅  |
| RNG Parameter                            |  N/A |  N/A |  N/A |  ✅  |  ✅  |
| Profile Information                      |  N/A |  N/A |  N/A |  ✅  |  ✅  |
| Validation Information                   |  N/A |  N/A |  N/A |  ✅  |  ✅  |
| Capability Information                   |  N/A |  N/A |  N/A |  ✅  |  ✅  |
| Authenticated Encryption Additional Data |  N/A |  N/A |  N/A |  N/A |  ✅  |
| Authenticated Encryption Tag             |  N/A |  N/A |  N/A |  N/A |  ✅  |

#### Transparent Key Structures
| Object                   | v1.0 | v1.1 | v1.2 | v1.3 | v1.4 |
| ------------------------ | ---- | ---- | ---- | ---- | ---- |
| Symmetric Key            |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| DSA Private/Public Key   |  ❌  |  ❌  |  ❌  |  ❌  |  ❌  |
| RSA Private/Public Key   |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| DH Private/Public Key    |  ❌  |  ❌  |  ❌  |  ❌  |  ❌  |
| ECDSA Private/Public Key |  ✅  |  ✅  |  ✅  |  💀  |  💀  |
| ECDH Private/Public Key  |  ❌  |  ❌  |  ❌  |  💀  |  💀  |
| ECMQV Private/Public     |  ❌  |  ❌  |  ❌  |  💀  |  💀  |
| EC Private/Public        |  N/A |  N/A |  N/A |  ✅  |  ✅  |

### Attributes
| Attribute                        | v1.0 | v1.1 | v1.2 | v1.3 | v1.4 |
| -------------------------------- | ---- | ---- | ---- | ---- | ---- |
| Unique Identifier                |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Name                             |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Object Type                      |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Cryptographic Algorithm          |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Cryptographic Length             |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Cryptographic Parameters         |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Cryptographic Domain Parameters  |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Certificate Type                 |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Certificate Identifier           |  ✅  |  💀  |  💀  |  💀  |  💀  |
| Certificate Subject              |  ✅  |  💀  |  💀  |  💀  |  💀  |
| Certificate Issuer               |  ✅  |  💀  |  💀  |  💀  |  💀  |
| Digest                           |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Operation Policy Name            |  ✅  |  ✅  |  ✅  |  💀  |  💀  |
| Cryptographic Usage Mask         |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Lease Time                       |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Usage Limits                     |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| State                            |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Initial Date                     |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Activation Date                  |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Process Start Date               |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Protect Stop Date                |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Deactivation Date                |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Destroy Date                     |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Compromise Occurrence Date       |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Compromise Date                  |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Revocation Reason                |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Archive Date                     |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Object Group                     |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Link                             |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Application Specific Information |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Contact Information              |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Last Change Date                 |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Custom Attribute                 |  ✅  |  ✅  |  ✅  |  ✅  |  ✅  |
| Certificate Length               |  N/A |  ✅  |  ✅  |  ✅  |  ✅  |
| X.509 Certificate Identifier     |  N/A |  ✅  |  ✅  |  ✅  |  ✅  |
| X.509 Certificate Subject        |  N/A |  ✅  |  ✅  |  ✅  |  ✅  |
| X.509 Certificate Issuer         |  N/A |  ✅  |  ✅  |  ✅  |  ✅  |
| Digital Signature Algorithm      |  N/A |  ✅  |  ✅  |  ✅  |  ✅  |
| Fresh                            |  N/A |  ✅  |  ✅  |  ✅  |  ✅  |
| Alternative Name                 |  N/A |  N/A |  ✅  |  ✅  |  ✅  |
| Key Value Present                |  N/A |  N/A |  ✅  |  ✅  |  ✅  |
| Key Value Location               |  N/A |  N/A |  ✅  |  ✅  |  ✅  |
| Original Creation Date           |  N/A |  N/A |  ✅  |  ✅  |  ✅  |
| Random Number Generator          |  N/A |  N/A |  N/A |  ✅  |  ✅  |
| PKCS#12 Friendly Name            |  N/A |  N/A |  N/A |  N/A |  ✅  |
| Description                      |  N/A |  N/A |  N/A |  N/A |  ✅  |
| Comment                          |  N/A |  N/A |  N/A |  N/A |  ✅  |
| Sensitive                        |  N/A |  N/A |  N/A |  N/A |  ✅  |
| Always Sensitive                 |  N/A |  N/A |  N/A |  N/A |  ✅  |
| Extractable                      |  N/A |  N/A |  N/A |  N/A |  ✅  |
| Never Extractable                |  N/A |  N/A |  N/A |  N/A |  ✅  |
