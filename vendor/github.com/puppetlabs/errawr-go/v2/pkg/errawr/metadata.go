package errawr

import "net/http"

type HTTPMetadata interface {
	Status() int
	Headers() http.Header
}

type Metadata interface {
	HTTP() (HTTPMetadata, bool)
}
