package trackers

import "context"

type Reporter interface {
	// WithNewTrace adds a stack trace at the error location to the report.
	WithNewTrace() Reporter

	// WithTrace adds the given stack trace to the report.
	WithTrace(t *Trace) Reporter

	// WithTags adds the given tags to this error report.
	WithTags(tags ...Tag) Reporter

	// AsWarning reduces the severity of this error to a warning.
	AsWarning() Reporter

	// Report submits this error to the reporting service asynchronously.
	Report(ctx context.Context) <-chan error

	// ReportSync submits this error to the reporting service and waits for a
	// response.
	ReportSync(ctx context.Context) error
}
