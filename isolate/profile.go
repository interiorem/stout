package isolate

import (
	"fmt"
	"encoding/json"
)

type Profile map[string]interface{}

func (p Profile) Type() string {
	return fmt.Sprintf("%s", p["type"])
}

func (p Profile) String() string {
	j, e := json.Marshal(p)
	if e == nil {
		return fmt.Sprintf("%s", j)
	} else {
		return "nil"
	}
}

