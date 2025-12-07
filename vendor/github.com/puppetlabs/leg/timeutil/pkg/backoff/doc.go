/*
Package backoff provides a set of algorithms for determining how long to wait
when an operation needs to be retried.

A backoff is produced by a generator with zero or more optional rules attached
to it. Rules modify the backoff by, for example, adding jitter, or capping the
duration at a certain amount.

You can create a factory for producing new backoffs (starting at their initial
value) using Build():

    factory := Build(Linear(5 * time.Second))

You can also specify the rules to use:

    factory := Build(
        Linear(5 * time.Second),
        MaxBound(30 * time.Second),
        NonSliding,
    )


Factories are thread-safe, but Backoff instances are not. Once you have a
factory, call New() to get a backoff duration producer:

    backoff, err := factory.New()
    if err != nil { ... }

    for !isOK() {
        wait, err := backoff.Next()
        if err != nil { ... }

        time.Sleep(wait)

        // Do work...
    }

See also the retry package in this module that leverages this backoff package to
make retrying processes easy.
*/
package backoff
