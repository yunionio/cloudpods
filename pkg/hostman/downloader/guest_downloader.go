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
	w http.ResponseWriter, compress bool, rateLimit int, sid string,
) *SGuestDownloadProvider {
	return &SGuestDownloadProvider{
		SDownloadProvider: NewDownloadProvider(w, compress, rateLimit),
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
