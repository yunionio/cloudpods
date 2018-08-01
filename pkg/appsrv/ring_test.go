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
