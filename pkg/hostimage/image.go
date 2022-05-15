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

package hostimage

/*
#cgo pkg-config: glib-2.0 zlib

#include "libqemuio.h"
#include "qemu/osdep.h"
*/
import "C"

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"unsafe"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

var qemuBlkCache sync.Map

type QemuioBlkDev struct {
	imagePath string
	readonly  bool
	encrypted bool
	refCount  int32

	blk *C.struct_QemuioBlk
}

func init() {
	C.qemuio_init()
}

func (qb *QemuioBlkDev) ReadQcow2(offset int64, count int64) ([]byte, int64) {
	if qb.blk == nil || offset < 0 || count < 0 {
		return nil, -1
	}
	var b = make([]byte, count)
	var total = C.int64_t(0)
	ret := C.read_qcow2(qb.blk, unsafe.Pointer(&b[0]), C.int64_t(offset), C.int64_t(count), &total)
	if ret < 0 {
		return nil, int64(ret)
	} else {
		return b, int64(total)
	}
}

func OpenQcow2(disk *qemuimg.SImageInfo, readonly bool) *QemuioBlkDev {
	key := fmt.Sprintf("%s_%v", disk.Path, readonly)
	if blk, ok := qemuBlkCache.Load(key); ok {
		qb := blk.(*QemuioBlkDev)
		atomic.AddInt32(&qb.refCount, 1)
		return qb
	}

	qb := &QemuioBlkDev{
		imagePath: disk.Path,
		readonly:  readonly,
	}
	diskPath := C.CString(disk.Path)
	imageOpts := C.CString(disk.ImageOptions())

	// sec options
	var secretOpts *C.char
	secOpt := disk.SecretOptions()
	if len(secOpt) > 0 {
		secretOpts = C.CString(secOpt)
		qb.encrypted = true
	}

	// qemu io open image
	blk := C.open_qcow2(diskPath, imageOpts, secretOpts, C.bool(readonly))

	C.free(unsafe.Pointer(diskPath))
	C.free(unsafe.Pointer(imageOpts))

	if secretOpts != nil {
		C.free(unsafe.Pointer(secretOpts))
	}

	if blk == nil {
		// failed open qemu image
		return nil
	}

	qb.blk = blk
	qb.refCount = 1
	qemuBlkCache.Store(key, qb)
	return qb
}

func LoadQcow2(imagePath string, readonly bool) *QemuioBlkDev {
	key := fmt.Sprintf("%s_%v", imagePath, readonly)
	if blk, ok := qemuBlkCache.Load(key); ok {
		return blk.(*QemuioBlkDev)
	}
	return nil
}

func (qb *QemuioBlkDev) Qcow2GetLength() int64 {
	if qb.blk == nil {
		return -1
	} else {
		return int64(C.qcow2_get_length(qb.blk))
	}
}

func (qb *QemuioBlkDev) CloseQcow2() {
	if qb.blk != nil && atomic.AddInt32(&qb.refCount, -1) == 0 {
		var secId *C.char
		if qb.encrypted {
			secId = C.CString(filepath.Base(qb.imagePath))
		}

		C.close_qcow2(qb.blk, secId)
		qemuBlkCache.Delete(fmt.Sprintf("%s_%v", qb.imagePath, qb.readonly))

		if secId != nil {
			C.free(unsafe.Pointer(secId))
		}
	}
}

type IImage interface {
	// Open image file and its backing file (if have)
	Open(imagePath string, readonly bool, encryptInfo *apis.SEncryptInfo) error

	// load opend qcow2 img form qemu blk cache
	Load(imagePath string, readonly, reference bool) error

	// Close may not really close image file handle, just reudce ref count
	Close()

	// If return number < 0 indicate read failed
	Read(offset, count int64) ([]byte, error)

	// Get image file length, not file actual length, it's image virtual size
	Length() int64
}

type SQcow2Image struct {
	fd *QemuioBlkDev
}

func (img *SQcow2Image) newQemuImage(imagePath string, encryptInfo *apis.SEncryptInfo) *qemuimg.SImageInfo {
	info := &qemuimg.SImageInfo{
		Path: imagePath,
	}

	if encryptInfo != nil {
		info.SetSecId(filepath.Base(imagePath))
		info.Password = encryptInfo.Key
		info.EncryptAlg = encryptInfo.Alg
	}
	return info
}

func (img *SQcow2Image) Open(imagePath string, readonly bool, encryptInfo *apis.SEncryptInfo) error {
	disk := img.newQemuImage(imagePath, encryptInfo)
	fd := OpenQcow2(disk, readonly)
	if fd == nil {
		return fmt.Errorf("open image %s failed", imagePath)
	} else {
		img.fd = fd
		return nil
	}
}

func (img *SQcow2Image) Load(imagePath string, readonly, reference bool) error {
	fd := LoadQcow2(imagePath, readonly)
	if fd == nil {
		return fmt.Errorf("image %s readonly: %v not found", imagePath, readonly)
	} else {
		atomic.AddInt32(&fd.refCount, 1)
		img.fd = fd
		return nil
	}
}

func (img *SQcow2Image) Read(offset, count int64) ([]byte, error) {
	buf, ret := img.fd.ReadQcow2(offset, count)
	if ret >= 0 {
		return buf[:ret], nil
	} else {
		return nil, errors.Errorf("failed read data %d", ret)
	}
}

func (img *SQcow2Image) Close() {
	img.fd.CloseQcow2()
}

func (img *SQcow2Image) Length() int64 {
	return img.fd.Qcow2GetLength()
}

type SFile struct {
	fd *os.File
}

func (f *SFile) Open(imagePath string, readonly bool, encryptInfo *apis.SEncryptInfo) error {
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

func (f *SFile) Load(imagePath string, readonly, reference bool) error {
	return fmt.Errorf("File don't support load")
}

func (f *SFile) Read(offset, count int64) ([]byte, error) {
	if _, err := f.fd.Seek(offset, io.SeekStart); err != nil {
		log.Errorf("seek file %s", err)
		return nil, errors.Wrap(err, "seek")
	}
	buf := make([]byte, count)
	r := bufio.NewReader(f.fd)
	n, err := r.Read(buf)
	if err != nil {
		return nil, errors.Wrap(err, "read")
	}
	return buf[:n], nil
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
