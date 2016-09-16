package core

import (
	"io"

	"github.com/iguazio/kibini/logger"
)

type logWriter struct {
	logger    logging.Logger
	formatter logFormatter
	writer    io.Writer
}

func newLogWriter(logger logging.Logger,
	formatter logFormatter,
	writer io.Writer) *logWriter {
	return &logWriter{
		logger,
		formatter,
		writer,
	}
}

func (lw *logWriter) Write(logRecord *logRecord) error {
	lw.writer.Write([]byte(lw.formatter.Format(logRecord)))

	return nil
}
