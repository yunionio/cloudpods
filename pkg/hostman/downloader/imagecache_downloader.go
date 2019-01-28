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
