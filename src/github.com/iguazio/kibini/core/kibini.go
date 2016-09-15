package core

import (
	"path/filepath"
	"fmt"
	"regexp"

	"github.com/iguazio/kibini/logger"
	"errors"
)

// whether to include matched services or exclude them
type serviceFilterType int
const (
	serviceFilterNone serviceFilterType = iota
	serviceFilterInclude
	serviceFilterExclude
)

type Kibini struct {
	logger   logging.Logger
}

func NewKibini(logger logging.Logger) *Kibini {
	return &Kibini{
		logger,
	}
}

func (k *Kibini) ProcessLogs(logDir string,
	formattedLogDir string,
	services string,
	noServices string) error {

	// get the log file names on which we shall work
	logFilePaths, err := k.getSourceLogFileNames(logDir, services, noServices)
	if err != nil {
		k.logger.Report(err, "Failed to get filtered log file names")
	}

	// create a log processor
	for _, logFilePath := range logFilePaths {
		logFileProcessor := newLogProcessor(k.logger, filepath.Join(logDir, logFilePath), nil)
		fmt.Println(logFileProcessor.inputFilePath)
	}

	//
	return nil
}

func (k *Kibini) getSourceLogFileNames(logDir string,
	services string,
	noServices string) ([]string, error) {
	var filteredLogFileNames []string
	var unfilteredLogFileNames []string
	var err error

	// get all log files in log directory
	unfilteredLogFileNames, err = filepath.Glob(filepath.Join(logDir, "*.log"))
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
		"filter": filter,
	}).Debug("Compiling service filter")

	compiledFilter, err = regexp.Compile(filter)
	return
}
