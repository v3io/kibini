package main

import (
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/iguazio/kibini/core"
	"github.com/iguazio/kibini/logger"
)

var (
	app       = kingpin.New("kibini", "Like a really bad Kibana if Kibana were any good").DefaultEnvars()
	appQuiet  		= app.Flag("quiet", "Don't log to stdout").Short('q').Bool()
	appLogDir  		= app.Flag("log-dir", "Where to look for platform logs").Required().String()
	appFormattedLogDir  	= app.Flag("formatted-log-dir", "Where to put the formatted logs").Required().String()
	appServices  		= app.Flag("services", "Process only these services").String()
	appNoServices  		= app.Flag("no-services", "Process all but these services").String()
)

func main() {
	app.Version("v0.0.1")

	// create a logger
	logger := logging.NewClient("kibini", ".", "log.txt", false)

	if *appQuiet {
		logger.SetLevel(logging.Error)
	} else {
		logger.SetLevel(logging.Debug)
	}

	// create kibini
	kibini := core.NewKibini(logger)

	// holds the result of the op
	var err error

	// parse the args, run the subcommand
	kingpin.MustParse(app.Parse(os.Args[1:]))

	err = kibini.ProcessLogs(*appLogDir,
		*appFormattedLogDir,
		*appServices,
		*appNoServices)

	select{}

	if err != nil {
		os.Exit(1)
	}

	os.Exit(0)
}
