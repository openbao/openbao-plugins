package impl

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/reflect/raymond"
)

var (
	formatHelpers = map[string]interface{}{
		"em":    formatHelperEm,
		"enum":  formatHelperEnum,
		"join":  formatHelperJoin,
		"pre":   formatHelperPre,
		"quote": formatHelperQuote,
	}
)

func formatWithArguments(template string, arguments map[string]interface{}) string {
	parsed, err := raymond.ParseWithOptions(template, raymond.TemplateOptions{NoEscape: true})
	if err != nil {
		return template
	}

	parsed.RegisterHelpers(formatHelpers)

	result, err := parsed.Exec(arguments)
	if err != nil {
		return template
	}

	return strings.TrimSpace(result)
}

func formatHelperEm(in string) string {
	return fmt.Sprintf("*%s*", strings.Replace(in, "*", `\*`, -1))
}

func formatHelperEnum(context interface{}, options *raymond.Options) string {
	val := reflect.ValueOf(context)
	switch val.Kind() {
	case reflect.Array, reflect.Slice:
		var buf bytes.Buffer
		len := val.Len()

		for i := 0; i < len; i++ {
			data := options.NewDataFrame()

			data.Set("index", i)
			data.Set("first", i == 0)
			data.Set("last", i == len-1)

			buf.WriteString("* ")
			buf.WriteString(options.FnOpts(raymond.FnOptions{
				Context:     val.Index(i).Interface(),
				Data:        data,
				BlockParams: []interface{}{i},
			}))

			if i <= len-2 {
				buf.WriteRune('\n')
			}
		}

		return buf.String()
	default:
		return ""
	}
}

func formatHelperJoin(context interface{}, options *raymond.Options) string {
	val := reflect.ValueOf(context)
	switch val.Kind() {
	case reflect.Array, reflect.Slice:
		var buf bytes.Buffer
		len := val.Len()

		for i := 0; i < len; i++ {
			data := options.NewDataFrame()

			data.Set("index", i)
			data.Set("first", i == 0)
			data.Set("last", i == len-1)

			buf.WriteString(options.FnOpts(raymond.FnOptions{
				Context:     val.Index(i).Interface(),
				Data:        data,
				BlockParams: []interface{}{i},
			}))

			if len > 2 {
				if i <= len-2 {
					buf.WriteString(", ")
				}
				if i == len-2 {
					buf.WriteString("and ")
				}
			} else if len == 2 && i == 0 {
				buf.WriteString(" and ")
			}
		}

		return buf.String()
	default:
		return ""
	}
}

func formatHelperPre(in string) string {
	return fmt.Sprintf("`%s`", strings.Replace(in, "`", "\\`", -1))
}

func formatHelperQuote(in string) string {
	return strconv.Quote(in)
}
