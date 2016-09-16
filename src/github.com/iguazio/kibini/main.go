package main

import (
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/iguazio/kibini/core"
	"github.com/iguazio/kibini/logger"
)

var (
	app                	= kingpin.New("kibini", "Like a really bad Kibana if Kibana were any good").DefaultEnvars()
	appQuiet           	= app.Flag("quiet", "Don't log to stdout").Short('q').Bool()
	appInputPath          	= app.Flag("input-path", "Where to look for platform logs").Required().String()
	appInputFollow      	= app.Flag("follow", "Tail -f the log files").Short('f').Bool()
	appOutputPath 		= app.Flag("output-path", "Where to output formatted log files").String()
	appOutputMode 		= app.Flag("output-mode", "single: merge all logs; per: one formtatted per input").Default("per").Enum("single", "per")
	appOutputStdout		= app.Flag("stdout", "Output to stdout (output-mode must be 'single'").Bool()
	appServices        	= app.Flag("services", "Process only these services").String()
	appNoServices      	= app.Flag("no-services", "Process all but these services").String()
)

func getOutputMode(outputModeString string) core.OutputMode {
	return map[string]core.OutputMode {
		"single": core.OutputModeSingle,
		"per": core.OutputModePer,
	}[outputModeString]
}

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

	err = kibini.ProcessLogs(*appInputPath,
		*appInputFollow,
		*appOutputPath,
		getOutputMode(*appOutputMode),
		*appOutputStdout,
		*appServices,
		*appNoServices)

	if err != nil {
		os.Exit(1)
	}

	os.Exit(0)
}
