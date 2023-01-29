package main

import (
	"os"
	"path/filepath"

	"github.com/v3io/kibini/pkg/kibini"
	"github.com/v3io/kibini/pkg/loggerus"

	"github.com/nuclio/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	app             = kingpin.New("kibini", "Like a really bad Kibana if Kibana were any good").DefaultEnvars()
	appQuiet        = app.Flag("quiet", "Don't log to stdout").Short('q').Bool()
	appInputPath    = app.Flag("input-path", "Where to look for platform logs").Default(".").String()
	appInputFollow  = app.Flag("follow", "Tail -f the log files").Short('f').Bool()
	appOutputPath   = app.Flag("output-path", "Where to output formatted log files").String()
	appOutputMode   = app.Flag("output-mode", "single: merge all logs; per: one formatted per input").Default("per").Enum("single", "per")
	appOutputStdout = app.Flag("stdout", "Output to stdout (output-mode must be 'single')").Bool()
	appSingleFile   = app.Arg("filename", "Format only the given filename").Default("\000").String()
	appColorSetting = app.Flag("color", "on: use colors when outputting to tty; off: don't use colors; always: always use color").Default("on").Enum("on", "off", "always")
	appWhoWidth     = app.Flag("who-width", "Set truncate width for 'who' field, default is 45").Default("45").Int()
	appRegex        = app.Flag("regex", "Process only log files that match the given regex").String()
	appNoRegex      = app.Flag("no-regex", "Process all log files expect those who match the given regex").String()
	version         string
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

func run() error {

	// version is being injected by build,
	if version == "" {

		// if non was given (running from IDE or so), fallback to latest
		version = "latest"
	}
	app.Version(version)

	// parse the args, run the subcommand
	kingpin.MustParse(app.Parse(os.Args[1:]))

	// set log level
	logLevel := logrus.DebugLevel
	if *appQuiet {
		logLevel = logrus.ErrorLevel
	}

	// create logger file
	loggerFile, err := os.Create("kibini.log.txt")
	if err != nil {
		return errors.Wrap(err, "Failed to create file")
	}

	// create a logger
	logger, err := loggerus.NewTextLoggerus("kibini", logLevel, loggerFile)
	if err != nil {
		return errors.Wrap(err, "Failed to create logger")
	}

	// create kibini
	kibini := core.NewKibini(logger)

	// do argument augmentation
	augmentArguments()

	return kibini.ProcessLogs(*appInputPath,
		*appInputFollow,
		*appOutputPath,
		getOutputMode(*appOutputMode),
		*appOutputStdout,
		*appRegex,
		*appNoRegex,
		*appSingleFile,
		*appColorSetting,
		*appWhoWidth)

}

func main() {

	if err := run(); err != nil {
		errors.PrintErrorStack(os.Stderr, err, 20)
		os.Exit(1)
	}

	os.Exit(0)
}
