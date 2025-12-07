package impl

import (
	"fmt"

	"github.com/puppetlabs/errawr-go/v2/pkg/errawr"
	"github.com/puppetlabs/errawr-go/v2/pkg/golang"
)

type ErrorDomain struct {
	Key   string
	Title string
}

type ErrorSection struct {
	Key   string
	Title string
}

type ErrorDescription struct {
	Friendly  string
	Technical string
}

type Error struct {
	Version          uint64
	ErrorDomain      *ErrorDomain
	ErrorSection     *ErrorSection
	ErrorCode        string
	ErrorTitle       string
	ErrorDescription *ErrorDescription
	ErrorArguments   ErrorArguments
	ErrorItems       ErrorItems
	ErrorMetadata    *ErrorMetadata
	ErrorSensitivity errawr.ErrorSensitivity

	causes []errawr.Error
	buggy  bool
}

func (e Error) Domain() errawr.ErrorDomain {
	return &ErrorDomainRepr{Delegate: e.ErrorDomain}
}

func (e Error) Section() errawr.ErrorSection {
	return &ErrorSectionRepr{Delegate: e.ErrorSection}
}

func (e Error) Code() string {
	return e.ErrorCode
}

func (e Error) ID() string {
	return fmt.Sprintf(`%s_%s_%s`, e.ErrorDomain.Key, e.ErrorSection.Key, e.ErrorCode)
}

func (e *Error) Is(id string) bool {
	return e != nil && e.ID() == id
}

func (e Error) Title() string {
	return e.ErrorTitle
}

func (e Error) Description() errawr.ErrorDescription {
	return &UnformattedErrorDescription{e.ErrorDescription}
}

func (e *Error) FormattedDescription() errawr.ErrorDescription {
	return &FormattedErrorDescription{delegate: e}
}

func (e Error) Arguments() map[string]interface{} {
	m := make(map[string]interface{})
	for k, a := range e.ErrorArguments {
		m[k] = a.Value
	}

	return m
}

func (e Error) ArgumentDescription(name string) string {
	argument, ok := e.ErrorArguments[name]
	if !ok {
		return ""
	}

	return argument.Description
}

func (e Error) Metadata() errawr.Metadata {
	if e.ErrorMetadata == nil {
		return &ErrorMetadata{}
	}

	return e.ErrorMetadata
}

func (e Error) Bug() errawr.Error {
	e.buggy = true
	return e.WithSensitivity(errawr.ErrorSensitivityBug)
}

func (e *Error) IsBug() bool {
	return e != nil && e.buggy
}

func (e Error) Items() (map[string]errawr.Error, bool) {
	return e.ErrorItems, e.ErrorItems != nil
}

func (e Error) WithSensitivity(sensitivity errawr.ErrorSensitivity) errawr.Error {
	if sensitivity > e.ErrorSensitivity {
		e.ErrorSensitivity = sensitivity
	}

	return &e
}

func (e Error) Sensitivity() errawr.ErrorSensitivity {
	return e.ErrorSensitivity
}

func (e Error) WithCause(cause error) errawr.Error {
	ce, ok := cause.(errawr.Error)
	if !ok {
		ce = golang.NewError(cause)
	}

	if ce.IsBug() {
		e.buggy = true
	}

	e.causes = append([]errawr.Error{}, e.causes...)
	e.causes = append(e.causes, ce)
	return &e
}

func (e Error) Causes() []errawr.Error {
	return e.causes
}

func (e Error) Error() string {
	var buggy string
	if e.IsBug() {
		buggy = " (BUG)"
	}

	repr := fmt.Sprintf(`%s%s: %s`, e.Code(), buggy, e.FormattedDescription().Technical())
	for _, cause := range e.Causes() {
		repr += fmt.Sprintf("\n%s", cause.Error())
	}

	return repr
}
