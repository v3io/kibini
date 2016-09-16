package core

import (
	"io"

	"github.com/iguazio/kibini/logger"
)

type logFormattedWriter struct {
	logger    logging.Logger
	formatter logFormatter
	writer    io.Writer
}

func newLogFormattedWriter(logger logging.Logger,
	formatter logFormatter,
	writer io.Writer) *logFormattedWriter {
	return &logFormattedWriter{
		logger:    logger,
		formatter: formatter,
		writer:    writer,
	}
}

func (lfw *logFormattedWriter) Write(logRecord *logRecord) error {
	lfw.writer.Write([]byte(lfw.formatter.Format(logRecord)))

	return nil
}
