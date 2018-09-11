package lockman

import "testing"

func TestFIFO_Pop(t *testing.T) {
	fifo := NewFIFO()
	data := []int{1, 2, 3}
	for i := 0; i < len(data); i += 1 {
		fifo.Push(data[i])
	}
	t.Logf("FIFO size: %d", fifo.Len())
	fifo.Enum(func(ele interface{}) {
		t.Logf("FIFO ele %#v", ele)
	})
	for i := 0; i < len(data); i += 1 {
		fifo.Pop(data[i])
	}
	t.Logf("FIFO size: %d", fifo.Len())
}
