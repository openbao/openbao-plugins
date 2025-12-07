package trackers

import "net/http"

type Middleware interface {
	WithTags(tags ...Tag) Middleware
	WithUser(u User) Middleware

	Wrap(target http.Handler) http.Handler
}
