// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
