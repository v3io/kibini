package core

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/andrew-d/go-termutil"
	"github.com/nuclio/errors"
	"github.com/nuclio/logger"
)

// whether to include matched services or exclude them
type serviceFilterType int

const (
	serviceFilterNone serviceFilterType = iota
	serviceFilterInclude
	serviceFilterExclude
)

type OutputMode int

const (
	OutputModeSingle OutputMode = iota
	OutputModePer
)

type Kibini struct {
	logger  logger.Logger
	readers map[string]logReader
}

func NewKibini(logger logger.Logger) *Kibini {
	return &Kibini{
		logger.GetChild("kibini-logger"),
		map[string]logReader{},
	}
}

func (k *Kibini) ProcessLogs(inputPath string,
	inputFollow bool,
	outputPath string,
	outputMode OutputMode,
	outputStdout bool,
	userRegex string,
	userNoRegex string,
	singleFile string,
	colorSetting string,
	whoWidth int) (err error) {
	var inputFileNames []string

	if singleFile != "\000" {

		// if the user specified one file: verify existence
		var fullSingleFilePath = filepath.Join(inputPath, singleFile)
		if _, err = os.Stat(fullSingleFilePath); err == nil {
			inputFileNames = append(inputFileNames, singleFile)
		} else {
			return errors.Wrap(err, "Given file not found in directory")
		}
	} else {

		// else, get the log file names on which we shall work
		inputFileNames, err = k.getSourceLogFileNames(inputPath, userRegex, userNoRegex)
		if err != nil {
			return errors.Wrap(err, "Failed to get filtered log file names")
		}
	}

	// create log writers - for each input file name, a list of writers will be provided
	logWritersByLogFileName, writerWaitGroup, err := k.createLogWriters(inputPath,
		inputFileNames,
		inputFollow,
		outputPath,
		outputMode,
		outputStdout,
		colorSetting,
		whoWidth)
	if err != nil {
		return errors.Wrap(err, "Failed to create log writers")
	}

	// create a log processor
	for _, inputFileName := range inputFileNames {
		inputFilePath := filepath.Join(inputPath, inputFileName)

		k.readers[inputFilePath] = newLogTailReader(k.logger,
			inputFilePath,
			logWritersByLogFileName[inputFileName])
	}

	var readerWaitGroup sync.WaitGroup

	// tell all log readers to start reading
	for inputFilePath, fileLogReader := range k.readers {
		k.logger.DebugWith("Starting to read",
			"inputFilePath", inputFilePath,
			"logReader", fileLogReader)

		readerWaitGroup.Add(1)

		// do the read in a go routine which upon completion signals the wait group
		go func(reader logReader) {

			// tell the reader to read - if it tails it might never stop
			reader.read(inputFollow) // nolint: errcheck

			// this specific reader is done
			readerWaitGroup.Done()
		}(fileLogReader)
	}

	// wait for all reads and writes to complete
	readerWaitGroup.Wait()
	writerWaitGroup.Wait()

	return nil
}

func (k *Kibini) getSourceLogFileNames(inputPath string,
	userRegex string,
	userNoRegex string) ([]string, error) {
	var filteredLogFileNames []string
	var unfilteredLogFileNames []string
	var err error

	// get all log files in log directory
	unfilteredLogFileNames, err = k.getLogFilesInDirectory(inputPath)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to list log directory")
	}

	// compile a match regex and get the mode (include / exclude)
	compiledServiceFilter, serviceFilterType, err := k.compileServiceFilter(userRegex, userNoRegex)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to compile service filter")
	}

	// iterate over all logfiles
	for _, unfilteredLogFileName := range unfilteredLogFileNames {
		filterMatch := true
		includeInFiltered := false

		// get base only
		unfilteredLogFileName = filepath.Base(unfilteredLogFileName)

		// if there's a filter, pass it through
		if compiledServiceFilter != nil {
			filterMatch = compiledServiceFilter.MatchString(unfilteredLogFileName)
		}

		includeInFiltered = serviceFilterType == serviceFilterNone ||
			(serviceFilterType == serviceFilterExclude && !filterMatch) ||
			(serviceFilterType == serviceFilterInclude && filterMatch)

		if includeInFiltered {
			filteredLogFileNames = append(filteredLogFileNames, unfilteredLogFileName)
		}
	}

	return filteredLogFileNames, nil
}

// getLogFilesInDirectory returns all the log files in the given inputPath.
// a log file is considered a file that ends with '.log' or 'log.<number>'
func (k *Kibini) getLogFilesInDirectory(inputPath string) (logFiles []string, err error) {
	var fileNamesInLogDir []string
	var logFileRegexp *regexp.Regexp

	// get all log files in log directory, first get all the files with 'log' in them
	fileNamesInLogDir, err = filepath.Glob(filepath.Join(inputPath, "*.log*"))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to find files")
	}

	// now get only files that ends with `.log` or `log.<digits>`
	logFileRegexp, err = regexp.Compile(`^.*\.(log|log\.[0-9]+)$`)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to compile regex")
	}

	for _, fileName := range fileNamesInLogDir {
		matched := logFileRegexp.MatchString(fileName)
		if matched {
			logFiles = append(logFiles, fileName)
		}
	}
	return
}

