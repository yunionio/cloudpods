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
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type ProcessInstanceManager struct {
	modulebase.ResourceManager
}

var (
	ProcessInstance ProcessInstanceManager
)

func (self *ProcessInstanceManager) Upload(s *mcclient.ClientSession, header http.Header, body io.Reader) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s", self.URLPath())
	resp, err := modulebase.RawRequest(self.ResourceManager, s, "POST", path, header, body)
	_, json, err := s.ParseJSONResponse("", resp, err)
	if err != nil {
		return nil, err
	}
	return json.Get(self.Keyword)
}

func init() {
	ProcessInstance = ProcessInstanceManager{NewITSMManager("process-instance", "process-instances",
		[]string{"id", "process_instance_id", "root_process_instance_id", "case_instance_id", "business_key", "ended", "suspended", "tenant_id"},
		[]string{},
	)}
	register(&ProcessInstance)
}
