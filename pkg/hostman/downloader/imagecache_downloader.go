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
	"path"

	"yunion.io/x/onecloud/pkg/hostman/storageman"
)

type SImageCacheDownloadProvider struct {
	*SDownloadProvider
	imageId string
}

func NewImageCacheDownloadProvider(
	w http.ResponseWriter, compress bool, rateLimit int, imageId string,
) *SImageCacheDownloadProvider {
	return &SImageCacheDownloadProvider{
		SDownloadProvider: NewDownloadProvider(w, compress, rateLimit),
		imageId:           imageId,
	}
}

func (s *SImageCacheDownloadProvider) getHeaders() http.Header {
	return http.Header{}
}

func (s *SImageCacheDownloadProvider) downloadFilePath() string {
	return path.Join(
		storageman.GetManager().LocalStorageImagecacheManager.GetPath(), s.imageId)
}

func (s *SImageCacheDownloadProvider) Start() error {
	return s.SDownloadProvider.Start(nil, nil, s.downloadFilePath(), s.getHeaders())
}
