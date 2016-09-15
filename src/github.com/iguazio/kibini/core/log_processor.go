package core

import (
	"io"
	"os"
	"encoding/json"
	"fmt"
	"time"
	"errors"

	"github.com/iguazio/kibini/logger"
	"github.com/hpcloud/tail"
	"strings"
	"bytes"
)

type logProcessor struct {
	logger   	logging.Logger
	inputFilePath	string
	logSink  	io.Writer
}

func newLogProcessor(logger logging.Logger,
	inputFilePath string,
	logSink io.Writer) *logProcessor {

	lp := &logProcessor{
		logger.GetChild("processor"),
		inputFilePath,
		logSink,
	}

	lp.logger.Debug("Created processor")

	// start tailing file
	go lp.tailFile(inputFilePath)

	return lp
}

func (lp *logProcessor) tailFile(filePath string) error {

	tailConfig := tail.Config{}
	tailConfig.Location = &tail.SeekInfo{0, os.SEEK_SET}
	tailConfig.Follow = true
	// tailConfig.Logger = tail.DiscardingLogger

	// start tailing the input file
	t, err := tail.TailFile(filePath, tailConfig)
	if err != nil {
		return lp.logger.Report(err, "Failed to tail file")
	}

	// read lines
	for line := range t.Lines {
		formattedLine, err := lp.processLine(line.Text)
		if err == nil {
			fmt.Println(formattedLine)
		}
	}

	return nil
}

func (lp *logProcessor) processLine(line string) (string, error) {
	var parsedLine struct {
		When		string `json:"when"`
		Who   	   	string `json:"who"`
		What   	   	string `json:"what"`
		Severity   	string `json:"severity"`
		More   		string `json:"more"`
	}

	if err := json.Unmarshal([]byte(line), &parsedLine); err != nil {
		return "", errors.New("Failed to parse line")
	}

	// parse time ourselves due to missing Z
	parsedWhen, err := time.Parse(time.RFC3339, parsedLine.When + "Z")
	if err != nil {
		return "", errors.New("Failed to parse when")
	}

	// output line
	formattedLine := fmt.Sprintf("%s %40s (%c) %s ",
		parsedWhen.Format("020106 15:04:05.000000"),
		lp.rtruncateString(parsedLine.Who, 40),
		parsedLine.Severity[0],
		parsedLine.What)

	if len(parsedLine.More) > 150 {
		formattedLine += "\n" + lp.indentJson(strings.Replace(parsedLine.More, "'", "\"", -1)) + "\n"
	} else {
		formattedLine += parsedLine.More
	}

	return formattedLine, nil
}

func (lp *logProcessor) rtruncateString(s string, length int) string {
	sLen := len(s)

	if length > sLen {
		length = sLen
	}

	return s[sLen-length:]
}

func (lp *logProcessor) indentJson(unindentedJson string) string {
	var out bytes.Buffer
	err := json.Indent(&out, []byte(unindentedJson), "", "\t")
	if err != nil {
		return unindentedJson
	}
	return out.String()
}
