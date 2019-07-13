package utils

import (
	"io"
	"os"
	"sync"
)

func BytesArrayToStrArray(ba [][]byte) []string {
	if len(ba) == 0 {
		return nil
	}
	res := make([]string, len(ba))
	for i := 0; i < len(ba); i++ {
		res[i] = string(ba[i])
	}
	return res
}

func StrArrayToBytesArray(sa []string) [][]byte {
	if len(sa) == 0 {
		return nil
	}
	res := make([][]byte, len(sa))
	for i := 0; i < len(sa); i++ {
		res[i] = []byte(sa[i])
	}
	return res
}

func WriteTo(data []byte, w io.Writer) error {
	var n = 0
	var length = len(data)
	for n < length {
		r, e := w.Write(data[n:])
		if e != nil {
			return e
		}
		n += r
	}
	return nil
}

type CloseOnce struct {
	*os.File

	once sync.Once
	err  error
}

func (c *CloseOnce) Close() error {
	c.once.Do(c.close)
	return c.err
}

func (c *CloseOnce) close() {
	c.err = c.File.Close()
}
