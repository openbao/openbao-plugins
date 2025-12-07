package errmark

var (
	// Short is a Marker for indicating that the representation of an error
	// should be abbreviated. In the case of this package, an error marked Short
	// will have the marker prefixes elided from the error message.
	Short = NewMarker("short")

	// RuleMarkedShort is a Rule that matches an error if that error has been
	// marked Short.
	RuleMarkedShort = RuleMarked(Short)
)

// MarkShort marks an error as short.
func MarkShort(err error) error {
	return Mark(err, Short)
}

// MarkShortIf marks an error as short if the error matches the given Rule.
func MarkShortIf(err error, rule Rule) error {
	return MarkIf(err, Short, rule)
}

// MarkedShort returns true if the given error has been marked short.
func MarkedShort(err error) bool {
	return Marked(err, Short)
}
