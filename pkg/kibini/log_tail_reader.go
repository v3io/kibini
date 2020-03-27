package core

import (
	"io"
	"path/filepath"

	"github.com/hpcloud/tail"
	"github.com/nuclio/errors"
	"github.com/nuclio/logger"
)

type logTailReader struct {
	logger        logger.Logger
	inputFilePath string
	logWriters    []logWriter
}

func newLogTailReader(logger logger.Logger,
	inputFilePath string,
	logWriters []logWriter) *logTailReader {

	r := &logTailReader{
		logger:        logger.GetChild("tail_reader").GetChild(filepath.Base(inputFilePath)),
		inputFilePath: inputFilePath,
		logWriters:    logWriters,
	}

	return r
}

func (ltr *logTailReader) read(follow bool) error {
	tailConfig := tail.Config{}
	tailConfig.Location = &tail.SeekInfo{Offset: 0, Whence: io.SeekStart}
	tailConfig.Follow = follow
	tailConfig.Logger = tail.DiscardingLogger

	// start tailing the input file
	t, err := tail.TailFile(ltr.inputFilePath, tailConfig)
	if err != nil {
		return errors.Wrap(err, "Failed to tail file")
	}

	ltr.logger.Debug("Tailing")

	// for each line in the file (both existing and newly added)
	for line := range t.Lines {

		// create a log record from the line
		if logRecord := newLogRecord(line.Text); logRecord != nil {

			// iterate over all writers and write this record
			for _, logWriter := range ltr.logWriters {
				if err := logWriter.Write(logRecord); err != nil {
					return errors.Wrap(err, "Failed to write record")
				}
			}
		}
	}

	ltr.logger.Debug("Successfully finished tailing")
	return nil
}
