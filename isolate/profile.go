package isolate

import (
	"fmt"
)

type Profile map[string]interface{}

func (p Profile) Type() string {
	return fmt.Sprintf("%s", p["type"])
}
