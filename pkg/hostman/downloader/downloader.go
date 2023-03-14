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

	"yunion.io/x/onecloud/pkg/util/pb"
	"yunion.io/x/onecloud/pkg/util/sparsefile"
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
	sparse    bool
}

func NewDownloadProvider(w http.ResponseWriter, compress, sparse bool, rateLimit int) *SDownloadProvider {
	if rateLimit <= 0 {
		rateLimit = DEFAULT_RATE_LIMIT
	}
	return &SDownloadProvider{w: w, rateLimit: rateLimit, compress: compress, sparse: sparse}
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

	log.Infof("Downloader Start Transfer %s, compress %t sparse %t rateLimit: %dMiB/s", downloadFilePath, d.compress, d.sparse, d.rateLimit)
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

	var reader io.Reader
	reader = fi

	size := stat.Size()
	d.w.Header().Set("X-File-Size", fmt.Sprintf("%d", size))
	if d.sparse {
		sparse, err := sparsefile.NewSparseFileReader(fi)
		if err != nil {
			return errors.Wrapf(err, "NewSparseFileReader")
		}
		size = sparse.Size()
		d.w.Header().Set("X-Sparse-Header", fmt.Sprintf("%d", sparse.HeaderSize()))
		reader = sparse
	}

	d.w.Header().Set("Content-Length", fmt.Sprintf("%d", size))

	var writer io.Writer = d.w

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

	pb := pb.NewProxyReader(reader, size)
	pb.SetRateLimit(d.rateLimit)
	pb.SetRefreshRate(time.Second * 10)
	pb.SetCallback(func() {
		log.Infof("transfer %s rate: %.2f MiB p/s percent: %.2f%%", downloadFilePath, pb.Rate(), pb.Percent())
	})

	_, err = io.Copy(writer, pb)
	if err != nil {
		return errors.Wrapf(err, "io.Copy")
	}

	if onDownloadComplete != nil {
		onDownloadComplete()
	}
	return nil
}
