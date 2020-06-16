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
	"testing"
)

func TestRing(t *testing.T) {
	var (
		r    = NewRing(10)
		push = func(v int32) {
			r.Push(v)
		}
		pop = func(want int32) {
			got := r.Pop().(int32)
			if got != want {
				t.Fatalf("got %d, want %d", got, want)
			}
			for i := r.header; i != r.tail; i = nextPointer(i, len(r.buffer)) {
				if r.buffer[i] != nil {
					t.Fatalf("head %d, tail %d, index %d not nil",
						r.header, r.tail, i)
				}
			}
		}
	)
	push(10)
	push(20)
	push(30)

	pop(10)
	pop(20)
	pop(30)
	if v := r.Pop(); v != nil {
		t.Fatalf("want nil, got %#v", v)
	}
}

func TestOverflow(t *testing.T) {
	r := NewRing(1)
	if r.Capacity() != 1 {
		t.Error("Wrong capacity")
	}
	if r.Size() != 0 {
		t.Error("Wrong size")
	}
	if r.Push(1) != true {
		t.Error("Push should success")
	}
	if r.Push(2) != false {
		t.Error("Push should fail")
	}
	r.Pop()
	r.Push(2)
	if r.Size() != 1 {
		t.Error("Wrong size")
	}
	r.Pop()
	if r.Size() != 0 {
		t.Error("Wrong size")
	}
	r.Push(3)
	if r.Size() != 1 {
		t.Error("Wrong size")
	}
}
