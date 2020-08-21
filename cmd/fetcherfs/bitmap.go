package main

type BitMap struct {
	bits []byte
	size int64
}

func (bm *BitMap) Set(idx int64) {
	if idx > bm.size {
		return
	}
	subscript := idx / 8
	pos := idx % 8
	bm.bits[subscript] = (bm.bits[subscript] | 1<<pos)
}

func (bm *BitMap) Has(idx int64) bool {
	if idx > bm.size {
		return false
	}
	subscript := idx / 8
	pos := idx % 8
	return bm.bits[subscript]&(1<<pos) > 0
}

func (bm *BitMap) Clean(idx int64) {
	if idx > bm.size {
		return
	}
	subscript := idx / 8
	pos := idx % 8
	bm.bits[subscript] &= ^(1 << pos)
}

func NewBitMap(size int64) *BitMap {
	bits := make([]byte, (size>>3)+1)
	return &BitMap{bits: bits, size: size}
}
