package logging

import (
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
)

// Log file mode
const fileMode = os.O_APPEND | os.O_CREATE | os.O_WRONLY

// Log file permissions
const filePermissions os.FileMode = 0666

// Create log entry to file with json serializer
func newFileEntry(dirPath string, fileName string) (*logrus.Entry, error) {
	filePath := fmt.Sprintf("%s/%s", dirPath, fileName)
	fileHandle, err := os.OpenFile(filePath, fileMode, filePermissions)

	if err != nil {
		return nil, err
	}

	logger := logrus.New()
	logger.Out = fileHandle
	logger.Formatter = &jsonFormatter{}

	return logrus.NewEntry(logger), nil
}

// Create log entry to stdout with text serializer
func createNewStdoutEntry() *logrus.Entry {
	logger := logrus.New()
	logger.Out = os.Stdout
	logger.Formatter = &textFormatter{ForceColors: true}

	return logrus.NewEntry(logger)
}
