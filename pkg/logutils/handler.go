package logutils

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/apex/log"
)

func getLevel(lvl log.Level) string {
	switch lvl {
	case log.DebugLevel:
		return "DEBUG"
	case log.InfoLevel:
		return "INFO"
	case log.WarnLevel:
		return "WARN"
	case log.ErrorLevel, log.FatalLevel:
		return "ERROR"
	default:
		return lvl.String()
	}
}

type logHandler struct {
	mu sync.Mutex
	io.Writer
}

// NewLogHandler returns new log.Handler writing to attached io.Writer
func NewLogHandler(w io.Writer) log.Handler {
	return &logHandler{Writer: w}
}

func (lh *logHandler) HandleLog(entry *log.Entry) error {
	keys := make([]string, 0, len(entry.Fields))
	for k := range entry.Fields {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	buf := new(bytes.Buffer)
	buf.WriteString(entry.Timestamp.Format(time.RFC3339))
	buf.WriteByte('\t')
	buf.WriteString(getLevel(entry.Level))
	buf.WriteByte('\t')
	buf.WriteString(entry.Message)
	if i := len(entry.Fields); i > 0 {
		buf.WriteByte('\t')
		buf.WriteByte('[')

		for _, k := range keys {
			buf.WriteString(fmt.Sprintf("%s: %v", k, entry.Fields[k]))
			i--
			if i > 0 {
				buf.WriteByte(',')
				buf.WriteByte(' ')
			}
		}
		buf.WriteByte(']')
	}
	buf.WriteByte('\n')

	lh.mu.Lock()
	defer lh.mu.Unlock()

	_, err := buf.WriteTo(lh.Writer)
	return err
}
