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

package server

import (
	"context"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
)

const (
	SESSION_ID = "<session-id>"
)

var (
	sftpMux     = sync.Mutex{}
	sftpClients = make(map[string]*sftp.Client)
)

func addSftpClient(sId string, client *sftp.Client) {
	sftpMux.Lock()
	defer sftpMux.Unlock()
	sftpClients[sId] = client
}

func delSftpClient(sId string) {
	sftpMux.Lock()
	defer sftpMux.Unlock()
	delete(sftpClients, sId)
}

func getSftpClient(sId string) (*sftp.Client, error) {
	sftpMux.Lock()
	defer sftpMux.Unlock()
	client, ok := sftpClients[sId]
	if !ok {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "%s", sId)
	}
	return client, nil
}

type sLinkFile struct {
	Name      string
	Path      string
	Size      int64
	IsDir     bool
	Mode      string
	IsRegular bool
	ModeNum   fs.FileMode
}

type sFileList struct {
	Name      string
	Path      string
	Size      int64
	ModTime   time.Time
	IsDir     bool
	Mode      string
	ModeNum   fs.FileMode
	IsRegular bool

	LinkFile *sLinkFile
}

type Files []sFileList

func (files Files) Len() int {
	return len(files)
}

func (files Files) Swap(i, j int) {
	files[i], files[j] = files[j], files[i]
}

func (files Files) Less(i, j int) bool {
	if files[i].IsDir != files[i].IsDir {
		// 文件夹在上
		var v = func(b bool) int {
			if b {
				return 0
			}
			return 1
		}
		return v(files[i].IsDir) < v(files[j].IsDir)
	}
	return files[i].Name < files[j].Name
}

func HandleSftpList(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, query, _ := appsrv.FetchEnv(ctx, w, r)
	dir := "/"
	if query.Contains("path") {
		dir, _ = query.GetString("path")
	}
	sId := params[SESSION_ID]
	files, err := func() (Files, error) {
		client, err := getSftpClient(sId)
		if err != nil {
			return nil, errors.Wrapf(err, "getSftpClient")
		}
		files, err := client.ReadDir(dir)
		if err != nil {
			return nil, errors.Wrapf(httperrors.FsErrorNormalize(err), "ReadDir %s", dir)
		}
		ret := Files{}
		for _, f := range files {
			vv := sFileList{
				Name:      f.Name(),
				Mode:      f.Mode().String(),
				ModeNum:   f.Mode().Perm(),
				IsRegular: f.Mode().IsRegular(),
				Size:      f.Size(),
				ModTime:   f.ModTime(),
				IsDir:     f.IsDir(),
				Path:      path.Join(dir, f.Name()),
			}
			f.Mode().IsRegular()
			if f.Mode().Type() == fs.ModeSymlink {
				if link, err := client.ReadLink(vv.Path); err == nil {
					vv.LinkFile = &sLinkFile{
						Name: link,
					}
					if stat, err := client.Stat(path.Join(dir, link)); err == nil {
						vv.LinkFile.IsDir = stat.IsDir()
						vv.LinkFile.Path = path.Join(dir, link)
						vv.LinkFile.Size = stat.Size()
						vv.LinkFile.Mode = stat.Mode().String()
						vv.LinkFile.ModeNum = stat.Mode().Perm()
						vv.LinkFile.IsRegular = stat.Mode().IsRegular()
					}
				} else {
					err = httperrors.FsErrorNormalize(err)
				}
			}
			ret = append(ret, vv)
		}
		return ret, nil
	}()
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	sort.Sort(files)
	appsrv.SendJSON(w, jsonutils.Marshal(files))
}

func HandleSftpUpload(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, query, _ := appsrv.FetchEnv(ctx, w, r)
	dir := "/"
	if query.Contains("path") {
		dir, _ = query.GetString("path")
	}
	sId := params[SESSION_ID]

	err := func() error {
		sftp, err := getSftpClient(sId)
		if err != nil {
			return errors.Wrapf(err, "getSftpClient")
		}

		r.ParseMultipartForm(32 << 20)
		file, header, err := r.FormFile("file")
		if err != nil {
			return errors.Wrapf(err, "FormFile")
		}

		defer file.Close()

		_, err = sftp.Stat(dir)
		if err != nil {
			return errors.Wrapf(httperrors.FsErrorNormalize(err), "stat %s", dir)
		}

		newFile, err := sftp.Create(path.Join(dir, header.Filename))
		if err != nil {
			return errors.Wrapf(httperrors.FsErrorNormalize(err), "create file")
		}

		defer file.Close()
		defer newFile.Close()

		_, err = newFile.ReadFrom(file)
		if err != nil {
			return errors.Wrapf(httperrors.FsErrorNormalize(err), "ReadFrom")
		}
		return nil
	}()
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, jsonutils.Marshal(map[string]string{"status": "success"}))
}

func HandleSftpDownload(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, query, _ := appsrv.FetchEnv(ctx, w, r)
	if !query.Contains("path") {
		httperrors.GeneralServerError(ctx, w, httperrors.NewMissingParameterError("path"))
		return
	}
	dir, _ := query.GetString("path")
	sId := params[SESSION_ID]

	err := func() error {
		sftp, err := getSftpClient(sId)
		if err != nil {
			return errors.Wrapf(err, "getSftpClient")
		}
		file, err := sftp.Stat(dir)
		if err != nil {
			return errors.Wrapf(httperrors.FsErrorNormalize(err), "stat %s", dir)
		}
		if file.IsDir() {
			return errors.Wrapf(httperrors.ErrInvalidStatus, "dir %s can not be downloaded", dir)
		}

		reader, err := sftp.Open(dir)
		if err != nil {
			return errors.Wrapf(httperrors.FsErrorNormalize(err), "open file")
		}
		defer reader.Close()

		w.Header().Add("Content-Disposition", "attachment;filename*=utf-8''"+strings.ReplaceAll(url.QueryEscape(file.Name()), "+", "%20"))
		w.Header().Add("Content-Type", "application/octet-stream")
		_, err = io.Copy(w, reader)
		if err != nil {
			return errors.Wrap(httperrors.FsErrorNormalize(err), "Copy")
		}
		return nil
	}()
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
}
