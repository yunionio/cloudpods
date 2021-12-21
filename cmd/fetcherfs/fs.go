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

package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"bazil.org/fuse/fs"
	"github.com/pierrec/lz4/v4"
	"github.com/pkg/errors"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/bitmap"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

// FetcherFs implements the feteher file system.
type FetcherFs struct {
	blocksize    int64
	url          string
	size         int64
	receivedSize int64
	blockCount   int64
	fetchedCount int64
	blockBitMap  *bitmap.BitMap
	readLock     sync.Mutex

	localFile *os.File
}

func (FetcherFs) Root() (fs.Node, error) {
	return Dir{}, nil
}

var fetcherFs *FetcherFs

func initFetcherFs() (*FetcherFs, error) {
	fetcherFs = &FetcherFs{
		blocksize: int64(opt.Blocksize) * 1024 * 1024,
		url:       opt.Url,
	}
	if err := fetcherFs.fetchMetaInfo(); err != nil {
		return nil, err
	}
	segs := strings.Split(opt.Url, "/")
	if len(segs[len(segs)-1]) == 0 {
		return nil, errors.Errorf("bad url: %s", opt.Url)
	}
	fd, err := ioutil.TempFile(opt.Tmpdir, fmt.Sprintf("%s.*", segs[len(segs)-1]))
	if err != nil {
		return nil, errors.Wrap(err, "create tempfile")
	}
	err = syscall.Fallocate(int(fd.Fd()), 0, 0, fetcherFs.size)
	if err != nil {
		os.Remove(path.Join(opt.Tmpdir, fd.Name()))
		return nil, errors.Wrap(err, "fallocate temp file")
	}
	fetcherFs.localFile = fd
	return fetcherFs, nil
}

func destoryInitFetcherFs() error {
	if fetcherFs == nil {
		return nil
	}
	log.Errorf("Remove path %s", fetcherFs.localFile.Name())
	err := os.Remove(fetcherFs.localFile.Name())
	if err != nil {
		return err
	}
	return nil
}

func (fs *FetcherFs) fetchMetaInfo() error {
	header := NewRequestHeader()
	res, _, err := httputils.JSONRequest(
		httputils.GetDefaultClient(),
		context.Background(),
		http.MethodHead,
		opt.Url,
		header,
		nil,
		false,
	)
	if err != nil {
		return err
	}
	if lengthStr := res.Get("Content-Length"); lengthStr != "" {
		fs.size, err = strconv.ParseInt(lengthStr, 10, 0)
		if err != nil {
			return errors.Wrap(err, "parse Content-Length")
		}
		fs.blockCount = fs.size / fs.blocksize
		if fs.size%fs.blocksize > 0 {
			fs.blockCount += 1
		}
		fs.blockBitMap = bitmap.NewBitMap(fs.blockCount)
	} else {
		return errors.Errorf("not found Content-Length")
	}
	return nil
}

func (fs *FetcherFs) GetSize() int64 {
	return fs.size
}

func (fs *FetcherFs) GetBlockCount() int64 {
	return fs.blockCount
}

func (fs *FetcherFs) GetFetchedCount() int64 {
	return fs.fetchedCount
}

func (fs *FetcherFs) GetReceivedSize() int64 {
	return fs.receivedSize
}

func (fs *FetcherFs) GetMeta() string {
	return fmt.Sprintf(
		"pid=%d\nblock_count=%d\nfetch_count=%d\nrecvsize=%d\ntoken=%s\n",
		os.Getpid(), fs.GetBlockCount(), fs.GetFetchedCount(),
		fs.GetReceivedSize(), opt.Token,
	)
}

func (fs *FetcherFs) doRead(size int, offset int64) ([]byte, error) {
	fs.readLock.Lock()
	defer fs.readLock.Unlock()
	var (
		start, end = fs.offsetToBlockIndexRange(size, offset)
		err        error
	)

	for idx := start; idx <= end; idx++ {
		err = fs.fetchData(idx)
		if err != nil {
			return nil, errors.Wrap(err, "fetch data")
		}
	}

	_, err = fs.localFile.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, errors.Wrap(err, "seek local file")
	}
	var buf = make([]byte, size)
	n, err := fs.localFile.Read(buf)
	if err != nil {
		return nil, errors.Wrap(err, "do read")
	}
	return buf[:n], nil
}

func (fs *FetcherFs) fetchData(idx int64) (err error) {
	for i := 0; i < 3; i++ {
		if err = fs.doFetchData(idx); err != nil {
			log.Errorf("fetch data %d failed: %s", idx, err)
		} else {
			break
		}
	}
	return
}

func (fs *FetcherFs) blockReady(idx int64) bool {
	return fs.blockBitMap.Has(idx)
}
func (fs *FetcherFs) setBlockReady(idx int64) {
	fs.blockBitMap.Set(idx)
}

func (fs *FetcherFs) doFetchData(idx int64) error {
	if fs.blockReady(idx) {
		return nil
	}
	log.Infof("start do fetch idx %d ", idx)

	var start = idx * fs.blocksize
	var end = start + fs.blocksize - 1
	if end >= fs.size {
		end = fs.size - 1
	}

	var header = NewRequestHeader()
	header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	log.Errorf("Range %s", header.Get("Range"))
	var (
		res, err = httputils.Request(
			httputils.GetDefaultClient(),
			context.Background(),
			http.MethodGet,
			fs.url, header,
			nil, false,
		)
	)
	if err != nil {
		return errors.Wrap(err, "http fetch data")
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		body, _ := ioutil.ReadAll(res.Body)
		return errors.Errorf("failed fetch data: %d %s", res.StatusCode, body)
	}

	var lz4Reader = lz4.NewReader(res.Body)
	if err := lz4Reader.Apply(lz4.ConcurrencyOption(-1)); err != nil {
		return errors.Errorf("Apply lz4 option: %v", err)
	}

	buf, err := ioutil.ReadAll(lz4Reader)
	if len(buf) != int(end-start+1) {
		return errors.Wrap(err, "written local file")
	}

	written, err := fs.localFile.WriteAt(buf, start)
	if err != nil {
		return errors.Wrap(err, "write local file")
	}
	if err := fs.localFile.Sync(); err != nil {
		log.Errorf("sync failed %s", err)
	}

	// on write success
	fs.setBlockReady(idx)
	fs.receivedSize += int64(written)
	fs.fetchedCount += 1
	return nil
}

// fetcherfs byte range to block range
func (fs *FetcherFs) offsetToBlockIndexRange(size int, offset int64) (int64, int64) {
	return offset / int64(fs.blocksize), (offset + int64(size) - 1) / fs.blocksize
}

func (fs *FetcherFs) destory() error {
	var header = NewRequestHeader()
	_, err := httputils.Request(
		httputils.GetDefaultClient(),
		context.Background(),
		http.MethodPost,
		fs.url, header,
		nil, false,
	)
	return err
}

func NewRequestHeader() http.Header {
	header := http.Header{}
	header.Set("X-Auth-Token", opt.Token)
	return header
}
