package logging

import "context"

type LoggerBuilder interface {
	Let(args ...interface{}) LoggerBuilder
	With(ctx context.Context) LoggerBuilder
	At(names ...string) LoggerBuilder

	Build() Logger
}

type builder struct {
	args []interface{}
}

func (b builder) Let(args ...interface{}) LoggerBuilder {
	if len(args) == 0 {
		return &b
	}

	args = normalize(args)

	next := append([]interface{}{}, b.args...)
	next = append(next, args...)

	return &builder{args: next}
}

func (b builder) With(ctx context.Context) LoggerBuilder {
	return b.Let(contextArgs(ctx)...)
}

func (b builder) At(names ...string) LoggerBuilder {
	return b.Let(packageArgs(names)...)
}

func (b builder) Build() Logger {
	return (&proxy{delegate: rootLogger}).Let(b.args...)
}

func Builder() LoggerBuilder {
	return &builder{}
}
