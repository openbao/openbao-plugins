package noop

import (
	"net/http"

	"github.com/puppetlabs/leg/instrumentation/alerts/internal/httputil"
	"github.com/puppetlabs/leg/instrumentation/alerts/trackers"
)

type Middleware struct {
	c *Capturer
}

func (m Middleware) WithTags(tags ...trackers.Tag) trackers.Middleware {
	return m
}

func (m Middleware) WithUser(u trackers.User) trackers.Middleware {
	return m
}

func (m Middleware) Wrap(target http.Handler) http.Handler {
	return httputil.Wrap(target, httputil.WrapStatic(m.c))
}
