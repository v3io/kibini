package main

import (
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/iguazio/kibini/core"
	"github.com/iguazio/kibini/logger"
	"path/filepath"
)

var (
	app             = kingpin.New("kibini", "Like a really bad Kibana if Kibana were any good").DefaultEnvars()
	appQuiet        = app.Flag("quiet", "Don't log to stdout").Short('q').Bool()
	appInputPath    = app.Flag("input-path", "Where to look for platform logs").Default(".").String()
	appInputFollow  = app.Flag("follow", "Tail -f the log files").Short('f').Bool()
	appOutputPath   = app.Flag("output-path", "Where to output formatted log files").String()
	appOutputMode   = app.Flag("output-mode", "single: merge all logs; per: one formatted per input").Default("per").Enum("single", "per")
	appOutputStdout = app.Flag("stdout", "Output to stdout (output-mode must be 'single')").Bool()
	appServices     = app.Flag("services", "Process only these services").String()
	appNoServices   = app.Flag("no-services", "Process all but these services").String()
	appSingleFile	= app.Arg("filename", "Format only the given filename").Default("\000").String()
)

func getOutputMode(outputModeString string) core.OutputMode {
	return map[string]core.OutputMode{
		"single": core.OutputModeSingle,
		"per":    core.OutputModePer,
	}[outputModeString]
}

func augmentArguments() {

	// if stdout is set, enforce single mode since stdout doesn't make sense we you do "per"
	if *appOutputStdout {
		*appOutputMode = "single"
	}

	// if user didn't pass output path and stdout is disabled, take input path
	// and shove into output path
	if *appOutputPath == "" && !*appOutputStdout {
		*appOutputPath = *appInputPath

		// if output mode is single - add a default merged name because path needs
		// to contain a file name and input path is always a dir
		if *appOutputMode == "single" {
			*appOutputPath = filepath.Join(*appOutputPath, "merged.log.fmt")
		}
	}
}

func main() {
	app.Version("v0.0.5")

	// create a logger
	logger := logging.NewClient("kibini", ".", "kibini.log.txt", true)

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

	// do argument augmentation
	augmentArguments()

	err = kibini.ProcessLogs(*appInputPath,
		*appInputFollow,
		*appOutputPath,
		getOutputMode(*appOutputMode),
		*appOutputStdout,
		*appServices,
		*appNoServices,
		*appSingleFile)

	if err != nil {
		os.Exit(1)
	}

	os.Exit(0)
}
