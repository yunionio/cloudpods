package hostimage

/*
#cgo pkg-config: glib-2.0 zlib

#include "libqemuio.h"
#include "qemu/osdep.h"
*/
import "C"

import (
	"fmt"
	"io"
	"os"
	"unsafe"
)

func init() {
	C.qemuio_init()
}

func ReadQcow2(qemuioBlk *C.struct_QemuioBlk, offset int64, count int64) ([]byte, int64) {
	if qemuioBlk == nil || offset < 0 || count < 0 {
		return nil, -1
	}
	b := make([]byte, count)
	var total = C.int64_t(0)
	ret := C.read_qcow2(qemuioBlk, unsafe.Pointer(&b[0]), C.int64_t(offset), C.int64_t(count), &total)
	if ret < 0 {
		return nil, int64(ret)
	} else {
		return b, int64(total)
	}
}

func OpenQcow2(imagePath string, readonly bool) *C.struct_QemuioBlk {
	return C.open_qcow2(C.CString(imagePath), C.bool(readonly))
}

func Qcow2GetLenth(qemuioBlk *C.struct_QemuioBlk) int64 {
	return int64(C.qcow2_get_length(qemuioBlk))
}

func CloseQcow2(qemuioBlk *C.struct_QemuioBlk) {
	C.close_qcow2(qemuioBlk)
}

type IImage interface {
	// Open image file and its backing file (if have)
	Open(imagePath string, readonly bool) error

	// Close may not really close image file handle, just reudce ref count
	Close()

	// If return number < 0 indicate read failed
	Read(offset, count int64) ([]byte, int64)

	// Get image file length, not file actual length, it's image virtual size
	Length() int64
}

type SQcow2Image struct {
	fd *C.struct_QemuioBlk
}

func (img *SQcow2Image) Open(imagePath string, readonly bool) error {
	fd := OpenQcow2(imagePath, readonly)
	if fd == nil {
		return fmt.Errorf("Open image %s failed", imagePath)
	} else {
		img.fd = fd
		return nil
	}
}

func (img *SQcow2Image) Read(offset, count int64) ([]byte, int64) {
	return ReadQcow2(img.fd, offset, count)
}

func (img *SQcow2Image) Close() {
	CloseQcow2(img.fd)
}

func (img *SQcow2Image) Length() int64 {
	return Qcow2GetLenth(img.fd)
}

type SFile struct {
	fd *os.File
}

func (f *SFile) Open(imagePath string, readonly bool) error {
	var mode = os.O_RDWR
	if readonly {
		mode = os.O_RDONLY
	}
	fd, err := os.OpenFile(imagePath, mode, 0644)
	if err != nil {
		return err
	} else {
		f.fd = fd
		return nil
	}
}

func (f *SFile) Read(offset, count int64) ([]byte, int64) {
	buf := make([]byte, count)
	var readCount int64 = 0
	for readCount < count {
		cnt, err := f.fd.Read(buf[readCount:])
		readCount += int64(cnt)
		if err == io.EOF {
			return buf[0:readCount], readCount
		}
		if err != nil {
			return nil, -1
		}
	}
	return buf, readCount
}

func (f *SFile) Close() {
	f.fd.Close()
}

func (f *SFile) Length() int64 {
	stat, e := f.fd.Stat()
	if e != nil {
		return -1
	}
	return stat.Size()
}
