package fido2

import (
	"errors"
	"net/http"

	"github.com/openbao/openbao/sdk/v2/logical"
)

func asCodedError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := errors.AsType[logical.HTTPCodedError](err); ok {
		return err
	}
	return logical.CodedError(http.StatusInternalServerError, err.Error())
}
