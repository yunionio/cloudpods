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

package appsrv

import (
	"sync"
)

type Ring struct {
	buffer       []interface{}
	header, tail int
	lock         *sync.Mutex
}

func NewRing(size int) *Ring {
	r := Ring{
		buffer: make([]interface{}, size+1),
		header: 0,
		tail:   0,
		lock:   &sync.Mutex{},
	}
	return &r
}

func nextPointer(idx int, size int) int {
	idx = idx + 1
	if idx >= size {
		idx = 0
	}
	return idx
}

func (r *Ring) Push(val interface{}) bool {
	r.lock.Lock()
	defer r.lock.Unlock()
	if nextPointer(r.header, len(r.buffer)) == r.tail {
		return false
	}
	r.buffer[r.header] = val
	r.header = nextPointer(r.header, len(r.buffer))
	return true
}

func (r *Ring) Pop() interface{} {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.tail == r.header {
		return nil
	}
	ret := r.buffer[r.tail]
	r.buffer[r.tail] = nil
	r.tail = nextPointer(r.tail, len(r.buffer))
	return ret
}

func (r *Ring) Capacity() int {
	return len(r.buffer) - 1
}

func (r *Ring) Size() int {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.tail <= r.header {
		return r.header - r.tail
	} else {
		return len(r.buffer) - r.tail + r.header
	}
}
