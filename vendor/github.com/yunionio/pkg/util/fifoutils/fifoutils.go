package fifoutils

import (
	"errors"

	"github.com/yunionio/log"
)

var ErrEmpty error

func init() {
	ErrEmpty = errors.New("fifo is empty")
}

type FIFO struct {
	array []interface{}
	len   int
}

func NewFIFO() *FIFO {
	fifo := FIFO{array: make([]interface{}, 0), len: 0}
	return &fifo
}

func (f *FIFO) Push(ele interface{}) {
	if f.len < len(f.array) {
		f.array[f.len] = ele
	} else {
		f.array = append(f.array, ele)
	}
	f.len += 1
}

func (f *FIFO) Pop() interface{} {
	if f.len <= 0 {
		return nil
	}
	ele := f.array[0]
	f.len -= 1
	for i := 0; i < f.len; i += 1 {
		f.array[i] = f.array[i+1]
	}
	f.array[f.len] = nil
	return ele
}

func (f *FIFO) Len() int {
	return f.len
}

func (f *FIFO) ElementAt(idx int) interface{} {
	if idx >= 0 && idx < f.len {
		return f.array[idx]
	} else {
		log.Fatalf("Out of index")
		return nil
	}
}
