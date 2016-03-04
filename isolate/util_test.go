package isolate

import (
	"bytes"
	"io"
	"io/ioutil"
	"strconv"
	"testing"
)

func assertErrNil(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
		t.FailNow()
	}
}

func assertIntEqual(t *testing.T, actual int, expected int) {
	if actual != expected {
		t.Fatalf("expected %d actual %d", expected, actual)
	}
}

func TestMultiReader(t *testing.T) {
	var fixture = []string{
		"0string0",
		"1string1",
		"2string2",
	}

	buffs := make([]io.ReadCloser, 0, len(fixture))
	for _, str := range fixture {
		buffs = append(buffs, ioutil.NopCloser(bytes.NewBufferString(str)))
	}

	mr := NewMultiReader(buffs...)

	const firstReadSize = 3
	var p = make([]byte, firstReadSize)
	nn, err := mr.Read(p)
	assertErrNil(t, err)
	assertIntEqual(t, nn, firstReadSize)
	out1, err := strconv.Atoi(string(p[0]))
	assertErrNil(t, err)
	t.Logf("read %s. size %d from %d", p, nn, out1)

	restBytesInOut1 := len(fixture[out1]) - nn + 1
	p = make([]byte, 10)
	nn, err = mr.Read(p)
	assertErrNil(t, err)
	out2, err := strconv.Atoi(string(p[0]))
	assertErrNil(t, err)
	t.Logf("read %s. size %d from %d", p, nn, out2)
	assertIntEqual(t, out2, out1)
	assertIntEqual(t, nn, restBytesInOut1+1)

	p = make([]byte, 10)
	nn, err = mr.Read(p)
	if err != io.EOF {
		t.Fatalf("expect EOF")
	}

	p = make([]byte, 10)
	nn, err = mr.Read(p)
	assertErrNil(t, err)
	out3, err := strconv.Atoi(string(p[0]))
	t.Logf("read %s. size %d from %d", p, nn, out3)
	assertIntEqual(t, nn, len(fixture[out3])+1)
}
