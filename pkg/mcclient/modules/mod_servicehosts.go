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

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
)

type ServiceNodeManager struct {
	ResourceManager
}

func (this *ServiceNodeManager) DoDeleteServiceHost(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s", this.KeywordPlural)

	body := jsonutils.NewDict()
	if params != nil {
		body.Add(params, this.Keyword)
	}

	return this._delete(s, path, body, this.Keyword)
}

var (
	ServiceHosts ServiceNodeManager
)

func init() {
	ServiceHosts = ServiceNodeManager{NewMonitorManager("service_host", "service_hosts",
		[]string{"id", "host_id", "host_name", "ip", "vcpu_count", "vmem_size", "disk", "project", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "project_id", "remark"},
		[]string{})}

	register(&ServiceHosts)
}
