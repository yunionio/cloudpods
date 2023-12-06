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
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"sync"

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

type sFileList struct {
	Name string
	Path string
	Mode fs.FileMode
}

func HandleSftpList(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, query, _ := appsrv.FetchEnv(ctx, w, r)
	dir := "/"
	if query.Contains("path") {
		dir, _ = query.GetString("path")
	}
	sId := params[SESSION_ID]
	files, err := func() ([]sFileList, error) {
		sftp, err := getSftpClient(sId)
		if err != nil {
			return nil, errors.Wrapf(err, "getSftpClient")
		}
		files, err := sftp.ReadDir(dir)
		if err != nil {
			return nil, errors.Wrapf(err, "ReadDir %s", dir)
		}
		ret := []sFileList{}
		for _, f := range files {
			ret = append(ret, sFileList{
				Name: f.Name(),
				Mode: f.Mode(),
				Path: path.Join(dir, f.Name()),
			})
		}
		return ret, nil
	}()
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
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
			return errors.Wrapf(err, "stat %s", dir)
		}

		newFile, err := sftp.Create(path.Join(dir, header.Filename))
		if err != nil {
			return errors.Wrapf(err, "create file")
		}

		defer file.Close()
		defer newFile.Close()

		_, err = newFile.ReadFrom(file)
		if err != nil {
			return errors.Wrapf(err, "ReadFrom")
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
			return errors.Wrapf(err, "stat %s", dir)
		}
		if file.IsDir() {
			return fmt.Errorf("dir %s can not download", dir)
		}

		reader, err := sftp.Open(dir)
		if err != nil {
			return errors.Wrapf(err, "open file")
		}
		defer reader.Close()

		w.Header().Add("Content-Disposition", "attachment;filename="+file.Name())
		w.Header().Add("Content-Type", "application/octet-stream")
		_, err = io.Copy(w, reader)
		return err
	}()
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
}
