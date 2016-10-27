package core

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"

	"github.com/iguazio/kibini/logger"
	"io"
	"sync"
	"time"
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
	logger  logging.Logger
	readers map[string]logReader
}

func NewKibini(logger logging.Logger) *Kibini {
	return &Kibini{
		logger,
		map[string]logReader{},
	}
}

func (k *Kibini) ProcessLogs(inputPath string,
	inputFollow bool,
	outputPath string,
	outputMode OutputMode,
	outputStdout bool,
	services string,
	noServices string) (err error) {

	// get the log file names on which we shall work
	inputFileNames, err := k.getSourceLogFileNames(inputPath, services, noServices)
	if err != nil {
		k.logger.Report(err, "Failed to get filtered log file names")
	}

	// create log writers - for each input file name, a list of writers will be provided
	logWritersByLogFileName, writerWaitGroup, err := k.createLogWriters(inputPath,
		inputFileNames,
		inputFollow,
		outputPath,
		outputMode,
		outputStdout)
	if err != nil {
		k.logger.Report(err, "Failed to create log writers")
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
		k.logger.With(logging.Fields{
			"inputFilePath": inputFilePath,
			"logReader":     fileLogReader,
		}).Debug("Starting to read")

		readerWaitGroup.Add(1)

		// do the read in a go routine which upon completion signals the wait group
		go func(reader logReader) {

			// tell the reader to read - if it tails it might never stop
			reader.read(inputFollow)

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
	services string,
	noServices string) ([]string, error) {
	var filteredLogFileNames []string
	var unfilteredLogFileNames []string
	var err error

	// get all log files in log directory
	unfilteredLogFileNames, err = filepath.Glob(filepath.Join(inputPath, "*.log"))
	if err != nil {
		return nil, k.logger.Report(err, "Failed to list log directory")
	}

	// compile a match regex and get the mode (include / exclude)
	compiledServiceFilter, serviceFilterType, err := k.compileServiceFilter(services, noServices)
	if err != nil {
		return nil, k.logger.Report(err, "Failed to compile service filter")
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

func (k *Kibini) compileServiceFilter(services string,
	noServices string) (compiledFilter *regexp.Regexp, filterType serviceFilterType, err error) {

	var filter string

	// can only either do filter by services or filter out certain services, not both at the same time
	if len(services) != 0 && len(noServices) != 0 {
		err = errors.New("'services' and 'no-services' are mutually exclusive")
		return
	} else if len(services) != 0 {
		filter = services
		filterType = serviceFilterInclude
	} else if len(noServices) != 0 {
		filter = noServices
		filterType = serviceFilterExclude
	} else {
		filterType = serviceFilterNone
		return
	}

	k.logger.With(logging.Fields{
		"filterType": filterType,
		"filter":     filter,
	}).Debug("Compiling service filter")

	compiledFilter, err = regexp.Compile(filter)
	return
}

func (k *Kibini) createLogWriters(inputPath string,
	inputFileNames []string,
	inputFollow bool,
	outputPath string,
	outputMode OutputMode,
	outputStdout bool) (logWriters map[string][]logWriter, writerWaitGroup *sync.WaitGroup, err error) {

	var outputFileWriter io.Writer
	writerWaitGroup = new(sync.WaitGroup)
	logWriters = map[string][]logWriter{}

	if outputMode == OutputModePer {

		// create a formatter/writer per file
		for _, inputFileName := range inputFileNames {
			outputFilePath := filepath.Join(outputPath, inputFileName+".fmt")

			outputFileWriter, err = k.createOutputFileWriter(outputFilePath)
			if err != nil {
				err = k.logger.With(logging.Fields{
					"outputFilePath": outputFilePath,
				}).Report(err, "Failed to create output file writer")
				return
			}

			// create a single formatter/writer for this input file
			humanReadableFormatter := newHumanReadableFormatter(false)
			logWriters[inputFileName] = []logWriter{
				newLogFormattedWriter(k.logger, humanReadableFormatter, outputFileWriter),
			}
		}
	} else if outputMode == OutputModeSingle {
		writers := []logWriter{}

		// create a formatter/writer which will receive the sorted log records from the merger
		if len(outputPath) != 0 {

			// create an output file writer
			outputFileWriter, err = k.createOutputFileWriter(outputPath)
			if err != nil {
				err = k.logger.With(logging.Fields{
					"outputPath": outputPath,
				}).Report(err, "Failed to create output file writer")
				return
			}

			fileWriter := newLogFormattedWriter(k.logger,
				newHumanReadableFormatter(false),
				outputFileWriter)

			writers = append(writers, fileWriter)
		}

		// if stdout is requested, create a writer for it
		if outputStdout {
			stdoutWriter := newLogFormattedWriter(k.logger,
				newHumanReadableFormatter(true),
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

	return
}

func (k *Kibini) createOutputFileWriter(outputFilePath string) (io.Writer, error) {
	var err error

	// create output file
	outputFile, err := os.OpenFile(outputFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, k.logger.With(logging.Fields{
			"outputFilePath": outputFilePath,
		}).Report(err, "Failed to open output file")
	}

	k.logger.With(logging.Fields{
		"outputFilePath": outputFilePath,
	}).Debug("Created output file writer")

	return outputFile, nil
}

func (k *Kibini) getMergerTimeouts(inputFollow bool) (time.Duration, time.Duration) {

	if !inputFollow {
		// if there's no follow, don't do any force flushing (it'll just waste cycles and may end up with a badly
		// sorted output). After 1 second of inactivity, assume that all readers finished reading
		return 1 * time.Second, 0

	} else {
		// if tail is specified, be more aggressive with checking for silent periods (750ms) and force flush
		// after 2 seconds
		return 750 * time.Millisecond, 2 * time.Second
	}
}
