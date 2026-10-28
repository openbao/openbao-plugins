// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fido2

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
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

func unwrapChallenge(salt *salt.Salt, encodedChallenge string) (string, logical.HTTPCodedError) {
	challenge, err := base64.RawURLEncoding.DecodeString(encodedChallenge)
	if err != nil {
		return "", logical.CodedError(http.StatusBadRequest, "invalid challenge: "+err.Error())
	}

	rawWithExpiry, actualHmac, ok := strings.CutLast(string(challenge), "\x00")
	if !ok || subtle.ConstantTimeCompare([]byte(salt.GetHMAC(rawWithExpiry)), []byte(actualHmac)) == 0 {
		return "", logical.CodedError(http.StatusBadRequest, "invalid challenge")
	}

	expiryStr, raw, ok := strings.Cut(rawWithExpiry, "\x00")
	if !ok {
		return "", logical.CodedError(http.StatusBadRequest, "invalid challenge")
	}
	expiry, err := strconv.ParseInt(expiryStr, 16, 64)
	if err != nil {
		return "", logical.CodedError(http.StatusBadRequest, "invalid challenge: "+err.Error())
	}
	if time.Now().After(time.Unix(expiry, 0)) {
		return "", logical.CodedError(http.StatusBadRequest, "challenge expired")
	}

	return raw, nil
}
