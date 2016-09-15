package logging

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
)

const DefaultTimestampFormat = "2006-01-02T15:04:05.000000"

type jsonFormatter struct {
	// TimestampFormat sets the format used for marshaling timestamps.
	TimestampFormat string
}

func (f *jsonFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	data := make(logrus.Fields, len(entry.Data)+5)

	for k, v := range entry.Data {
		switch v := v.(type) {
		case error:

			// Otherwise errors are ignored by `encoding/json`
			// https://github.com/Sirupsen/logrus/issues/137
			data[k] = v.Error()
		default:
			data[k] = v
		}
	}

	//prefixFieldClashes(data)
	timestampFormat := f.TimestampFormat

	if timestampFormat == "" {
		timestampFormat = DefaultTimestampFormat
	}

	// "when": "2016-06-19T09:56:29.043641"
	data["when"] = entry.Time.Format(timestampFormat)

	// "who": "access_control"
	data["who"] = entry.Data["who"]

	// "severity": "DEBUG"
	data["severity"] = strings.ToUpper(entry.Level.String())

	// "what": "Using etcd discovery"
	data["what"] = entry.Message

	// "more": "{'etcd_address': '127.0.0.1:5251'}
	data["more"] = buildMoreValue(&data)

	// "lang": "go"
	data["lang"] = "go"

	serialized, err := json.Marshal(data)

	if err != nil {
		return nil, fmt.Errorf("Failed to marshal fields to JSON, %v", err)
	}

	return append(serialized, '\n'), nil
}

// Build data["more"] value
func buildMoreValue(data *logrus.Fields) string {
	additionalData := make([]string, 0)

	for key, value := range *data {
		switch key {
		case "when":
		case "who":
		case "severity":
		case "what":
		case "more":
		default:
			formatted_value := fmt.Sprintf("{'%v':'%v'}", key, convertValueToString(value))
			additionalData = append(additionalData, formatted_value)

			//The key was copied to additional_data (No need for duplication)
			delete(*data, key)
		}
	}

	formattedOutput := fmt.Sprintf("%v", additionalData)

	// Removes '[', ']'
	return formattedOutput[1 : len(formattedOutput)-1]
}

// Convert the given value to string
func convertValueToString(value interface{}) string {
	switch value := value.(type) {
	case string:
		return value
	case error:
		//return error message
		return value.Error()
	default:
		return fmt.Sprintf("%v", value)
	}
}
