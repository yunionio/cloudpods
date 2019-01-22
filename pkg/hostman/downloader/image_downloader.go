package downloader

import (
	"fmt"
	"net/http"
	"os"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/onecloud/pkg/util/tarutils"
	"yunion.io/x/pkg/utils"
)

type SImageDownloadProvider struct {
	*SDownloadProvider
	disk           storageman.IDisk
	compressFormat string
}

func NewImageDownloadProvider(w http.ResponseWriter, compress bool, rateLimit int, disk storageman.IDisk, compressFormat string) *SImageDownloadProvider {
	return &SImageDownloadProvider{
		SDownloadProvider: NewDownloadProvider(w, compress, rateLimit),
		disk:              disk,
		compressFormat:    compressFormat,
	}
}

func (i *SImageDownloadProvider) fullPath() string {
	return i.disk.GetPath()
}

func (i *SImageDownloadProvider) downloadFilePath() string {
	if utils.IsInStringArray(i.compressFormat, []string{"qcow2", "tar"}) {
		return i.fullPath() + "." + i.compressFormat
	} else {
		return i.fullPath()
	}
}

func (i *SImageDownloadProvider) prepareDownload() error {
	log.Infof(fmt.Sprintf("Compress %s to %s", i.fullPath(), i.downloadFilePath()))
	switch i.compressFormat {
	case "qcow2":
		img, err := qemuimg.NewQemuImage(i.fullPath())
		if err != nil {
			return err
		}
		_, err = img.CloneQcow2(i.downloadFilePath(), true)
		return err
	case "tar":
		return tarutils.TarSparseFile(i.fullPath(), i.downloadFilePath())
	default:
		return nil
	}
}

func (i *SImageDownloadProvider) onDownloadComplete() {
	if i.downloadFilePath() != i.fullPath() && fileutils2.Exists(i.downloadFilePath()) {
		os.Remove(i.downloadFilePath())
	}
}

func (i *SImageDownloadProvider) getHeaders() http.Header {
	hdrs := http.Header{}
	if utils.IsInStringArray(i.compressFormat, []string{"qcow2", "tar"}) {
		hdrs.Set("X-Image-Meta-Disk_format", i.compressFormat)
	}
	return hdrs
}

func (i *SImageDownloadProvider) Start() error {
	return i.SDownloadProvider.Start(i.prepareDownload, i.onDownloadComplete,
		i.downloadFilePath(), i.getHeaders())
}

func (i *SImageDownloadProvider) HandlerHead() error {
	headers := i.getHeaders()
	if len(i.compressFormat) > 0 {
		headers.Set("X-Image-Meta-Checksum", "error")
	} else {
		checksum, err := fileutils2.MD5(i.fullPath())
		if err != nil {
			return err
		}
		headers.Set("X-Image-Meta-Checksum", checksum)
	}
	for k := range headers {
		i.w.Header().Add(k, headers.Get(k))
	}
	i.w.WriteHeader(200)
	return nil
}
