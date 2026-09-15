# How to onboard a new users

```mermaid
flowchart TD
    P --> ORG_BYOD
    
    ORG_BYOD -- BYOD --> ONLY
    ORG_BYOD -- Organization provided --> ORG

    ONLY -- yes --> SELF
    ONLY -- no --> TOKEN


    P(Provision and new user)
    ORG_BYOD{"Does the user have an organization provided hardware / software or 'Bring your own device' (BYOD)?"}
    ORG[Use attestation]

    ONLY{Does the user already have another auth method?}

    SELF[Use self-service]
    TOKEN[Use <a target="_top" href="#provision-token">token based approach</a>]

    click SELF "#provision-self-service"
```

## Self-Service <a name="provision-self-service"></a>

To allow self-service, an identity admin needs to:

- Find the entity id of the target user (or create an entity, if the user never
  logged-in before.
- Find the fido2 mounts accessor.
- Create an alias for the fido2 mounts accessor mapping to the same entity.

Example when using userpass for user "alice" and mounting the fido2 plugin at "passkey":

```bash
USERPASS_ACCESSOR=$(bao read --field=accessor sys/auth/userpass)
FIDO2_ACCESSOR=$(bao read --field=accessor sys/auth/passkey)

ENTITY_ID=$(bao write --field=id identity/lookup/entity \
    alias_mount_accessor=$USERPASS_ACCESSOR \
    alias_name=alice)

bao write identity/entity-alias \
    canonical_id=$ENTITY_ID \
    mount_accessor=$FIDO2_ACCESSOR \
    name=alice
```

## Single use tokens  <a name="provision-token"></a>
