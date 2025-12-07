package impl

import "sync"

type UnformattedErrorDescription struct {
	delegate *ErrorDescription
}

func (ued UnformattedErrorDescription) Friendly() string {
	return ued.delegate.Friendly
}

func (ued UnformattedErrorDescription) Technical() string {
	return ued.delegate.Technical
}

type FormattedErrorDescription struct {
	delegate *Error

	friendly  *string
	technical *string

	mut sync.Mutex
}

func (fed *FormattedErrorDescription) format(target **string, source string) string {
	if *target == nil {
		fed.mut.Lock()
		defer fed.mut.Unlock()

		if *target == nil {
			formatted := formatWithArguments(source, fed.delegate.Arguments())
			*target = &formatted
		}
	}

	return **target
}

func (fed *FormattedErrorDescription) Friendly() string {
	return fed.format(&fed.friendly, fed.delegate.ErrorDescription.Friendly)
}

func (fed *FormattedErrorDescription) Technical() string {
	return fed.format(&fed.technical, fed.delegate.ErrorDescription.Technical)
}
