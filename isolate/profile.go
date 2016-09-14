package isolate

import (
	"fmt"
)

type Profile map[string]interface{}

func (p Profile) Type() string {
	return fmt.Sprintf("%s", p["type"])
}

func (p Profile) Dump() string {
	return fmt.Sprintf("%s", p)
}

