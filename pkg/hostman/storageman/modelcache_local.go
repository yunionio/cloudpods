package storageman

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"syscall"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/log"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/pkg/errors"
)

const (
	MODEL_CACHE_DOWNLOADING_SUFFIX = ".tmp"
)

type SLocalModelCache struct {
	blobId  string
	Manager *SLocalImageCacheManager
	Size    int64

	// cond *sync.Conds

	AccessAt time.Time
}

func NewLocalModelCache(blobId string, imagecacheManager *SLocalImageCacheManager) *SLocalModelCache {
	modelCache := new(SLocalModelCache)
	modelCache.blobId = blobId
	modelCache.Manager = imagecacheManager
	// modelCache.cond = sync.NewCond(new(sync.Mutex))
	return modelCache
}

func (m *SLocalModelCache) GetPath() string {
	return path.Join(m.Manager.GetModelPath(), m.blobId)
}

func (m *SLocalModelCache) GetTmpPath() string {
	return m.GetPath() + MODEL_CACHE_DOWNLOADING_SUFFIX
}

func (m *SLocalModelCache) GetSizeMb() int64 {
	return m.Size / 1024 / 1024
}

func (m *SLocalModelCache) Access(ctx context.Context, modelName string) error {
	mPath := m.GetPath()
	// check exist, if not, fetch it
	if !fileutils2.Exists(mPath) {
		return m.fetch(modelName)
	}
	// update ATime in both struct and file
	now := time.Now()
	m.AccessAt = now
	fi := m.getFileInfo()
	if fi == nil {
		return errors.Errorf("No file info for %s", mPath)
	}
	return os.Chtimes(mPath, now, fi.ModTime())
}

func (m *SLocalModelCache) Load() error {
	var (
		blobPath = m.GetPath()
	)
	if !fileutils2.Exists(m.GetPath()) {
		return errors.Wrapf(cloudprovider.ErrNotFound, blobPath)
	}

	m.updateCacheInfo()

	return nil
}

func (m *SLocalModelCache) Remove(ctx context.Context) error {
	if fileutils2.Exists(m.GetPath()) {
		if err := syscall.Unlink(m.GetPath()); err != nil {
			return errors.Wrap(err, m.GetPath())
		}
	}

	if fileutils2.Exists(m.GetTmpPath()) {
		if err := syscall.Unlink(m.GetTmpPath()); err != nil {
			return errors.Wrap(err, m.GetTmpPath())
		}
	}

	return nil
}

func (m *SLocalModelCache) fetch(model string) error {
	var (
		url      = fmt.Sprintf(api.LLM_OLLAMA_LIBRARY_BASE_URL, fmt.Sprintf("%s/blobs/%s", model, m.blobId))
		filePath = m.GetPath()
		tmpPath  = m.GetTmpPath()
	)

	req, _ := http.NewRequest("GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if nil != err {
		return err
	}
	defer resp.Body.Close()

	// check get status
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(tmpPath)
	if nil != err {
		return err
	}
	defer out.Close()

	// copy to get resp
	if _, err := io.Copy(out, resp.Body); nil != err {
		os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, filePath); nil != err {
		return err
	}

	m.updateCacheInfo()

	return nil
}

func (m *SLocalModelCache) getFileInfo() fs.FileInfo {
	if fi, err := os.Stat(m.GetPath()); err != nil {
		log.Errorln(err)
		return nil
	} else {
		return fi
	}
}

func (m *SLocalModelCache) updateCacheInfo() {
	fi := m.getFileInfo()
	if fi != nil {
		m.Size = fi.Size()
		if fi.Sys() != nil {
			atime := fi.Sys().(*syscall.Stat_t).Atim
			m.AccessAt = time.Unix(atime.Sec, atime.Nsec)
		}
	}
}
