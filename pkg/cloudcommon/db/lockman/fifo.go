package lockman

import (
	"container/list"
	"yunion.io/x/log"
)

type FIFO struct {
	fifo *list.List
}

func NewFIFO() *FIFO {
	return &FIFO{fifo: list.New()}
}

func (f *FIFO) Push(ele interface{}) {
	f.fifo.PushBack(ele)
}

func (f *FIFO) Pop(ele interface{}) interface{} {
	e := f.fifo.Front()
	for e != nil && e.Value != ele {
		e = e.Next()
	}
	if e != nil {
		v := f.fifo.Remove(e)
		if v != ele {
			log.Fatalf("remove element not identical!!")
		}
	}
	return nil
}

func (f *FIFO) Len() int {
	return f.fifo.Len()
}

type ElementInspectFunc func(ele interface{})

func (f *FIFO) Enum(eif ElementInspectFunc) {
	e := f.fifo.Front()
	for e != nil {
		eif(e.Value)
		e = e.Next()
	}
}