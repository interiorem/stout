package isolate

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/noxiouz/stout/pkg/logutils"
	"golang.org/x/net/context"
)

type SpawnConfig struct {
	Opts       RawProfile
	Name       string
	Executable string
	Args       map[string]string
	Env        map[string]string
}

type (
	dispatcherInit func(context.Context, ResponseStream) Dispatcher

	Box interface {
		Spool(ctx context.Context, name string, opts RawProfile) error
		Spawn(ctx context.Context, config SpawnConfig, output io.Writer) (Process, error)
		Inspect(ctx context.Context, workerid string) ([]byte, error)
		Close() error
	}

	ResponseStream interface {
		Write(ctx context.Context, num uint64, data []byte) error
		Error(ctx context.Context, num uint64, code [2]int, msg string) error
		Close(ctx context.Context, num uint64) error
	}

	Process interface {
		Kill() error
	}

	Boxes map[string]Box

	BoxConfig map[string]interface{}

	GlobalState struct {
		Mtn     *MtnState
	}

	JSONEncodedDuration time.Duration
	Config struct {
		Version     int      `json:"version"`
		Endpoints   []string `json:"endpoints"`
		DebugServer string   `json:"debugserver"`
		Logger      struct {
			Level  logutils.Level `json:"level"`
			Output string	 `json:"output"`
		} `json:"logger"`
		Metrics struct {
			Type   string	      `json:"type"`
			Period JSONEncodedDuration `json:"period"`
			Args   json.RawMessage     `json:"args"`
		} `json:"metrics"`
		Isolate map[string]struct {
			Type string	    `json:"type"`
			Args BoxConfig `json:"args"`
		} `json:"isolate"`
		Mtn struct {
			Enable bool `json:"enable,omitempty"`
			Allocbuffer int `json:"allocbuffer,omitempty"`
			Url string `json:"url,omitempty"`
			Label string `json:"label,omitempty"`
			Ident string `json:"ident,omitempty"`
			DbPath string `json:"dbpath,omitempty"`
			Headers map[string]string `json:"headers,omitempty"`
		} `json:"mtn,omitempty"`
	}
)

func (d *JSONEncodedDuration) UnmarshalJSON(b []byte) error {
	parsed, err := time.ParseDuration(strings.Trim(string(b), "\""))
	if err != nil {
		return err
	}

	*d = JSONEncodedDuration(parsed)
	return nil
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
	config.Mtn.Headers = make(map[string]string)
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

const BoxesTag = "isolate.boxes.tag"

var (
	notificationByte = []byte("")
)

func NotifyAboutStart(wr io.Writer) {
	wr.Write(notificationByte)
}

func getBoxes(ctx context.Context) Boxes {
	val := ctx.Value(BoxesTag)
	box, ok := val.(Boxes)
	if !ok {
		panic("context.Context does not contain Box")
	}
	return box
}
