package core

import (
	"os"

	"github.com/hpcloud/tail"

	"github.com/iguazio/kibini/logger"
)

type logTailReader struct {
	logger        logging.Logger
	inputFilePath string
	logWriters    []*logWriter
}

func newLogTailReader(logger logging.Logger,
	inputFilePath string,
	logWriters []*logWriter) *logTailReader {

	r := &logTailReader{
		logger.GetChild("processor"),
		inputFilePath,
		logWriters,
	}

	r.logger.Debug("Created processor")

	return r
}

func (r *logTailReader) read(follow bool) error {
	tailConfig := tail.Config{}
	tailConfig.Location = &tail.SeekInfo{0, os.SEEK_SET}
	tailConfig.Follow = follow
	tailConfig.Logger = tail.DiscardingLogger

	// start tailing the input file
	t, err := tail.TailFile(r.inputFilePath, tailConfig)
	if err != nil {
		return r.logger.Report(err, "Failed to tail file")
	}

	// for each line in the file (both existing and newly added)
	for line := range t.Lines {

		// create a log record from the line
		if logRecord := newLogRecord(line.Text); logRecord != nil {

			// iterate over all writers and write this record
			for _, logWriter := range r.logWriters {
				logWriter.Write(logRecord)
			}
		}
	}

	return nil
}
