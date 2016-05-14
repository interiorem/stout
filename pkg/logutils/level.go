package logutils

import (
	"bytes"

	"github.com/apex/log"
)

type Level log.Level

func (l *Level) UnmarshalJSON(data []byte) error {
	level, err := log.ParseLevel(string(bytes.Trim(data, "\"")))
	if err != nil {
		return err
	}
	(*l) = Level(level)
	return nil
}
