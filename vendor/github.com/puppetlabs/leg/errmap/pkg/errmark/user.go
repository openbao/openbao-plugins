package errmark

var (
	// User is a Marker for indicating that an error was caused by user action
	// and not by a system error. It is typically used to ask a user to change
	// application input to remedy the problem.
	User = NewMarker("user")

	// RuleMarkedUser is a Rule that matches an error if the error has been
	// marked as a user error.
	RuleMarkedUser = RuleMarked(User)
)

// MarkUser marks an error as a user error.
func MarkUser(err error) error {
	return Mark(err, User)
}

// MarkUserIf marks an error as a user error if the error matches the given
// Rule.
func MarkUserIf(err error, rule Rule) error {
	return MarkIf(err, User, rule)
}

// MarkedUser returns true if the given error has been marked as a user error.
func MarkedUser(err error) bool {
	return Marked(err, User)
}
