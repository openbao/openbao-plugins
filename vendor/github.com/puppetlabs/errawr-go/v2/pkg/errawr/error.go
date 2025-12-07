package errawr

// ErrorDomain is the accessor type for error domains.
type ErrorDomain interface {
	// Key is the unique short representation of this domain.
	Key() string

	// Title is the human-readable representation of this domain.
	Title() string

	// Is tests whether this domain key is equivalent to the passed key.
	Is(key string) bool
}

// ErrorSection is the accessor type for error section.
type ErrorSection interface {
	// Key is the unique short representation of this section.
	Key() string

	// Title is the human-readable representation of this section.
	Title() string

	// Is tests whether this section key is equivalent to the passed key.
	Is(key string) bool
}

// ErrorDescription is the accessor type for error descriptions in different
// states.
type ErrorDescription interface {
	// Friendly is an end-user-friendly description of an error.
	Friendly() string

	// Technical is a description of an error suitable for sending to a support
	// person.
	Technical() string
}

// ErrorSensitivity is how sensitive an error is to being revealed outside of
// the domain in which it was generated.
type ErrorSensitivity int

const (
	// ErrorSensitivityNone is the default error sensitivity. These errors can
	// be presented anywhere, even to third parties.
	ErrorSensitivityNone ErrorSensitivity = 0

	// ErrorSensitivityEdge restricts errors to components that are part of the
	// same system. Edge-sensitive errors may cross error domains, but may not
	// be propagated to third-party components.
	ErrorSensitivityEdge ErrorSensitivity = 100

	// ErrorSensitivityBug restricts errors to be reasonably displayed in
	// certain intra-system interfaces, but which may be further restricted than
	// edge errors.
	ErrorSensitivityBug ErrorSensitivity = 200

	// ErrorSensitivityAll restricts errors to only being shown within the same
	// domain.
	ErrorSensitivityAll ErrorSensitivity = 1000
)

// Error is the type of all user-facing errors.
type Error interface {
	error

	// Domain is the broad domain for this error.
	Domain() ErrorDomain

	// Section is the domain-specific section for this error.
	Section() ErrorSection

	// Code is the name for this error.
	Code() string

	// ID is the complete identifier for this error.
	ID() string

	// Is tests whether this error's ID is equivalent to the passed ID.
	Is(id string) bool

	// Title is the short title for this error.
	Title() string

	// Description returns the unformatted descriptions of this error.
	Description() ErrorDescription

	// FormattedDescription returns an ASCII-printable formatted description
	// of this error.
	FormattedDescription() ErrorDescription

	// Arguments is the read-only argument map for this error.
	Arguments() map[string]interface{}

	// ArgumentDescription returns a description of the given argument, if
	// available.
	ArgumentDescription(name string) string

	// Metadata returns additional environment-specific information for this error.
	Metadata() Metadata

	// Bug causes this error to become a buggy error. Buggy errors are subject
	// to additional reporting. Buggy errors implicitly have a sensitivity of at
	// least ErrorSensitivityBug.
	Bug() Error

	// IsBug returns true if this error is buggy.
	IsBug() bool

	// Items returns the errors contained by this error. If this error does not
	// have the container trait, this method returns false.
	Items() (map[string]Error, bool)

	// WithSensitivity sets this error's sensitivity. Subsequent calls to this
	// method can only further restrict sensitivity, not make the error less
	// sensitive.
	WithSensitivity(sensitivity ErrorSensitivity) Error

	// Sensitivity returns the sensitivity for this error.
	Sensitivity() ErrorSensitivity

	// WithCause causes this error to be caused by the given error. If it is
	// already caused by another error, it will be caused by both errors.
	WithCause(cause error) Error

	// Causes returns the list of causes for this error.
	Causes() []Error
}
