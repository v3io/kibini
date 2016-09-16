package core

import (
	"bytes"
	"encoding/json"
	"time"
)

type logRecord struct {
	WhenRaw  string `json:"when"`
	When     time.Time
	Who      string `json:"who"`
	What     string `json:"what"`
	Severity string `json:"severity"`
	More     string `json:"more"`
}

func newLogRecord(unparsedLogRecord string) *logRecord {
	var err error
	logRecord := logRecord{}

	if err := json.Unmarshal([]byte(unparsedLogRecord), &logRecord); err != nil {
		return nil
	}

	// parse time ourselves due to missing Z
	logRecord.When, err = time.Parse(time.RFC3339, logRecord.WhenRaw+"Z")
	if err != nil {
		return nil
	}

	return &logRecord
}

func (lr *logRecord) rtruncateString(s string, length int) string {
	sLen := len(s)

	if length > sLen {
		length = sLen
	}

	return s[sLen-length:]
}

func (lr *logRecord) indentJson(unindentedJson string) string {
	var out bytes.Buffer
	err := json.Indent(&out, []byte(unindentedJson), "", "\t")
	if err != nil {
		return unindentedJson
	}
	return out.String()
}
