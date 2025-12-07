package sentry

import (
	"net/http"

	"github.com/puppetlabs/leg/instrumentation/alerts/internal/httputil"
	"github.com/puppetlabs/leg/instrumentation/alerts/trackers"
)

type Middleware struct {
	c *Capturer
}

func (m Middleware) WithTags(tags ...trackers.Tag) trackers.Middleware {
	return &Middleware{
		c: m.c.withTags(tags),
	}
}

func (m Middleware) WithUser(u trackers.User) trackers.Middleware {
	return &Middleware{
		c: m.c.withUser(u),
	}
}

func (m Middleware) Wrap(target http.Handler) http.Handler {
	return httputil.Wrap(target, func(r *http.Request) trackers.Capturer {
		return m.c.withHTTP(r)
	})
}
