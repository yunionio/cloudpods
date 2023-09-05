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
	ContainerRegistries *ContainerRegistryManager
)

func init() {
	ContainerRegistries = NewContainerRegistryManager()
	modules.Register(ContainerRegistries)
}

type ContainerRegistryManager struct {
	*ResourceManager
}

func NewContainerRegistryManager() *ContainerRegistryManager {
	man := &ContainerRegistryManager{
		ResourceManager: NewResourceManager("container_registry", "container_registries",
			NewResourceCols("Url", "Type"),
			NewColumns()),
	}
	man.SetSpecificMethods("images")
	return man
}

func (m *ContainerRegistryManager) UploadImage(s *mcclient.ClientSession, id string, params jsonutils.JSONObject, body io.Reader, size int64) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/%s/upload-image", m.URLPath(), id)
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

type DownloadImageByManagerInput struct {
	Insecure bool   `json:"insecure"`
	Image    string `json:"image"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (m *ContainerRegistryManager) DownloadImageByManager(s *mcclient.ClientSession, input *DownloadImageByManagerInput) (string, io.Reader, int64, error) {
	path := fmt.Sprintf("/%s/download-image", m.URLPath())
	return m.downloadImage(s, path, input)
}

func (m *ContainerRegistryManager) DownloadImage(s *mcclient.ClientSession, id string, imageName string, imageTag string) (string, io.Reader, int64, error) {
	params := map[string]string{
		"image_name": imageName,
		"tag":        imageTag,
	}
	path := fmt.Sprintf("/%s/%s/download-image", m.URLPath(), url.PathEscape(id))
	return m.downloadImage(s, path, params)
}

func (m *ContainerRegistryManager) downloadImage(s *mcclient.ClientSession, path string, params interface{}) (string, io.Reader, int64, error) {
	query := jsonutils.Marshal(params)
	queryString := query.QueryString()
	if len(queryString) > 0 {
		path = fmt.Sprintf("%s?%s", path, queryString)
	}
	resp, err := modulebase.RawRequest(*m.ResourceManager.ResourceManager, s, "GET", path, nil, nil)
	if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		sizeBytes, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
		if err != nil {
			log.Errorf("Download image unknown size")
			sizeBytes = -1
		}
		return resp.Header.Get("Image-Filename"), resp.Body, sizeBytes, nil
	}
	_, _, err = s.ParseJSONResponse("", resp, err)
	return "", nil, -1, err
}
