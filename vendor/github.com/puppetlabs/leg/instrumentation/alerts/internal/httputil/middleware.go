package httputil

import (
	"fmt"
	"net/http"

	"github.com/puppetlabs/leg/instrumentation/alerts/trackers"
)

type WrapFunc func(r *http.Request) trackers.Capturer

func WrapStatic(c trackers.Capturer) WrapFunc {
	return func(r *http.Request) trackers.Capturer { return c }
}

func Wrap(target http.Handler, fn WrapFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := fn(r)
		r = r.WithContext(trackers.NewContextWithCapturer(r.Context(), c))

		defer func() {
			var reporter trackers.Reporter

			rv := recover()
			switch rvt := rv.(type) {
			case nil:
				return
			case error:
				reporter = c.Capture(rvt)
			default:
				reporter = c.CaptureMessage(fmt.Sprint(rvt))
			}

			reporter.Report(r.Context())
		}()

		target.ServeHTTP(w, r)
	})
}
