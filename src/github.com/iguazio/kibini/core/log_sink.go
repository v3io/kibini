package core

import "github.com/iguazio/kibini/logger"

type logSink struct {
	logger   	logging.Logger
	outputFilePath 	string
}

func NewLogSink(logger logging.Logger,
	outputFilePath string) *logSink {
	return &logSink{
		logger,
		outputFilePath,
	}
}
