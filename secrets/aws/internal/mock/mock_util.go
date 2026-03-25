package mock_aws

import (
	"encoding/json"

	"go.uber.org/mock/gomock"
)

// Eq is just like [go.uber.org/mock/gomock.Eq], but formats the "want" and
// "got" via [encoding/json.Marshal] because the AWS Input types don't have a
// `String()` method.
func Eq(x any) gomock.Matcher {
	matcher := gomock.Eq(x)

	format := func(x any) string {
		m, err := json.Marshal(x)
		if err != nil {
			panic(err)
		}
		return string(m)
	}

	matcher = gomock.WantFormatter(gomock.StringerFunc(func() string {
		return format(x)
	}), matcher)

	matcher = gomock.GotFormatterAdapter(gomock.GotFormatterFunc(format), matcher)

	return matcher
}
