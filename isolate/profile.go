package isolate

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/tinylib/msgp/msgp"
)

const (
	typeKey = "type"
)

var (
	ErrNoTypeField    = fmt.Errorf("profile does not contain `%s` field", typeKey)
	ErrTwiceDecodedTo = fmt.Errorf("DecodeTo is called twice")
)

var profilesPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 1024)
	},
}

type RawProfile interface {
	DecodeTo(v msgp.Decodable) error
}

func newCocaineProfile() *cocaineProfile {
	buff := profilesPool.Get().([]byte)
	buff = buff[:0]
	return &cocaineProfile{buff: buff}
}

type cocaineProfile struct {
	buff []byte
}

func (p *cocaineProfile) Type() (string, error) {
	raw := msgp.Locate(typeKey, p.buff)
	if len(raw) == 0 {
		return "", ErrNoTypeField
	}

	t, _, err := msgp.ReadStringBytes(raw)
	return t, err
}

func (p *cocaineProfile) Write(b []byte) (int, error) {
	p.buff = append(p.buff, b...)
	return len(b), nil
}

// DecodeTo unpacks []byte to some profile. Profile must not be used after DecodeTo
func (p *cocaineProfile) DecodeTo(v msgp.Decodable) error {
	if p.buff != nil {
		err := msgp.Decode(bytes.NewReader(p.buff), v)
		profilesPool.Put(p.buff)
		p.buff = nil
		return err
	}

	return ErrTwiceDecodedTo
}

func (p *cocaineProfile) String() string {
	return string(p.buff)
}
