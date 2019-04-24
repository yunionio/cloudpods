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

package modules

import (
	"fmt"
	"io"
	"net/http"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
)

type InfoManager struct {
	ResourceManager
}

var (
	Info InfoManager
)

func (this *InfoManager) Update(s *mcclient.ClientSession, header http.Header, body io.Reader) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s", this.URLPath())
	resp, err := this.rawRequest(s, "POST", path, header, body)
	_, json, err := s.ParseJSONResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return json.Get(this.Keyword)
}

func init() {
	Info = InfoManager{NewYunionAgentManager("info", "infos",
		[]string{},
		[]string{})}
	register(&Info)
}
