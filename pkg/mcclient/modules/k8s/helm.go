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

package k8s

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Repos *RepoManager
)

func init() {
	Repos = NewRepoManager()
	modules.Register(Repos)
}

type RepoManager struct {
	*ResourceManager
}

func NewRepoManager() *RepoManager {
	return &RepoManager{
		ResourceManager: NewResourceManager("repo", "repos",
			NewResourceCols("Url", "Is_Public", "Source", "Type", "Backend"),
			NewColumns()),
	}
}

func (m *RepoManager) UploadChart(s *mcclient.ClientSession, id string, params jsonutils.JSONObject, body io.Reader, size int64) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/%s/upload-chart?%s", m.URLPath(), id, params.QueryString())
	headers := http.Header{}
	headers.Add("Content-Type", "application/octet-stream")
	if size > 0 {
		headers.Add("Content-Length", fmt.Sprintf("%d", size))
	}
	resp, err := modulebase.RawRequest(*m.ResourceManager.ResourceManager, s, httputils.POST, path, headers, body)
	_, json, err := s.ParseJSONResponse("", resp, err)
	if err != nil {
		return nil, err
	}
	return json, nil
}

func (m *RepoManager) DownloadChart(s *mcclient.ClientSession, id string, chartName string, chartVersion string) (string, io.Reader, int64, error) {
	params := map[string]string{
		"chart_name": chartName,
		"version":    chartVersion,
	}
	query := jsonutils.Marshal(params)
	path := fmt.Sprintf("/%s/%s/download-chart", m.URLPath(), url.PathEscape(id))
	queryString := query.QueryString()
	if len(queryString) > 0 {
		path = fmt.Sprintf("%s?%s", path, queryString)
	}
	resp, err := modulebase.RawRequest(*m.ResourceManager.ResourceManager, s, "GET", path, nil, nil)
	if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		sizeBytes, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
		if err != nil {
			log.Errorf("Download chart unknown size")
			sizeBytes = -1
		}
		return resp.Header.Get("Chart-Filename"), resp.Body, sizeBytes, nil
	}
	_, _, err = s.ParseJSONResponse("", resp, err)
	return "", nil, -1, err
}
