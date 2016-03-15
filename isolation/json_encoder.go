package isolation

import (
	"encoding/json"
)

type jsonArgsDecoder struct{}

var (
	_ ArgsUnpacker = jsonArgsDecoder{}
)

func (j jsonArgsDecoder) Unpack(in interface{}, out ...interface{}) error {
	data, err := json.Marshal(in)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &out)
	return err
}
