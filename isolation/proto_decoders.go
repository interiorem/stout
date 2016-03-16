package isolation

import (
	"encoding/json"
	"io"

	"github.com/ugorji/go/codec"
)

type jsonArgsDecoder struct{}

func (j jsonArgsDecoder) Unpack(in interface{}, out ...interface{}) error {
	data, err := json.Marshal(in)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &out)
	return err
}

type msgpackArgsDecoder struct{}

var (
	mPayloadHandler codec.MsgpackHandle
	payloadHandler  = &mPayloadHandler

	mhAsocket = codec.MsgpackHandle{
		BasicHandle: codec.BasicHandle{
			EncodeOptions: codec.EncodeOptions{
				StructToArray: true,
			},
		},
	}
	hAsocket = &mhAsocket
)

func newMsgpackDecoder(r io.Reader) Decoder {
	return codec.NewDecoder(r, hAsocket)
}

func newMsgpackEncoder(w io.Writer) Encoder {
	return codec.NewEncoder(w, hAsocket)
}

func (m msgpackArgsDecoder) Unpack(in interface{}, out ...interface{}) error {
	var buf []byte
	if err := codec.NewEncoderBytes(&buf, payloadHandler).Encode(in); err != nil {
		return err
	}
	if err := codec.NewDecoderBytes(buf, payloadHandler).Decode(out); err != nil {
		return err
	}
	return nil
}

var (
	_ ArgsUnpacker = jsonArgsDecoder{}
	_ ArgsUnpacker = msgpackArgsDecoder{}
)
