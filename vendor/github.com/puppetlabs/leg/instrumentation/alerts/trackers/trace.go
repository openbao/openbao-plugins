package trackers

import "runtime"

const (
	TraceStackDepth = 128
)

type Trace struct {
	pcs []uintptr
}

func (t *Trace) Frames() *runtime.Frames {
	return runtime.CallersFrames(t.pcs)
}

func NewTrace(skip int) *Trace {
	pcs := make([]uintptr, TraceStackDepth)
	runtime.Callers(skip+2, pcs)

	return &Trace{
		pcs: pcs,
	}
}
