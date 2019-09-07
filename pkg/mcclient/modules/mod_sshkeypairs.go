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
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type SSshkeypairManager struct {
	modulebase.ResourceManager
}

func (this *SSshkeypairManager) List(s *mcclient.ClientSession, params jsonutils.JSONObject) (*modulebase.ListResult, error) {
	url := "/sshkeypairs"
	if params != nil {
		if queryStr := params.QueryString(); queryStr != "" {
			url = fmt.Sprintf("%s?%s", url, queryStr)
		}
	}
	body, err := modulebase.Get(this.ResourceManager, s, url, "sshkeypair")
	if err != nil {
		return nil, err
	}
	result := modulebase.ListResult{Data: []jsonutils.JSONObject{body}}
	return &result, nil
}

var (
	Sshkeypairs SSshkeypairManager
)

func init() {
	Sshkeypairs = SSshkeypairManager{NewComputeManager("sshkeypair", "sshkeypairs",
		[]string{},
		[]string{})}

	registerComputeV2(&Sshkeypairs)
}
