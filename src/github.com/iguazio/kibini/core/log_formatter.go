package core

import (
	"fmt"
	"strings"

	"github.com/mgutz/ansi"
)

type logFormatter interface {
	Format(logRecord *logRecord) string
}

type humanReadableFormatter struct {
	color bool
}

func newHumanReadableFormatter(color bool) *humanReadableFormatter {
	return &humanReadableFormatter{
		color,
	}
}

func (hrf *humanReadableFormatter) Format(logRecord *logRecord) string {
	var formatted string
	severityCode := logRecord.Severity[0]

	if !hrf.color {
		formatted = fmt.Sprintf("%s %30s (%c) %s ",
			logRecord.When.Format("020106 15:04:05.000000"),
			logRecord.rtruncateString(logRecord.Who, 30),
			logRecord.Severity[0],
			logRecord.What)
	} else {
		formatted = fmt.Sprintf("%s%s %30s%s (%s%c%s) %s%s%s ",
			ansi.LightBlack,
			logRecord.When.Format("020106 15:04:05.000000"),
			logRecord.rtruncateString(logRecord.Who, 30),
			ansi.Reset,
			getSeverityColor(severityCode), severityCode, ansi.Reset,
			ansi.Cyan, logRecord.What, ansi.Reset)
	}

	if len(logRecord.More) > 150 {
		formatted += "\n" + logRecord.indentJson(strings.Replace(logRecord.More, "'", "\"", -1)) + "\n"
	} else {
		formatted += logRecord.More
	}

	return formatted + "\n"
}

func getSeverityColor(severityCode byte) string {
	switch string(severityCode) {
	case "V":
		return ansi.LightBlue
	case "W":
		return ansi.Yellow
	case "E":
		return ansi.Red
	}

	return ansi.Reset
}
