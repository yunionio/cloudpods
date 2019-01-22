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
	w http.ResponseWriter, compress bool, rateLimit int, snapshotPath string,
) *SSnapshotDownloadProvider {
	return &SSnapshotDownloadProvider{
		SDownloadProvider: NewDownloadProvider(w, compress, rateLimit),
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
