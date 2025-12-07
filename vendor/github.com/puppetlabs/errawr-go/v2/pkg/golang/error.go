package golang

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/puppetlabs/errawr-go/v2/pkg/errawr"
)

type ErrorDomain struct{}

func (ErrorDomain) Key() string {
	return "err"
}

func (ErrorDomain) Title() string {
	return ""
}

func (ed ErrorDomain) Is(key string) bool {
	return key == ed.Key()
}

type ErrorSection struct{}

func (ErrorSection) Key() string {
	return "golang"
}

func (ErrorSection) Title() string {
	return ""
}

func (es ErrorSection) Is(key string) bool {
	return key == es.Key()
}

type UnformattedErrorDescription struct{}

func (UnformattedErrorDescription) Friendly() string {
	return `{{cause}}`
}

func (UnformattedErrorDescription) Technical() string {
	return `{{cause}}`
}

type FormattedErrorDescription struct {
	delegate error
}

func (fed FormattedErrorDescription) Friendly() string {
	return fed.delegate.Error()
}

func (fed FormattedErrorDescription) Technical() string {
	return fed.delegate.Error()
}

type ErrorMetadata struct{}

func (ErrorMetadata) HTTP() (errawr.HTTPMetadata, bool) {
	return nil, false
}

type Error struct {
	delegate error

	causes      []errawr.Error
	buggy       bool
	sensitivity errawr.ErrorSensitivity
}

func (Error) Domain() errawr.ErrorDomain {
	return ErrorDomain{}
}

func (Error) Section() errawr.ErrorSection {
	return ErrorSection{}
}

func (e Error) Code() string {
	ty := reflect.TypeOf(e.delegate)
	for ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}

	code := ty.PkgPath()
	if len(code) > 0 {
		code += "_"
	}
	code += ty.Name()

	mapper := func(r rune) rune {
		if r > unicode.MaxASCII || !unicode.In(r, unicode.Letter, unicode.Digit) {
			return '_'
		}

		return r
	}

	return strings.Map(mapper, code)
}

func (e Error) ID() string {
	return fmt.Sprintf(`%s_%s_%s`, e.Domain().Key(), e.Section().Key(), e.Code())
}

func (e Error) Is(id string) bool {
	return id == e.ID()
}

func (e Error) Title() string {
	return reflect.TypeOf(e.delegate).String()
}

func (Error) Description() errawr.ErrorDescription {
	return UnformattedErrorDescription{}

}

func (e Error) FormattedDescription() errawr.ErrorDescription {
	return FormattedErrorDescription{e.delegate}
}

func (e Error) Arguments() map[string]interface{} {
	return map[string]interface{}{
		"cause": e.delegate.Error(),
	}
}

func (e Error) ArgumentDescription(name string) string {
	return ""
}

func (Error) Metadata() errawr.Metadata {
	return ErrorMetadata{}
}

func (e Error) Bug() errawr.Error {
	e.buggy = true
	return e.WithSensitivity(errawr.ErrorSensitivityBug)
}

func (e *Error) IsBug() bool {
	return e != nil && e.buggy
}

func (e Error) Items() (map[string]errawr.Error, bool) {
	return nil, false
}

func (e Error) WithSensitivity(sensitivity errawr.ErrorSensitivity) errawr.Error {
	if sensitivity > e.sensitivity {
		e.sensitivity = sensitivity
	}

	return &e
}

func (e Error) Sensitivity() errawr.ErrorSensitivity {
	return e.sensitivity
}

func (e Error) WithCause(cause error) errawr.Error {
	ce, ok := cause.(errawr.Error)
	if !ok {
		ce = NewError(cause)
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

func NewError(err error) errawr.Error {
	return &Error{
		delegate: err,

		sensitivity: errawr.ErrorSensitivityEdge,
	}
}
