package logutils

import (
	"io"
	"os"
)

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }

func newNopCloser(w io.Writer) io.WriteCloser {
	return nopCloser{w}
}

func NewLogFileOutput(filepath string) (io.WriteCloser, error) {
	switch filepath {
	case os.Stderr.Name():
		return newNopCloser(os.Stderr), nil
	case os.Stdout.Name():
		return newNopCloser(os.Stdout), nil
	default:
		return os.OpenFile(filepath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	}
}
