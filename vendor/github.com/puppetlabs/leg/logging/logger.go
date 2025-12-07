package logging

import (
	"context"

	"github.com/inconshreveable/log15"
)

type Logger interface {
	Let(ctx ...interface{}) Logger
	With(ctx context.Context) Logger
	At(names ...string) Logger
	Stack() Logger

	Debug(msg string, ctx ...interface{})
	Info(msg string, ctx ...interface{})
	Warn(msg string, ctx ...interface{})
	Error(msg string, ctx ...interface{})
	Crit(msg string, ctx ...interface{})
}

type Ctx log15.Ctx

func (c Ctx) toArray() []interface{} {
	arr := make([]interface{}, len(c)*2)

	i := 0
	for k, v := range c {
		arr[i] = k
		arr[i+1] = v
		i += 2
	}

	return arr
}

func normalize(ctx []interface{}) []interface{} {
	if len(ctx) == 1 {
		if m, ok := ctx[0].(Ctx); ok {
			return m.toArray()
		}

		if m, ok := ctx[0].(log15.Ctx); ok {
			return Ctx(m).toArray()
		}
	}

	return ctx
}
