package handler

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/inconshreveable/log15"
)

const (
	logMsgOffset = 40
	identLen     = 38
)

func formatStandardLogRecord(r *log15.Record) []byte {
	buf := &bytes.Buffer{}

	buf.WriteString(r.Time.Format(time.RFC3339))
	buf.WriteByte(' ')

	switch r.Lvl {
	case log15.LvlDebug:
		buf.WriteString("[debug] ")
	case log15.LvlInfo:
		buf.WriteString("[info]  ")
	case log15.LvlWarn:
		buf.WriteString("[warn]  ")
	case log15.LvlError:
		buf.WriteString("[error] ")
	case log15.LvlCrit:
		buf.WriteString("[crit]  ")
	}

	ctx := make([]string, 0, len(r.Ctx)/2)
	var pkg, ident string

	for i := 0; i < len(r.Ctx); i += 2 {
		k := r.Ctx[i]
		v := r.Ctx[i+1]

		switch k {
		case "package":
			pkg = fmt.Sprintf("%+v", v)
			if len(pkg) > 11 {
				pkg = fmt.Sprintf("[...%s]", pkg[len(pkg)-11:len(pkg)])
			} else {
				pkg = fmt.Sprintf("[%s]", pkg)
			}
		case "request":
			ident = fmt.Sprintf("[%v]", v)
		default:
			ctx = append(ctx, fmt.Sprintf("%v=%v", k, v))
		}
	}

	if len(pkg) > 0 {
		buf.WriteString(fmt.Sprintf("%-14s", pkg))
	} else {
		buf.WriteString(strings.Repeat(" ", 16))
	}

	if len(ident) > 0 {
		buf.WriteByte(' ')
		buf.WriteString(ident)
	} else {
		buf.WriteString(strings.Repeat(" ", identLen+1))
	}

	if len(r.Msg) > 0 {
		buf.WriteByte(' ')
		buf.WriteString(r.Msg)
	}

	if len(ctx) > 0 {
		if len(r.Msg) > logMsgOffset-1 {
			buf.WriteByte(' ')
		} else {
			buf.WriteString(strings.Repeat(" ", logMsgOffset-len(r.Msg)))
		}

		buf.WriteString(strings.Join(ctx, " "))
	}

	buf.WriteByte('\n')

	return buf.Bytes()
}

var (
	StandardFormatter = log15.FormatFunc(formatStandardLogRecord)
)
