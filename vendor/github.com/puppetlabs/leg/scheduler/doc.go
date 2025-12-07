/*
Package scheduler provides a managed API to Goroutines using Lifecycles.

The most basic type of management is using the Schedulable interface with a
Scheduler:

    worker := scheduler.SchedulableFunc(func(ctx context.Context, er scheduler.ErrorReporter) {
        for {
            select {
            case <-ctx.Done():
                return
            case <-time.After(100 * time.Millisecond):
                fmt.Println("Mmm... pie.")
            }
        }
    })

    l := scheduler.NewScheduler(scheduler.OneSchedulable(worker))

    sl := l.Start(scheduler.LifecycleStartOptions{})

    time.Sleep(1 * time.Second)

    // Tell the scheduler to start closing.
    sl.Close()

    // Wait for all managed routines to finish.
    <-sl.Done()

Schedulers terminate when all of their children exit.

You can choose from three canned error behaviors for most lifecycles:
ErrorBehaviorDrop, ErrorBehaviorCollect, and ErrorBehaviorTerminate.
ErrorBehaviorDrop ignores errors, allowing the lifecycle to continue executing
normally. ErrorBehaviorCollect stores all errors returned (potentially allowing
for unbounded memory growth, so use with discretion) and provides them when the
lifecycle completes. ErrorBehaviorTerminate causes the lifecycle to close as
soon as it receives an error. You may implement your own error behaviors by
conforming to the ErrorBehavior interface.

If you have a few lifecycles that are parameterized differently and you want to
manage them together, the Parent lifecycle aggregates them and runs them in
parallel.

This package also provides a more sophisticated lifecycle, Segment. A Segment
provides a worker pool and a mechanism for dispatching work. Dispatchers
implement the Descriptor interface and work items implement the Process
interface. The example above could equivalently be written as follows:

    proc := scheduler.ProcessFunc(func(ctx context.Context) error {
        fmt.Println("Mmm... pie.")
        return nil
    })

    l := scheduler.NewSegment(1, []scheduler.Descriptor{
        scheduler.NewIntervalDescriptor(100*time.Millisecond, proc),
    })
    // Start, close, and wait on the lifecycle as before.

Descriptors are particularly useful when asynchronously waiting on events from
external APIs for processing.
*/
package scheduler
