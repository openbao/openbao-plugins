package impl

import (
	"fmt"
	"strings"

	"github.com/puppetlabs/errawr-go/v2/pkg/errawr"
)

func Copy(e errawr.Error) *Error {
	if ce, ok := e.(*Error); ok {
		return ce
	}

	code := strings.TrimPrefix(e.Code(), fmt.Sprintf("%s_%s_", e.Domain().Key(), e.Section().Key()))

	arguments := e.Arguments()
	eas := make(ErrorArguments, len(arguments))
	for name, value := range arguments {
		eas[name] = &ErrorArgument{
			Value: value,
		}
	}

	var eis ErrorItems
	if items, ok := e.Items(); ok {
		eis = make(ErrorItems, len(items))
		for path, err := range items {
			eis[path] = err
		}
	}

	metadata := &ErrorMetadata{}
	if hm, ok := e.Metadata().HTTP(); ok {
		metadata.HTTPErrorMetadata = &HTTPErrorMetadata{
			ErrorStatus:  hm.Status(),
			ErrorHeaders: HTTPErrorMetadataHeaders(hm.Headers()),
		}
	}

	return &Error{
		Version: errawr.Version,
		ErrorDomain: &ErrorDomain{
			Key:   e.Domain().Key(),
			Title: e.Domain().Title(),
		},
		ErrorSection: &ErrorSection{
			Key:   e.Section().Key(),
			Title: e.Section().Title(),
		},
		ErrorCode:  code,
		ErrorTitle: e.Title(),
		ErrorDescription: &ErrorDescription{
			Friendly:  e.Description().Friendly(),
			Technical: e.Description().Technical(),
		},
		ErrorArguments:   eas,
		ErrorItems:       eis,
		ErrorMetadata:    metadata,
		ErrorSensitivity: e.Sensitivity(),

		causes: e.Causes(),
		buggy:  e.IsBug(),
	}
}
