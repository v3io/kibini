package core

import (
	"encoding/json"
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
			logRecord.When.Format("02.01.06 15:04:05.000000"),
			logRecord.rtruncateString(logRecord.Who, 30),
			logRecord.Severity[0],
			logRecord.What)
	} else {
		formatted = fmt.Sprintf("%s%s %30s%s: (%s%c%s) %s%s%s ",
			ansi.LightBlack,
			logRecord.When.Format("020106 15:04:05.000000"),
			logRecord.rtruncateString(logRecord.Who, 30),
			ansi.Reset,
			hrf.getSeverityColor(severityCode), severityCode, ansi.Reset,
			ansi.Cyan, logRecord.What, ansi.Reset)
	}

	// marshal the string
	marshalledMore, _ := json.MarshalIndent(logRecord.More, "", "    ")

	// if the string is short, apply some magic to it so that it looks nice
	if len(marshalledMore) < 150 {
		formatted += hrf.formatShortMore(marshalledMore)
	} else {
		formatted += strings.Replace(string(marshalledMore), "\\n", "\n", -1)
	}

	return formatted + "\n"
}

func (hrf *humanReadableFormatter) getSeverityColor(severityCode byte) string {
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

func (hrf *humanReadableFormatter) formatShortMore(marshalledMore []byte) string {
	shortMore := []byte{}

	for idx, c := range marshalledMore {

		// skip newlines
		if c == '\n' {
			continue

			// convert tabs to space, except for the first one since we don't want a space
			// after the curly brackets
		} else if c == '\t' {

			// delete the first tab, replace the rest with spaces
			if idx != 2 {
				shortMore = append(shortMore, ' ')
			}
		} else {
			shortMore = append(shortMore, c)
		}
	}

	return string(shortMore)
}
