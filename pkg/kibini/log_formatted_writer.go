package core

import (
	"io"

	"github.com/nuclio/errors"
	"github.com/nuclio/logger"
)

type logFormattedWriter struct {
	logger    logger.Logger
	formatter logFormatter
	writer    io.Writer
}

func newLogFormattedWriter(logger logger.Logger,
	formatter logFormatter,
	writer io.Writer) *logFormattedWriter {
	return &logFormattedWriter{
		logger:    logger.GetChild("formatted-writer"),
		formatter: formatter,
		writer:    writer,
	}
}

func (lfw *logFormattedWriter) Write(logRecord *logRecord) error {
	if _, err := lfw.writer.Write([]byte(lfw.formatter.Format(logRecord))); err != nil {
		return errors.Wrap(err, "Failed to write log record")
	}

	return nil
}
