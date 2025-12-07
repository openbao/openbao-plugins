package request

import "github.com/google/uuid"

type Request struct {
	Identifier string
}

func New() *Request {
	return &Request{
		Identifier: uuid.New().String(),
	}
}
