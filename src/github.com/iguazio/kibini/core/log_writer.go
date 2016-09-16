package core

type logWriter interface {
	Write(logRecord *logRecord) error
}