func (k *Kibini) compileServiceFilter(userRegex string,
	userNoRegex string) (compiledFilter *regexp.Regexp, filterType serviceFilterType, err error) {

	var filter string

	// can only either do filter by userRegex or filter out using userNoRegex, not both at the same time
	if len(userRegex) != 0 && len(userNoRegex) != 0 {
		err = errors.New("'--regex' and '--no-regex' are mutually exclusive")
		return
	} else if len(userRegex) != 0 {
		filter = userRegex
		filterType = serviceFilterInclude
	} else if len(userNoRegex) != 0 {
		filter = userNoRegex
		filterType = serviceFilterExclude
	} else {
		filterType = serviceFilterNone
		return
	}

	k.logger.DebugWith("Compiling service filter",
		"filterType", filterType,
		"filter", filter)

	compiledFilter, err = regexp.Compile(filter)
	return
}

func (k *Kibini) createLogWriters(inputPath string,
	inputFileNames []string,
	inputFollow bool,
	outputPath string,
	outputMode OutputMode,
	outputStdout bool,
	colorSetting string,
	whoWidth int) (map[string][]logWriter, *sync.WaitGroup, error) {

	writerWaitGroup := new(sync.WaitGroup)
	logWriters := map[string][]logWriter{}
	color := k.determineColorSetting(colorSetting, outputStdout)

	if outputMode == OutputModePer {

		// create a formatter/writer per file
		for _, inputFileName := range inputFileNames {
			outputFilePath := filepath.Join(outputPath, inputFileName+".fmt")

			outputFileWriter, err := k.createOutputFileWriter(outputFilePath)
			if err != nil {
				return nil, nil, errors.Wrap(err, "Failed to create output file writer")
			}

			// create a single formatter/writer for this input file
			humanReadableFormatter := newHumanReadableFormatter(color, whoWidth)
			logWriters[inputFileName] = []logWriter{
				newLogFormattedWriter(k.logger, humanReadableFormatter, outputFileWriter),
			}
		}
	} else if outputMode == OutputModeSingle {
		writers := []logWriter{}

		// create a formatter/writer which will receive the sorted log records from the merger
		if len(outputPath) != 0 {

			// create an output file writer
			outputFileWriter, err := k.createOutputFileWriter(outputPath)
			if err != nil {
				return nil, nil, errors.Wrap(err, "Failed to create output file writer")
			}

			fileWriter := newLogFormattedWriter(k.logger,
				newHumanReadableFormatter(color, whoWidth),
				outputFileWriter)

			writers = append(writers, fileWriter)
		}

		// if stdout is requested, create a writer for it
		if outputStdout {
			stdoutWriter := newLogFormattedWriter(k.logger,
				newHumanReadableFormatter(color, whoWidth),
				os.Stdout)

			writers = append(writers, stdoutWriter)
		}

		// get timeouts for merger
		inactivityFlushTimeout, forceFlushTimeout := k.getMergerTimeouts(inputFollow)

		// create a log merger writer that will receive all records, merge them (sorted) and then output
		// them to log writer
		logMerger := newLogMerger(k.logger,
			writerWaitGroup,
			!inputFollow, // if not following, stop after first flush
			!inputFollow, // if not following, stop after the first quiet period
			inactivityFlushTimeout,
			forceFlushTimeout,
			writers)

		// set the log merger as the writer for all input files
		for _, inputFileName := range inputFileNames {
			logWriters[inputFileName] = []logWriter{logMerger}
		}
	}

	return logWriters, writerWaitGroup, nil
}

func (k *Kibini) createOutputFileWriter(outputFilePath string) (io.Writer, error) {
	var err error

	// create output file
	outputFile, err := os.OpenFile(outputFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to open output file")
	}

	k.logger.DebugWith("Created output file writer",
		"outputFilePath", outputFilePath)

	return outputFile, nil
}

func (k *Kibini) getMergerTimeouts(inputFollow bool) (time.Duration, time.Duration) {

	if !inputFollow {

		// if there's no follow, don't do any force flushing (it'll just waste cycles and may end up with a badly
		// sorted output). After 1 second of inactivity, assume that all readers finished reading
		return 1 * time.Second, 0

	}

	// if tail is specified, be more aggressive with checking for silent periods (750ms) and force flush
	// after 2 seconds
	return 750 * time.Millisecond, 2 * time.Second
}

// determine weather to use colors according to user color setting arg and output format:
// If user setting is "always", use colors.
// Else, use color if: we are outputting to stdout AND stdout is a tty AND user setting is not "off"
func (k *Kibini) determineColorSetting(colorSetting string, stdout bool) (color bool) {
	color = false
	if colorSetting == "always" {
		color = true
	} else if stdout && colorSetting != "off" && termutil.Isatty(os.Stdout.Fd()) {
		color = true
	}
	return
}
