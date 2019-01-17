package remotefile

import (
	"compress/zlib"
	"context"
	"io"
	"net/http"
	"os"
	"syscall"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type SImageDesc struct {
	Name   string
	Format string
	Id     string
	Chksum string
	Path   string
	Size   int64
}

type SRemoteFile struct {
	ctx          context.Context
	url          string
	downloadUrl  string
	localPath    string
	tmpPath      string
	preChksum    string
	compress     bool
	timeout      time.Duration
	extraHeaders map[string]string

	chksum string
	format string
	name   string
}

func NewRemoteFile(
	ctx context.Context, url, localPath string, compress bool,
	PreChksum string, timeout int, extraHeaders map[string]string,
	tmpPath string, downloadUrl string,
) *SRemoteFile {
	if timeout <= 0 {
		timeout = 24 * 3600 //24 hours
	}
	if len(tmpPath) == 0 {
		tmpPath = localPath
	}

	return &SRemoteFile{
		ctx:          ctx,
		url:          url,
		localPath:    localPath,
		compress:     compress,
		preChksum:    PreChksum,
		timeout:      time.Duration(timeout) * time.Second,
		extraHeaders: extraHeaders,
		tmpPath:      tmpPath,
		downloadUrl:  downloadUrl,
	}
}

func (r *SRemoteFile) Fetch() bool {
	if len(r.preChksum) > 0 {
		return r.fetch(r.preChksum)
	} else if fileutils2.Exists(r.localPath) {
		if !r.VerifyIntegrity() {
			return r.fetch("")
		} else {
			return true
		}
	} else {
		return r.fetch("")
	}
}

func (r *SRemoteFile) GetInfo() *SImageDesc {
	fi, err := os.Stat(r.localPath)
	if err != nil {
		log.Errorln(err)
		return nil
	}

	return &SImageDesc{
		Name:   r.name,
		Format: r.format,
		Chksum: r.chksum,
		Path:   r.localPath,
		Size:   fi.Size(),
	}
}

func (r *SRemoteFile) VerifyIntegrity() bool {
	if r.download(false, "") {
		localChksum, err := fileutils2.MD5(r.localPath)
		if err != nil {
			log.Errorln(err)
			return false
		}
		if localChksum == r.chksum {
			log.Infof("identical chksum, skip download")
			return true
		}
	}
	return r.fetch("")
}

func (r *SRemoteFile) fetch(preChksum string) bool {
	var (
		fetchSucc = false
		retryCnt  = 0
	)

	for !fetchSucc && retryCnt < 3 {
		r.format = ""
		r.chksum = ""
		fetchSucc = r.download(true, preChksum)
		if fetchSucc {
			if len(r.chksum) > 0 && fileutils2.Exists(r.tmpPath) {
				if localChksum, err := fileutils2.MD5(r.tmpPath); err != nil {
					log.Errorln(err)
					fetchSucc = false
				} else if r.chksum != localChksum {
					fetchSucc = false
				}
			}

			if !fetchSucc {
				retryCnt += 1
			} else if r.localPath != r.tmpPath {
				if fileutils2.Exists(r.localPath) {
					if err := syscall.Unlink(r.localPath); err != nil {
						log.Errorln(err)
					}
				}
				if err := syscall.Rename(r.tmpPath, r.localPath); err != nil {
					log.Errorln(err)
				}
			}
		}
	}
	return fetchSucc
}

func (r *SRemoteFile) download(getData bool, preChksum string) bool {
	fi, err := os.Create(r.tmpPath)
	if err != nil {
		log.Errorln(err)
		return false
	}
	defer fi.Close()

	var header = http.Header{}
	header.Set("X-Auth-Token", auth.GetTokenString())
	if len(preChksum) > 0 {
		header.Set("X-Image-Meta-Checksum", preChksum)
	}
	if r.compress {
		header.Set("X-Compress-Content", "zlib")
	}
	if len(r.extraHeaders) > 0 {
		for k, v := range r.extraHeaders {
			header.Set(k, v)
		}
	}
	var method, url = "HEAD", r.url
	if getData {
		if len(r.downloadUrl) > 0 {
			url = r.downloadUrl
		}
		method = "GET"
	}

	httpCli := httputils.GetTimeoutClient(r.timeout)
	resp, err := httputils.Request(httpCli, r.ctx,
		httputils.THttpMethod(method), url, header, nil, false)
	if err != nil {
		log.Errorln(err)
		return false
	} else {
		if resp.StatusCode < 300 {
			defer resp.Body.Close()
			if getData {
				var reader = resp.Body

				if r.compress {
					zlibRC, err := zlib.NewReader(resp.Body)
					if err != nil {
						log.Errorf("New zlib Reader error: %s", err)
						return false
					}
					defer zlibRC.Close()
					reader = zlibRC
				}

				_, err := io.Copy(fi, reader)
				if err != nil {
					log.Errorln(err)
					return false
				}
			}
			r.setProperties(resp.Header)
			return true
		} else if resp.StatusCode == 304 {
			if err := os.Remove(r.tmpPath); err != nil {
				log.Errorf("Fail to remove file %s", r.tmpPath)
			}
			return true
		} else {
			log.Errorf("Remote file fetch error %d", resp.StatusCode)
			return false
		}
	}
	return false

}

func (r *SRemoteFile) setProperties(header http.Header) {
	r.chksum = header.Get("X-Image-Meta-Checksum")
	r.format = header.Get("X-Image-Meta-Disk_format")
	r.name = header.Get("X-Image-Meta-Name")
}
