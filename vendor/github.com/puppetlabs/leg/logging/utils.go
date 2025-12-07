package logging

import "strings"

func packageArgs(names []string) []interface{} {
	return []interface{}{
		Ctx{"package": strings.Join(names, "/")},
	}
}
