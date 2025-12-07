package request

import "context"

type contextKey int

const (
	requestContextKey contextKey = iota
)

func NewContext(ctx context.Context, req *Request) context.Context {
	return context.WithValue(ctx, requestContextKey, req)
}

func FromContext(ctx context.Context) (req *Request, ok bool) {
	req, ok = ctx.Value(requestContextKey).(*Request)
	return
}
