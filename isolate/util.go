package isolate

import (
	"errors"
	"io"
	"strconv"
)

var (
	ErrBufferTooSmall = errors.New("provided buffer is too small for a output number")
)

type chunk struct {
	body   []byte
	number int
	err    error
}

type multiReader struct {
	rcs []io.ReadCloser

	readCh  chan chunk
	closeCh chan struct{}

	lastReadBody []byte
	lastReadNum  int
}

func NewMultiReader(rcs ...io.ReadCloser) io.ReadCloser {
	readLoop := func(num int, reader io.ReadCloser, pushChan chan chunk, closeCh chan struct{}) {
		defer reader.Close()
		var (
			buffsize = 1024
			err      error
			nn       int
		)
		for {
			buff := make([]byte, buffsize)
			nn, err = reader.Read(buff)

			chnk := chunk{
				body:   buff[:nn],
				number: num,
				err:    err,
			}
			// TODO: increase buffSize if needed

			select {
			case pushChan <- chnk:
				if err != nil {
					return
				}
			case <-closeCh:
				return
			}
		}
	}

	m := &multiReader{
		rcs:     rcs,
		readCh:  make(chan chunk, len(rcs)),
		closeCh: make(chan struct{}),
	}

	for i, reader := range rcs {
		go readLoop(i, reader, m.readCh, m.closeCh)
	}

	return m
}

func (m *multiReader) Close() error {
	for _, readCloser := range m.rcs {
		readCloser.Close()
	}

	select {
	case <-m.closeCh:
	default:
		close(m.closeCh)
	}

	return nil
}

func copyToBuff(dst []byte, src *[]byte, num int) (int, error) {
	numstr := strconv.Itoa(num)
	if len(numstr) > len(dst) {
		return 0, ErrBufferTooSmall
	}

	nn := copy(dst, numstr)
	appended := copy(dst[nn:], *src)
	*src = (*src)[appended:]
	return nn + appended, nil
}

func (m *multiReader) Read(p []byte) (int, error) {
	if len(m.lastReadBody) > 0 {
		return copyToBuff(p, &m.lastReadBody, m.lastReadNum)
	}

	select {
	case chunk := <-m.readCh:
		m.lastReadNum = chunk.number
		m.lastReadBody = chunk.body
		nn, err := copyToBuff(p, &m.lastReadBody, m.lastReadNum)
		if chunk.err != nil {
			return nn, chunk.err
		}
		return nn, err

	case <-m.closeCh:
		return 0, io.EOF
	}
}
