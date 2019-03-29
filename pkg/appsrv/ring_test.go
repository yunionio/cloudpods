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
	r := NewRing(10)
	var v int32 = 10
	r.Push(v)
	v = 20
	r.Push(v)
	v = 30
	r.Push(v)
	v1 := r.Pop().(int32)
	if v1 != 10 {
		t.Error("Fail")
	}
	v2 := r.Pop().(int32)
	if v2 != 20 {
		t.Error("Fail")
	}
	v3 := r.Pop().(int32)
	if v3 != 30 {
		t.Error("Fail")
	}
	v4 := r.Pop()
	if v4 != nil {
		t.Error("Fail")
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
