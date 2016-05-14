package config

import (
	"encoding/json"
	"fmt"

	"github.com/noxiouz/stout/isolate"
	"github.com/noxiouz/stout/pkg/logutils"
)

// Config describes a configuration file for the daemon
type Config struct {
	Endpoints   []string `json:"endpoints"`
	DebugServer string   `json:"debugserver"`
	Logger      struct {
		Level  logutils.Level `json:"level"`
		Output string         `json:"output"`
	} `json:"logger"`
	Isolate map[string]isolate.BoxConfig `json:"isolate"`
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
