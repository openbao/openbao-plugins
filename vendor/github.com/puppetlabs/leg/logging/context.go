package logging

import "context"

type contextKey int

const (
	argsContextKey contextKey = iota
)

func NewContext(ctx context.Context, args ...interface{}) context.Context {
	args = normalize(args)

	next, ok := ctx.Value(argsContextKey).([]interface{})
	if ok {
		next = append([]interface{}{}, next...)
		next = append(next, args...)
	} else {
		next = args
	}

	return context.WithValue(ctx, argsContextKey, next)
}

func contextArgs(ctx context.Context) []interface{} {
	args, _ := ctx.Value(argsContextKey).([]interface{})
	return args
}
