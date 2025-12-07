package logging

import (
	"os"

	"github.com/inconshreveable/log15"
	"github.com/puppetlabs/leg/logging/handler"
)

var (
	rootFormatter = handler.StandardFormatter
	rootHandler   = log15.StreamHandler(os.Stdout, rootFormatter)
	rootLevel     = log15.LvlDebug

	rootLogger log15.Logger
)

func init() {
	setLogger()
}

func setLogger() {
	handler := rootHandler
	handler = log15.LvlFilterHandler(rootLevel, handler)

	logger := log15.New()
	logger.SetHandler(handler)

	rootLogger = logger
}

func SetHandler(in log15.Handler) {
	rootHandler = in
	setLogger()
}

func SetLevel(in log15.Lvl) {
	rootLevel = in
	setLogger()
}
