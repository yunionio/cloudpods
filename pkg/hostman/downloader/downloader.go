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

package downloader

import (
	"compress/zlib"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

const (
	CHUNK_SIZE         = 1024 * 8
	DEFAULT_RATE_LIMIT = 50
	COMPRESS_LEVEL     = 1
)

type SDownloadProvider struct {
	w         http.ResponseWriter
	rateLimit int
	compress  bool
}

func NewDownloadProvider(w http.ResponseWriter, compress bool, rateLimit int) *SDownloadProvider {
	if rateLimit <= 0 {
		rateLimit = DEFAULT_RATE_LIMIT
	}
	return &SDownloadProvider{w, rateLimit, compress}
}

func (d *SDownloadProvider) Start(
	prepareDownload func() error, onDownloadComplete func(),
	downloadFilePath string, headers http.Header,
) error {
	if prepareDownload != nil {
		if err := prepareDownload(); err != nil {
			log.Errorln(err)
			return err
		}
	}
	if headers.Get("Content-Type") == "" {
		headers.Set("Content-Type", "application/octet-stream")
	}
	for k := range headers {
		d.w.Header().Add(k, headers.Get(k))
	}

	log.Infof("Downloader Start Transfer %s, compress %t", downloadFilePath, d.compress)
	spath, err := filepath.EvalSymlinks(downloadFilePath)
	if err == nil {
		downloadFilePath = spath
	}
	fi, err := os.Open(downloadFilePath)
	if err != nil {
		return errors.Wrapf(err, "os.Open(%s)", downloadFilePath)
	}
	defer fi.Close()

	stat, err := fi.Stat()
	if err != nil {
		return errors.Wrapf(err, "fi.Stat")
	}
	d.w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))

	var (
		end                  = false
		chunk                = make([]byte, CHUNK_SIZE)
		writer     io.Writer = d.w
		startTime            = time.Now()
		sendBytes            = 0
		writeChunk []byte
	)

	if d.compress {
		zw, err := zlib.NewWriterLevel(d.w, COMPRESS_LEVEL)
		if err != nil {
			log.Errorln(err)
			return err
		}
		writer = zw
		defer zw.Close()
		defer zw.Flush() // it's cool
	}

	for !end {
		size, err := fi.Read(chunk)
		if err != nil {
			if err != io.EOF {
				log.Errorln(err)
				return err
			} else {
				end = true
			}
		}

		writeChunk = chunk[:size]
		if size, err = writer.Write(writeChunk); err != nil {
			log.Errorln(err)
			return err
		} else {
			sendBytes += size
			timeDur := time.Now().Sub(startTime)
			exceptDur := float64(sendBytes) / 1000.0 / 1000.0 / float64(d.rateLimit)
			if exceptDur > timeDur.Seconds() {
				time.Sleep(time.Duration(exceptDur-timeDur.Seconds()) * time.Second)
			}
		}
	}

	// if d.compress {
	// 	zw := writer.(*zlib.Writer)
	// 	zw.Flush()
	// }

	sendMb := float64(sendBytes) / 1000.0 / 1000.0
	timeDur := time.Now().Sub(startTime)
	log.Infof("Send data: %fMB rate: %fMB/sec", sendMb, sendMb/timeDur.Seconds())

	if onDownloadComplete != nil {
		onDownloadComplete()
	}
	return nil
}
