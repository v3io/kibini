package core

type logReader interface {
	read(follow bool) error
}
