package passthrough

import (
	"context"
	"fmt"

	"github.com/puppetlabs/leg/instrumentation/alerts/trackers"
)

type Reporter struct {
	c     *Capturer
	err   error
	trace bool
	fs    *trackers.Trace
	tags  []trackers.Tag
}

func (r Reporter) WithNewTrace() trackers.Reporter {
	return &Reporter{
		c:     r.c,
		err:   r.err,
		trace: true,
		fs:    r.fs,
		tags:  append([]trackers.Tag{}, r.tags...),
	}
}

func (r Reporter) WithTrace(t *trackers.Trace) trackers.Reporter {
	return &Reporter{
		c:     r.c,
		err:   r.err,
		trace: true,
		fs:    t,
		tags:  append([]trackers.Tag{}, r.tags...),
	}
}

func (r Reporter) WithTags(tags ...trackers.Tag) trackers.Reporter {
	return &Reporter{
		c:     r.c,
		err:   r.err,
		trace: r.trace,
		fs:    r.fs,
		tags:  append(append([]trackers.Tag{}, r.tags...), tags...),
	}
}

func (r Reporter) AsWarning() trackers.Reporter {
	return &Reporter{
		c:     r.c,
		err:   r.err,
		trace: r.trace,
		fs:    r.fs,
		tags:  append([]trackers.Tag{}, r.tags...),
	}
}

func (r Reporter) Report(ctx context.Context) <-chan error {
	if r.err == nil {
		ch := make(chan error, 1)
		ch <- nil
		return ch
	}

	// TODO Improve message format
	message := fmt.Sprintf("Error: %v\nUser: %v\nTags: %v\nPackages: %v\n",
		r.err.Error(), r.c.user, r.c.tags, r.c.appPackages)

	if r.trace {
		gfs := r.fs.Frames()
		for {
			gf, more := gfs.Next()
			if !more {
				break
			}

			if gf.Func == nil {
				continue
			}

			message += fmt.Sprintf("%v %v %v %v\n", gf.PC, gf.Function, gf.File, gf.Line)
		}
	}

	fmt.Println(message)

	ch := make(chan error, 1)
	ch <- nil
	return ch
}

func (r Reporter) ReportSync(ctx context.Context) error {
	return <-r.Report(ctx)
}
