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
	"os"
	"path"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/tarutils"
)

type SGuestDownloadProvider struct {
	*SDownloadProvider
	serverId string
}

func NewGuestDownloadProvider(
	w http.ResponseWriter, compress, sparse bool, rateLimit int, sid string,
) *SGuestDownloadProvider {
	return &SGuestDownloadProvider{
		SDownloadProvider: NewDownloadProvider(w, compress, sparse, rateLimit),
		serverId:          sid,
	}
}

func (s *SGuestDownloadProvider) fullPath() string {
	return path.Join(options.HostOptions.ServersPath, s.serverId)
}

func (s *SGuestDownloadProvider) getHeaders() http.Header {
	hdrs := http.Header{}
	hdrs.Set("X-Image-Meta-Disk_format", "tar")
	return hdrs
}

func (i *SGuestDownloadProvider) onDownloadComplete() {
	if fileutils2.Exists(i.downloadFilePath()) {
		os.Remove(i.downloadFilePath())
	}
}

func (s *SGuestDownloadProvider) downloadFilePath() string {
	return s.fullPath() + ".tar"
}

func (s *SGuestDownloadProvider) prepareDownload() error {
	log.Infof("Compress %s to %s", s.fullPath(), s.downloadFilePath())
	return tarutils.TarSparseFile(s.fullPath(), s.downloadFilePath())
}

func (s *SGuestDownloadProvider) Start() error {
	return s.SDownloadProvider.Start(s.prepareDownload,
		s.onDownloadComplete, s.downloadFilePath(), s.getHeaders())
}
