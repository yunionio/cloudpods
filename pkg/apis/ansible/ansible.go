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

package ansible

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/util/ansible"
)

type AnsiblePlaybookCreateInput struct {
	apis.Meta

	Name     string
	Playbook ansible.Playbook
}

type AnsiblePlaybookUpdateInput AnsiblePlaybookCreateInput

type AnsibleHost struct {
	User     string `json:"user"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Name     string `json:"name"`
	Password string `json:"password"`
	OsType   string `json:"os_type"`
}

type AnsiblePlaybookReferenceCreateInput struct {
	apis.SharableVirtualResourceCreateInput

	SAnsiblePlaybookReference
	PlaybookParams map[string]interface{} `json:"playbook_params"`
}

type AnsiblePlaybookReferenceUpdateInput struct {
}

type AnsiblePlaybookReferenceRunInput struct {
	Host AnsibleHost
	Args jsonutils.JSONObject
}

type AnsiblePlaybookReferenceRunOutput struct {
	AnsiblePlaybookInstanceId string
}

type AnsiblePlaybookReferenceStopInput struct {
	AnsiblePlaybookInstanceId string
}

type AnsiblePlaybookInstanceListInput struct {
	apis.StatusStandaloneResourceListInput
	AnsiblePlayboookReferenceId string
}
