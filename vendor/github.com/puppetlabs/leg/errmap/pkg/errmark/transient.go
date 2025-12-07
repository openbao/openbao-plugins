package errmark

var (
	// Transient is a Marker for indicating that a given error is temporary in
	// nature. This is typically used to ask a client to retry work in the
	// future.
	Transient = NewMarker("transient")

	// RuleMarkedTransient is a Rule that matches an error if that error has
	// been marked Transient.
	RuleMarkedTransient = RuleMarked(Transient)
)

// MarkTransient marks an error as transient.
func MarkTransient(err error) error {
	return Mark(err, Transient)
}

// MarkTransientIf marks an error as transient if the error matches the given
// Rule.
func MarkTransientIf(err error, rule Rule) error {
	return MarkIf(err, Transient, rule)
}

// MarkedTransient returns true if the given error has been marked transient.
func MarkedTransient(err error) bool {
	return Marked(err, Transient)
}
