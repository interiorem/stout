package logutils

import (
	"encoding/json"
	"testing"

	"github.com/apex/log"
)

func TestUnmarshalJSONLevel(t *testing.T) {
	var result struct {
		LogLevel Level `json:"level"`
	}

	cases := []struct {
		body  []byte
		level log.Level
	}{
		{[]byte(`{"level": "info"}`), log.InfoLevel},
		{[]byte(`{"level": "error"}`), log.ErrorLevel},
		{[]byte(`{"level": "debug"}`), log.DebugLevel},
	}

	for _, c := range cases {
		if err := json.Unmarshal(c.body, &result); err != nil {
			t.Fatalf("%v", err)
		}

		if log.Level(result.LogLevel) != c.level {
			t.Fatalf("expected %v, actual %v", c.level, result.LogLevel)
		}
		t.Log(result.LogLevel)
	}

	if err := json.Unmarshal([]byte(`{"level": "devvvd"}`), &result); err == nil {
		t.Fatal("error is expected")
	}
}
