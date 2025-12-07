package impl

import (
	"net/http"

	"github.com/puppetlabs/errawr-go/v2/pkg/errawr"
)

type HTTPErrorMetadataHeaders map[string][]string

type HTTPErrorMetadata struct {
	ErrorStatus  int
	ErrorHeaders HTTPErrorMetadataHeaders
}

func (hem HTTPErrorMetadata) Status() int {
	return hem.ErrorStatus
}

func (hem HTTPErrorMetadata) Headers() http.Header {
	m := make(http.Header, len(hem.ErrorHeaders))
	for k, vs := range hem.ErrorHeaders {
		m[k] = append([]string{}, vs...)
	}

	return m
}

type ErrorMetadata struct {
	HTTPErrorMetadata *HTTPErrorMetadata
}

func (em ErrorMetadata) HTTP() (errawr.HTTPMetadata, bool) {
	return em.HTTPErrorMetadata, em.HTTPErrorMetadata != nil
}
