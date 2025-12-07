package logging

import (
	"context"

	"github.com/inconshreveable/log15"
)

type proxy struct {
	delegate log15.Logger
}

func (p proxy) Let(args ...interface{}) Logger {
	return &proxy{delegate: p.delegate.New(normalize(args)...)}
}

func (p proxy) With(ctx context.Context) Logger {
	return p.Let(contextArgs(ctx)...)
}

func (p proxy) At(names ...string) Logger {
	return p.Let(packageArgs(names)...)
}

func (p proxy) Stack() Logger {
	delegate := p.delegate.New()
	delegate.SetHandler(log15.CallerStackHandler("%+v", delegate.GetHandler()))
	return &proxy{delegate: delegate}
}

func (p proxy) Debug(msg string, ctx ...interface{}) {
	p.delegate.Debug(msg, normalize(ctx)...)
}

func (p proxy) Info(msg string, ctx ...interface{}) {
	p.delegate.Info(msg, normalize(ctx)...)
}

func (p proxy) Warn(msg string, ctx ...interface{}) {
	p.delegate.Warn(msg, normalize(ctx)...)
}

func (p proxy) Error(msg string, ctx ...interface{}) {
	p.delegate.Error(msg, normalize(ctx)...)
}

func (p proxy) Crit(msg string, ctx ...interface{}) {
	p.delegate.Crit(msg, normalize(ctx)...)
}
