package sentry

import (
	"context"

	raven "github.com/getsentry/raven-go"
	"github.com/puppetlabs/leg/instrumentation/alerts/trackers"
)

type Reporter struct {
	c     *Capturer
	err   error
	trace bool
	fs    *trackers.Trace
	tags  []trackers.Tag
	level raven.Severity
}

func (r Reporter) WithNewTrace() trackers.Reporter {
	return &Reporter{
		c:     r.c,
		err:   r.err,
		trace: true,
		fs:    r.fs,
		tags:  append([]trackers.Tag{}, r.tags...),
		level: r.level,
	}
}

func (r Reporter) WithTrace(t *trackers.Trace) trackers.Reporter {
	return &Reporter{
		c:     r.c,
		err:   r.err,
		trace: true,
		fs:    t,
		tags:  append([]trackers.Tag{}, r.tags...),
		level: r.level,
	}
}

func (r Reporter) WithTags(tags ...trackers.Tag) trackers.Reporter {
	return &Reporter{
		c:     r.c,
		err:   r.err,
		trace: r.trace,
		fs:    r.fs,
		tags:  append(append([]trackers.Tag{}, r.tags...), tags...),
		level: r.level,
	}
}

func (r Reporter) AsWarning() trackers.Reporter {
	return &Reporter{
		c:     r.c,
		err:   r.err,
		trace: r.trace,
		fs:    r.fs,
		tags:  append([]trackers.Tag{}, r.tags...),
		level: raven.WARNING,
	}
}

func (r Reporter) Report(ctx context.Context) <-chan error {
	if r.err == nil {
		ch := make(chan error, 1)
		ch <- nil
		return ch
	}

	p := raven.NewPacket(r.err.Error())
	p.AddTags(tagsToSentryTags(r.c.tags))
	p.Level = r.level

	if r.trace {
		var frames []*raven.StacktraceFrame

		gfs := r.fs.Frames()
		for {
			gf, more := gfs.Next()
			if !more {
				break
			}

			if gf.Func == nil {
				continue
			}

			frame := raven.NewStacktraceFrame(gf.PC, gf.Function, gf.File, gf.Line, 3, r.c.appPackages)
			if frame == nil {
				continue
			}

			frames = append(frames, frame)
		}

		var st *raven.Stacktrace
		if len(frames) > 0 {
			// Per the Sentry source code, Sentry wants the frames in the
			// opposite order (oldest to newest).
			for i, j := 0, len(frames)-1; i < j; i, j = i+1, j-1 {
				frames[i], frames[j] = frames[j], frames[i]
			}

			st = &raven.Stacktrace{Frames: frames}
		}

		p.Interfaces = append(p.Interfaces, raven.NewException(r.err, st))
	}

	if r.c.user != nil {
		p.Interfaces = append(p.Interfaces, &raven.User{ID: r.c.user.ID, Email: r.c.user.Email})
	}

	if r.c.http != nil {
		p.Interfaces = append(p.Interfaces, r.c.http)
	}

	_, ch := r.c.client.Capture(p, tagsToSentryTags(r.tags))
	return ch
}

func (r Reporter) ReportSync(ctx context.Context) error {
	return <-r.Report(ctx)
}
