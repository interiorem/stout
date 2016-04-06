package isolate

import (
	"fmt"
)

// type Isolate struct {
// 	Type string                 `json:"type",codec:"type"`
// 	Args map[string]interface{} `json:"args",codec:"args"`
// }

type Profile map[string]interface{}

func (p Profile) Type() string {
	return fmt.Sprintf("%s", p["type"])
}
