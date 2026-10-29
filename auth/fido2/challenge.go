// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fido2

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/hashicorp/go-hclog"
	"github.com/openbao/openbao/sdk/v2/helper/salt"
	"github.com/openbao/openbao/sdk/v2/logical"
)

func wrapChallenge(salt *salt.Salt, rawChallenge string) protocol.URLEncodedBase64 {
	expiry := time.Now().Add(5 * time.Minute).Unix() // https://www.w3.org/TR/webauthn-3/#sctn-timeout-recommended-range

	otpWithExpiry := fmt.Sprintf("%016x\x00%s", expiry, rawChallenge) // we use %016x for the timestamp so that alphabetical sorting gives use timestamp sorting

	hmac := salt.GetHMAC(otpWithExpiry)

	challenge := fmt.Sprintf("%s\x00%s", otpWithExpiry, hmac)
	return protocol.URLEncodedBase64(challenge)
}

func unwrapChallenge(salt *salt.Salt, encodedChallenge string) (string, string, logical.HTTPCodedError) {
	challenge, err := base64.RawURLEncoding.DecodeString(encodedChallenge)
	if err != nil {
		return "", "", logical.CodedError(http.StatusBadRequest, "invalid challenge: "+err.Error())
	}

	rawWithExpiry, actualHmac, ok := strings.CutLast(string(challenge), "\x00")
	if !ok || subtle.ConstantTimeCompare([]byte(salt.GetHMAC(rawWithExpiry)), []byte(actualHmac)) == 0 {
		return "", "", logical.CodedError(http.StatusBadRequest, "invalid challenge")
	}

	expiryStr, raw, ok := strings.Cut(rawWithExpiry, "\x00")
	if !ok {
		return "", "", logical.CodedError(http.StatusBadRequest, "invalid challenge")
	}
	expiry, err := strconv.ParseInt(expiryStr, 16, 64)
	if err != nil {
		return "", "", logical.CodedError(http.StatusBadRequest, "invalid challenge: "+err.Error())
	}
	if time.Now().After(time.Unix(expiry, 0)) {
		return "", "", logical.CodedError(http.StatusBadRequest, "challenge expired")
	}

	return raw, string(challenge), nil
}

func consumeChallenge(ctx context.Context, storage logical.Storage, challenge string) error {
	return logical.WithTransaction(ctx, storage, func(s logical.Storage) error {
		path := escapedPathJoin("v1", "consumed-challenges", challenge)
		val, err := s.Get(ctx, path)
		if err != nil {
			return err
		}
		if val != nil {
			return logical.CodedError(http.StatusForbidden, "challenge reuse detected")
		}

		return s.Put(ctx, &logical.StorageEntry{
			Key:   path,
			Value: []byte("{}"),
		})
	})
}

func cleanupExpiredConsumedChallenges(ctx context.Context, logger hclog.Logger, storage logical.Storage) error {
	prefix := "v1/consumed-challenges/"

	view := logical.NewStorageView(storage, prefix)

	return logical.ScanViewPaginated(ctx, view, logger, logical.DefaultScanViewPageLimit, func(page, index int, entry string) (cont bool, err error) {
		expiryStr, _, ok := strings.Cut(entry, "\x00")
		if ok {
			expiry, err := strconv.ParseInt(expiryStr, 16, 64)
			if err == nil {
				deleteTime := time.Unix(expiry, 0).Add(time.Hour) // add an hour for clock skew
				if time.Now().Before(deleteTime) {
					// we found the first entry we don't want to delete just yet
					// because the entries are sorted, we can abort the search
					return false, nil
				}
			} else {
				logger.Warn("deleting invalid entry: timestamp invalid", "error", err, "path", view.ExpandKey(entry))
			}
		} else {
			logger.Warn("deleting invalid entry: missing null byte", "path", view.ExpandKey(entry))
		}

		err = view.Delete(ctx, entry)
		if err != nil {
			return false, err
		}
		return true, nil
	})
}
