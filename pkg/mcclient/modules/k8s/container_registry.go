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

	"yunion.io/x/jsonutils"
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
	return &ContainerRegistryManager{
		ResourceManager: NewResourceManager("container_registry", "container_registries",
			NewResourceCols("Url", "Type"),
			NewColumns()),
	}
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
