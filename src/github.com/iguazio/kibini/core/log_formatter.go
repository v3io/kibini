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
	whoWidth int
}

func newHumanReadableFormatter(color bool, whoWidth int) *humanReadableFormatter {
	return &humanReadableFormatter{
		color,
		whoWidth,
	}
}

func (hrf *humanReadableFormatter) Format(logRecord *logRecord) string {
	var formatted string
	severityCode := logRecord.Severity[0]

	if !hrf.color {
		formatted = fmt.Sprintf("%s %30s (%c) %s ",
			logRecord.When.Format("02.01.06 15:04:05.000000"),
			logRecord.rtruncateString(logRecord.Who, hrf.whoWidth),
			logRecord.Severity[0],
			logRecord.What)
	} else {
		formatted = fmt.Sprintf("%s%s %30s%s: (%s%c%s) %s%s%s ",
			ansi.LightBlack,
			logRecord.When.Format("020106 15:04:05.000000"),
			logRecord.rtruncateString(logRecord.Who, hrf.whoWidth),
			ansi.Reset,
			hrf.getSeverityColor(severityCode), severityCode, ansi.Reset,
			ansi.Cyan, logRecord.What, ansi.Reset)
	}

	// if there's a context, add it to more as a string in quotations
	if len(logRecord.Ctx) > 0 {
		rm := json.RawMessage(fmt.Sprintf("\"%s\"", logRecord.Ctx))
		logRecord.More["ctx"] = &rm
	}

	marshalledMore, err := json.MarshalIndent(logRecord.More, "", "    ")

	if err != nil {
		formatted += fmt.Sprintf("<Error formatting more: %s>", err)
	} else {

		// if the string is short, apply some magic to it so that it looks nice
		if len(marshalledMore) < 150 {
			formatted += hrf.formatShortMore(marshalledMore)
		} else {
			formatted += strings.Replace(string(marshalledMore), "\\n", "\n", -1)
		}
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
	result := string(marshalledMore)

	// replace newlines and stuff with what was supposed to be a tab
	result = strings.Replace(result, "\n", "", -1)
	result = strings.Replace(result, "{    ", "{", -1)
	result = strings.Replace(result, ",    ", ", ", -1)

	return result
}
