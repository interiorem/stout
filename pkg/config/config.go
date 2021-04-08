package config

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/interiorem/stout/pkg/logutils"
	"github.com/interiorem/stout/isolate"
)

type JSONEncodedDuration time.Duration

func (d *JSONEncodedDuration) UnmarshalJSON(b []byte) error {
	parsed, err := time.ParseDuration(strings.Trim(string(b), "\""))
	if err != nil {
		return err
	}

	*d = JSONEncodedDuration(parsed)
	return nil
}

// Config describes a configuration file for the daemon
type Config struct {
	Version     int      `json:"version"`
	Endpoints   []string `json:"endpoints"`
	DebugServer string   `json:"debugserver"`
	Logger      struct {
		Level  logutils.Level `json:"level"`
		Output string         `json:"output"`
	} `json:"logger"`
	Metrics struct {
		Type   string              `json:"type"`
		Period JSONEncodedDuration `json:"period"`
		Args   json.RawMessage     `json:"args"`
	} `json:"metrics"`
	Isolate map[string]struct {
		Type string            `json:"type"`
		Args isolate.BoxConfig `json:"args"`
	} `json:"isolate"`
}

func (c *Config) Validate() error {
	if len(c.Isolate) == 0 {
		return fmt.Errorf("`isolate` section must containe at least one item")
	}

	if len(c.Endpoints) == 0 {
		return fmt.Errorf("`endpoints` section must containe at least one item")
	}

	return nil
}

func Parse(data []byte) (*Config, error) {
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}
