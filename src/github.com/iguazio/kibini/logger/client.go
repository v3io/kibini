package logging

import (
	"errors"
	"fmt"

	"github.com/Sirupsen/logrus"
)

// Type aliasing like this allows us to not require everyone who uses the logger to also import logrus.
// They still need to import us, of course.
type Fields logrus.Fields

// Type aliasing for log level
type Level logrus.Level

// Supported log levels
const (
	Error Level = Level(logrus.ErrorLevel)
	Warn  Level = Level(logrus.WarnLevel)
	Info  Level = Level(logrus.InfoLevel)
	Debug Level = Level(logrus.DebugLevel)
)

// This is the internal, unexported client type used as a wrapper around logrus' Entries type.
// It implements the Logger interface and proxy Logger interface calls to all its entries
type client struct {
	entries []*logrus.Entry
}

// The only exported type of the logging package is the Logger interface.
// Any client or service constructor should include this as its first parameter.
type Logger interface {
	SetLevel(Level)
	WithField(string, interface{}) Logger
	GetChild(string) Logger
	With(Fields) Logger
	WithError(error) Logger
	Report(error, string) error
	Debug(string)
	Info(string)
	Warn(string)
	Error(string)
}

// Creates a new logging client composed of file and stdout log entries
func NewClient(name string, dirPath string, logFileName string, disableStdout bool) client {
	entries := make([]*logrus.Entry, 0)

	if !disableStdout {
		entries = append(entries, createNewStdoutEntry())
	}

	if dirPath != "" {
		entry, err := newFileEntry(dirPath, logFileName)

		if err != nil {
			panic(fmt.Sprintf("Could not open file: %s/%s\nerror: %s", dirPath, logFileName, err))
		}

		entries = append(entries, entry)
	}

	c := client{entries: entries}

	return c.WithField("who", name).(client)
}

// Change log level
func (c client) SetLevel(level Level) {
	for _, entry := range c.entries {
		entry.Logger.Level = logrus.Level(level)
	}
}

// Add a single field to the Entry.
// Return client with updated entries
func (c client) WithField(key string, value interface{}) Logger {
	return c.With(Fields{key: value})
}

// Returns a Logger with the given Fields. Calling any log method (e.g. Debug()) on the result
// will cause the given message to also include parameterized field data.
// Proxies a call to *logrus.Entry.WithFields, converting the given Fields to logrus.Fields as required.
func (c client) With(f Fields) Logger {
	fields := logrus.Fields(f)
	entries := make([]*logrus.Entry, len(c.entries))

	for index, entry := range c.entries {
		entries[index] = entry.WithFields(fields)
	}

	return client{entries: entries}
}

// Creates a new child Logger from an existing parent.
// Meant for use inside service/client constructors, with the resulting child being assigned as their Logger.
// Example: api.NewService() receives a client with the name "api" and calls GetChild on it with "service".
// The result is a child client with the name "api.service", and that is assigned as the api.Service's logger.
func (c client) GetChild(name string) Logger {

	// Since all entries share the same name, we can use the first one's name safely
	currentName := c.entries[0].Data["who"]
	newName := fmt.Sprintf("%s.%s", currentName, name)
	return c.WithField("who", newName)
}

// Returns a Logger with the given error. Calling any log method (e.g. Error()) on the result
// will cause the given message to also include the error. Typically the log method will be Error(),
// but this is not enforced and this method can be used to warn or inform about errors as well.
func (c client) WithError(err error) Logger {
	entries := make([]*logrus.Entry, len(c.entries))

	for index, entry := range c.entries {
		entries[index] = entry.WithError(err)
	}

	return client{entries: entries}
}

// Reports the given error with a description and returns a new error based on the description.
// Meant to be called by any code that returns an error.
// A typical usage scenario when interacting with a third-party module:
//
// 	func (s service) DoSomethingWithX() error {
//		x, err := y.GetX()
//
//		if err != nil {
//			return s.logger.Report(err, "Failed to get X")
//		}
//
//		x.DoSomething()
//		return nil
//	}

func (c client) Report(err error, desc string) error {
	for _, entry := range c.entries {
		entry.WithError(err).Warn(desc)
	}

	return errors.New(desc)
}

func (c client) Debug(message string) {
	c.log(message, Debug)
}

func (c client) Info(message string) {
	c.log(message, Info)
}

func (c client) Warn(message string) {
	c.log(message, Warn)
}

func (c client) Error(message string) {
	c.log(message, Error)
}

func (c client) log(message string, level Level) {
	type entryLogFunc func(...interface{})
	var f entryLogFunc

	for _, entry := range c.entries {

		// Select logger method based on the given level
		switch level {
		case Debug:
			f = entry.Debug
		case Error:
			f = entry.Error
		case Info:
			f = entry.Info
		case Warn:
			f = entry.Warn
		}

		f(message)
	}
}
