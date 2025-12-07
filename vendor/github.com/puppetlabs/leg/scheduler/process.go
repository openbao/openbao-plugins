package scheduler

import (
	"context"

	"github.com/puppetlabs/leg/instrumentation/alerts/trackers"
	logging "github.com/puppetlabs/leg/logging"
	"github.com/puppetlabs/leg/request"
)

// Process is the primary unit of work for external users of the scheduler. It
// can be generalized by the Schedulable interface.
//
// A process may (and should expect to) be run multiple times depending on the
// configuration of the descriptor that emits it.
//
// Errors returned by a process are reported to a handler defined by the error
// behavior of the scheduler executing them.
type Process interface {
	Description() string
	Run(ctx context.Context) error
}

func processError(req *request.Request, p Process, err error) error {
	return &ProcessError{
		Request: req,
		Process: p,
		Cause:   err,
	}
}

// SchedulableProcess makes the given process conform to the Schedulable
// interface.
func SchedulableProcess(p Process) Schedulable {
	return SchedulableFunc(func(ctx context.Context, er ErrorReporter) {
		capturer, _ := trackers.CapturerFromContext(ctx)
		capturer = coalesceCapturer(capturer)

		req := request.New()

		ctx = request.NewContext(ctx, req)
		ctx = logging.NewContext(ctx, "request", req.Identifier)

		log(ctx).Debug("process running", "description", p.Description())

		err := capturer.Try(ctx, func(ctx context.Context) {
			if err := p.Run(ctx); err != nil {
				log(ctx).Warn("process failed", "error", err)

				er.Put(processError(req, p, err))
				capturer.Capture(err).AsWarning().Report(ctx)
			} else {
				log(ctx).Debug("process complete")
			}
		})
		if err != nil {
			log(ctx).Crit("process panic()!", "error", err)
			er.Put(processError(req, p, coercePanic(err)))
		}
	})
}

// ProcessFunc converts an arbitrary function to a process.
type ProcessFunc func(ctx context.Context) error

var _ Process = ProcessFunc(nil)

// Description of an arbitrary function is always "<anonymous>" unless provided
// by DescribeProcessFunc.
func (ProcessFunc) Description() string {
	return "<anonymous>"
}

// Run calls the underlying function.
func (p ProcessFunc) Run(ctx context.Context) error {
	return p(ctx)
}

type describedProcessFunc struct {
	desc string
	fn   ProcessFunc
}

var _ Process = &describedProcessFunc{}

func (p describedProcessFunc) Description() string {
	return p.desc
}

func (p describedProcessFunc) Run(ctx context.Context) error {
	return p.fn.Run(ctx)
}

// DescribeProcessFunc associates the given description with the arbitrary
// process function and returns it as a process.
func DescribeProcessFunc(desc string, fn ProcessFunc) Process {
	return &describedProcessFunc{desc: desc, fn: fn}
}

// Descriptor provides a way to emit work to a scheduler. Descriptors are
// provided to segment lifecycles and are monitored by them for completion.
//
// Errors returned by descriptors are reported to a handler defined by the error
// behavior of the scheduler executing them.
type Descriptor interface {
	Run(ctx context.Context, pc chan<- Process) error
}

// DescriptorFunc converts an arbitrary function to a descriptor.
type DescriptorFunc func(ctx context.Context, pc chan<- Process) error

var _ Descriptor = DescriptorFunc(nil)

// Run calls the underlying fuction.
func (d DescriptorFunc) Run(ctx context.Context, pc chan<- Process) error {
	return d(ctx, pc)
}

func descriptorError(desc Descriptor, err error) error {
	return &DescriptorError{
		Descriptor: desc,
		Cause:      err,
	}
}

// SchedulableDescriptor adapts a descriptor to the Schedulable interface.
func SchedulableDescriptor(d Descriptor, pc chan<- Process) Schedulable {
	return SchedulableFunc(func(ctx context.Context, er ErrorReporter) {
		if err := d.Run(ctx, pc); err != nil {
			log(ctx).Warn("descriptor ended with error", "error", err)
			er.Put(descriptorError(d, err))
		}
	})
}

type manySchedulableDescriptor struct {
	delegates []Descriptor
	pc        chan<- Process
}

func (msd *manySchedulableDescriptor) Len() int {
	return len(msd.delegates)
}

func (msd *manySchedulableDescriptor) Run(ctx context.Context, i int, er ErrorReporter) {
	SchedulableDescriptor(msd.delegates[i], msd.pc).Run(ctx, er)
}

// ManySchedulableDescriptor adapts a slice of descriptors to the
// ManySchedulable interface.
func ManySchedulableDescriptor(ds []Descriptor, pc chan<- Process) ManySchedulable {
	return &manySchedulableDescriptor{
		delegates: ds,
		pc:        pc,
	}
}
