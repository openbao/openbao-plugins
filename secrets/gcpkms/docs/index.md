# Google Cloud KMS secrets engine

The Google Cloud KMS OpenBao secrets engine provides encryption and key management
via [Google Cloud KMS][kms]. It supports management of keys, including creation,
rotation, and revocation, as well as encrypting and decrypting data with managed
keys. This enables management of KMS keys through OpenBao's policies and IAM
system.

## Setup

Most secrets engines must be configured in advance before they can perform their
functions. These steps are usually completed by an operator or configuration
management tool.

Enable the Google Cloud KMS secrets engine:

```text
$ bao secrets enable gcpkms
Success! Enabled the gcpkms secrets engine at: gcpkms/
```

By default, the secrets engine will mount at the name of the engine. To
enable the secrets engine at a different path, use the `-path` argument.

Configure the secrets engine with account credentials and/or scopes:

```text
$ bao write gcpkms/config \
    credentials=@my-credentials.json
Success! Data written to: gcpkms/config
```

If you are running OpenBao from inside [Google Compute Engine][gce] or [Google
Kubernetes Engine][gke], the instance or pod service account can be used in
place of specifying the credentials JSON file. For more information on
authentication, see the [authentication section](#authentication) below.

Create a Google Cloud KMS key:

```text
$ bao write gcpkms/keys/my-key \
  key_ring=projects/my-project/locations/my-location/keyRings/my-keyring \
  rotation_period=72h
```

The `key_ring` parameter is specified in the following format:

```text
projects/<project>/locations/<location>/keyRings/<keyring>
```

where:

- `<project>` - the name of the GCP project (e.g. "my-project")
- `<location>` - the location of the KMS key ring (e.g. "us-east1", "global")
- `<keyring>` - the name of the KMS key ring (e.g. "my-keyring")

## Usage

After the secrets engine is configured and a user/machine has a OpenBao token with
the proper permission, it can be used to encrypt, decrypt, and manage keys. The
following sections describe the different ways in which keys can be managed.

### Symmetric Encryption/Decryption

This section describes using a Cloud KMS key for symmetric
encryption/decryption. This is one of the most common types of encryption.
Google Cloud manages the key ring which is used to encrypt and decrypt data.

<table>
  <thead>
    <tr>
      <th>Purpose</th>
      <th>Supported Algorithms</th>
    </tr>
    <tr>
      <td valign="top">
        <code>encrypt_decrypt</code>
      </td>
      <td valign="top">
        <code>symmetric_encryption</code>
      </td>
    </tr>
  </thead>
</table>

Create or use an existing key eligible for symmetric encryption/decryption:

```text
$ bao write gcpkms/keys/my-key \
    key_ring=projects/.../my-keyring \
    purpose=encrypt_decrypt \
    algorithm=symmetric_encryption
```

Encrypt plaintext data using the `/encrypt` endpoint with a named key:

```text
$ bao write gcpkms/encrypt/my-key plaintext="hello world"
Key            Value
---            -----
ciphertext     CiQAuMv0lTiKjrF43Lgr4...
key_version    1
```

Unlike OpenBao's transit backend, plaintext data does not need to be base64
encoded. The endpoint will automatically convert data.

Note that OpenBao is not _storing_ this data. The caller is responsible for
storing the resulting ciphertext.

Decrypt ciphertext using the `/decrypt` endpoint with a named key:

```text
$ bao write gcpkms/decrypt/my-key ciphertext=CiQAuMv0lTiKjrF43Lgr4...
Key          Value
---          -----
plaintext    hello world
```

For easier scripting, it is also possible to extract the plaintext directly:

```text
$ bao write -field=plaintext gcpkms/decrypt/my-key ciphertext=CiQAuMv0lTiKjrF43Lgr4...
hello world
```

Rotate the underlying encryption key. This will generate a new crypto key
version on Google Cloud KMS and set that version as the active key.

```text
$ bao write -f gcpkms/keys/rotate/my-key
WARNING! The following warnings were returned from OpenBao:

  * The crypto key version was rotated successfully, but it can take up to 2
  hours for the new crypto key version to become the primary. In practice, it
  is usually much shorter. Be sure to issue a read operation and verify the
  key version if you require new data to be encrypted with this key.

Key            Value
---            -----
key_version    2
```

As the message says, rotation is not immediate. Depending on a number of
factors, the propagation of the new key can take quite some time. If you
have a need to immediately encrypt data with this new key, query the API to
wait for the key to become the primary. Alternatively, you can specify the
`key_version` parameter to lock to the exact key for use with encryption.

Re-encrypt already-encrypted ciphertext to be encrypted with a new version of
the crypto key. OpenBao will decrypt the value using the appropriate key in the
keyring and then encrypt the resulting plaintext with the newest key in the
keyring.

```text
$ bao write gcpkms/reencrypt/my-key ciphertext=CiQAuMv0lTiKjrF43Lgr4...
Key            Value
---            -----
ciphertext     CiQAuMv0lZTTozQA/ElqM...
key_version    2
```

This process **does not** reveal the plaintext data. As such, a OpenBao policy
could grant an untrusted process the ability to re-encrypt ciphertext data,
since the process would not be able to get access to the plaintext data.

Trim old key versions by deleting Cloud KMS crypto key versions that are
older than the `min_version` allowed on the key.

```text
$ bao write gcpkms/keys/config/my-key min_version=10
```

Then delete all keys older than version 10. This will make it impossible to
encrypt, decrypt, or sign values with the older key by conventional means.

```text
$ bao write -f gcpkms/keys/trim/my-key
```

Delete the key to delete all key versions and OpenBao's record of the key.

```text
$ bao delete gcpkms/keys/my-key
```

This will make it impossible to encrypt, decrypt, or sign values by
conventional means.

### Asymmetric decryption

This section describes using a Cloud KMS key for asymmetric decryption. In this
model Google Cloud manages the key ring and exposes the public key via an API
endpoint. The public key is used to encrypt data offline and produce ciphertext.
When the plaintext is desired, the user submits the ciphertext to Cloud KMS
which decrypts the value using the corresponding private key.

<table>
  <thead>
    <tr>
      <th>Purpose</th>
      <th>Supported Algorithms</th>
    </tr>
    <tr>
      <td valign="top">
        <code>asymmetric_decrypt</code>
      </td>
      <td valign="top">
        <code>rsa_decrypt_oaep_2048_sha256</code>
        <br />
        <code>rsa_decrypt_oaep_3072_sha256</code>
        <br />
        <code>rsa_decrypt_oaep_4096_sha256</code>
      </td>
    </tr>
  </thead>
</table>

Create or use an existing key eligible for symmetric encryption/decryption:

```text
$ bao write gcpkms/keys/my-key \
    key_ring=projects/.../my-keyring \
    purpose=asymmetric_decrypt \
    algorithm=rsa_decrypt_oaep_4096_sha256
```

Retrieve the public key from Cloud KMS:

```text
$ gcloud kms keys versions get-public-key <crypto-key-version> \
    --location <location> \
    --keyring <key-ring> \
    --key <key> \
    --output-file ~/mykey.pub
```

Encrypt plaintext data with the public key. Note this varies widely between
programming languages. The following example uses OpenSSL, but you can use your
language's built-ins as well.

```text
$ openssl pkeyutl -in ~/my-secret-file \
    -encrypt -pubin \
    -inkey ~/mykey.pub \
    -pkeyopt rsa_padding_mode:oaep \
    -pkeyopt rsa_oaep_md:sha256 \
    -pkeyopt rsa_mgf1_md:sha256
```

Note that this encryption happens offline (meaning outside of OpenBao), and
the encryption is happening with a _public_ key. Only Cloud KMS has the
corresponding _private_ key.

Decrypt ciphertext using the `/decrypt` endpoint with a named key:

```text
$ bao write gcpkms/decrypt/my-key key_version=1 ciphertext=CiQAuMv0lTiKjrF43Lgr4...
Key          Value
---          -----
plaintext    hello world
```

### Asymmetric signing

This section describes using a Cloud KMS key for asymmetric signing. In this
model Google Cloud manages the key ring and exposes the public key via an API
endpoint. A message or digest is signed with the corresponding private key, and
can be verified by anyone with the corresponding public key.

<table>
  <thead>
    <tr>
      <th>Purpose</th>
      <th>Supported Algorithms</th>
    </tr>
    <tr>
      <td valign="top">
        <code>asymmetric_sign</code>
      </td>
      <td valign="top">
        <code>rsa_sign_pss_2048_sha256</code>
        <br />
        <code>rsa_sign_pss_3072_sha256</code>
        <br />
        <code>rsa_sign_pss_4096_sha256</code>
        <br />
        <code>rsa_sign_pkcs1_2048_sha256</code>
        <br />
        <code>rsa_sign_pkcs1_3072_sha256</code>
        <br />
        <code>rsa_sign_pkcs1_4096_sha256</code>
        <br />
        <code>ec_sign_p256_sha256</code>
        <br />
        <code>ec_sign_p384_sha384</code>
      </td>
    </tr>
  </thead>
</table>

Create or use an existing key eligible for symmetric encryption/decryption:

```text
$ bao write gcpkms/keys/my-key \
    key_ring=projects/.../my-keyring \
    purpose=asymmetric_sign \
    algorithm=ec_sign_p384_sha384
```

Calculate the base64-encoded binary digest. Use the hashing algorithm that
corresponds to they key type:

```text
$ export DIGEST=$(openssl dgst -sha384 -binary /my/file | base64)
```

Ask Cloud KMS to sign the digest:

```text
$ bao write gcpkms/sign/my-key key_version=1 digest=$DIGEST
Key          Value
---          -----
signature    MGYCMQDbOS2462SKMsGdh2GQ...
```

Verify the signature of the digest:

```text
$ bao write gcpkms/verify/my-key key_version=1 digest=$DIGEST signature=$SIGNATURE
Key      Value
---      -----
valid    true
```

Note: it is also possible to verify this signature without OpenBao. Download
the public key from Cloud KMS, and use a tool like OpenSSL or your
programming language primitives to verify the signature.

## Authentication

The Google Cloud KMS OpenBao secrets backend uses the official Google Cloud Golang
SDK. This means it supports the common ways of [providing credentials to Google
Cloud][cloud-creds]. In addition to specifying `credentials` directly via OpenBao
configuration, you can also get configuration from the following values **on the
OpenBao server**:

1. The environment variables `GOOGLE_APPLICATION_CREDENTIALS`. This is specified
   as the **path** to a Google Cloud credentials file, typically for a service
   account. If this environment variable is present, the resulting credentials are
   used. If the credentials are invalid, an error is returned.

2. Default instance credentials. When no environment variable is present, the
   default service account credentials are used. This is useful when running OpenBao
   on [Google Compute Engine][gce] or [Google Kubernetes Engine][gke]

For more information on service accounts, please see the [Google Cloud Service
Accounts documentation][service-accounts].

To use this secrets engine, the service account must have the following
minimum scope(s):

```text
https://www.googleapis.com/auth/kms
```

### Required permissions

The credentials given to OpenBao must have the following role:

```text
roles/cloudkms.admin
```

If OpenBao will not be creating keys, you can reduce the permissions. For example,
to create keys out of band and have OpenBao manage the encryption/decryption, you
only need the following permissions:

```text
roles/cloudkms.cryptoKeyEncrypterDecrypter
```

To sign and verify, you only need the following permissions:

```text
roles/cloudkms.signerVerifier
```

For more information, please see the [Google Cloud KMS IAM documentation][kms-iam].

## FAQ

**How is this different than OpenBao's transit secrets engine?**<br />
OpenBao's [transit][openbao-transit] secrets engine uses in-memory keys to
encrypt/decrypt keys. In general it will be faster and more performant. However,
users who need physical, off-site, or out-of-band key management can use the
[Google Cloud KMS][kms] secrets engine to get those benefits while leveraging
OpenBao's policy and identity system.

**Can OpenBao use an existing KMS key?**<br />
You can use the `/register` endpoint to configure OpenBao to talk to an existing
Google Cloud KMS key. As long as the IAM permissions are correct, OpenBao will be
able to encrypt/decrypt data and rotate the key. See the [api][api] for more
information.

**Can this be used with a hardware key like an HSM?**<br />
Yes! You can set `protection_level` to "hsm" when creating a key, or use an
existing Cloud KMS key that is backed by an HSM.

**How much does this cost?**<br />
The plugin is free and open source. KMS costs vary by key type and the number of
operations. Please see the [Cloud KMS pricing page][kms-pricing] for more
details.


## API

The Google Cloud KMS secrets engine has a full HTTP API. Please see the
[Google Cloud KMS secrets engine API docs][api] for more details.

[api]: api.md
[cloud-creds]: https://cloud.google.com/docs/authentication/production#providing_credentials_to_your_application
[gce]: https://cloud.google.com/compute/
[gke]: https://cloud.google.com/kubernetes-engine/
[kms]: https://cloud.google.com/kms
[kms-iam]: https://cloud.google.com/kms/docs/reference/permissions-and-roles
[kms-pricing]: https://cloud.google.com/kms/pricing
[service-accounts]: https://cloud.google.com/compute/docs/access/service-accounts
[openbao-transit]: https://openbao.org/docs/secrets/transit
