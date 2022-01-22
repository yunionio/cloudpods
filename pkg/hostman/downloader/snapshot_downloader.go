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
	"net/http"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

type SSnapshotDownloadProvider struct {
	*SDownloadProvider
	snapshotPath string
}

func NewSnapshotDownloadProvider(
	w http.ResponseWriter, compress, sparse bool, rateLimit int, snapshotPath string,
) *SSnapshotDownloadProvider {
	return &SSnapshotDownloadProvider{
		SDownloadProvider: NewDownloadProvider(w, compress, sparse, rateLimit),
		snapshotPath:      snapshotPath,
	}
}

func (s *SSnapshotDownloadProvider) getHeaders() http.Header {
	hdrs := http.Header{}
	hdrs.Set("X-Image-Meta-Disk_format", "")
	return hdrs
}

func (s *SSnapshotDownloadProvider) HandlerHead() error {
	headers := s.getHeaders()
	if fileutils2.Exists(s.snapshotPath) {
		chksum, err := fileutils2.MD5(s.snapshotPath)
		if err != nil {
			return err
		}
		headers.Set("X-Image-Meta-Checksum", chksum)
	}
	s.w.WriteHeader(200)
	return nil
}

func (s *SSnapshotDownloadProvider) downloadFilePath() string {
	return s.snapshotPath
}

func (s *SSnapshotDownloadProvider) Start() error {
	return s.SDownloadProvider.Start(nil, nil, s.downloadFilePath(), s.getHeaders())
}
