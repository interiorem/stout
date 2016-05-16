package logutils

import (
	"io/ioutil"
	"testing"

	"github.com/apex/log"
)

func BenchmarkLog4FieldsEntry(b *testing.B) {
	lh := NewLogHandler(ioutil.Discard)
	entry := log.NewEntry(nil).WithFields(
		log.Fields{"FieldA": 1, "FieldB": 2, "FieldC": "fieldc", "D": 200.3})

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			lh.HandleLog(entry)
		}
	})
}
