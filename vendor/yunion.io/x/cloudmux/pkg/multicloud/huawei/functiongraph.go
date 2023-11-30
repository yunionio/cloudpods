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

package huawei

import (
	"net/url"
	"time"
)

type SFunction struct {
	FuncUrn      string    `json:"func_urn"`
	FuncName     string    `json:"func_name"`
	DomainId     string    `json:"domain_id"`
	Namespace    string    `json:"namespace"`
	ProjectName  string    `json:"project_name"`
	Package      string    `json:"package"`
	Runtime      string    `json:"runtime"`
	Timeout      int       `json:"timeout"`
	Handler      string    `json:"handler"`
	MemorySize   int       `json:"memory_size"`
	CPU          int       `json:"cpu"`
	CodeType     string    `json:"code_type"`
	CodeFilename string    `json:"code_filename"`
	CodeSize     int       `json:"code_size"`
	DomainNames  string    `json:"domain_names"`
	UserData     string    `json:"user_data"`
	Digest       string    `json:"digest"`
	Version      string    `json:"version"`
	ImageName    string    `json:"image_name"`
	Xrole        string    `json:"xrole"`
	AppXrole     string    `json:"app_xrole"`
	LastModified time.Time `json:"last_modified"`
	FuncCode     struct {
	} `json:"func_code"`
	FuncCode0 struct {
	} `json:"FuncCode"`
	Concurrency    int `json:"concurrency"`
	ConcurrentNum  int `json:"concurrent_num"`
	StrategyConfig struct {
		Concurrency   int `json:"concurrency"`
		ConcurrentNum int `json:"concurrent_num"`
	} `json:"strategy_config"`
	InitializerHandler  string `json:"initializer_handler"`
	InitializerTimeout  int    `json:"initializer_timeout"`
	EnterpriseProjectId string `json:"enterprise_project_id"`
	FuncVpcId           string `json:"func_vpc_id"`
	LongTime            bool   `json:"long_time"`
	LogConfig           struct {
		GroupName  string `json:"group_name"`
		GroupId    string `json:"group_id"`
		StreamName string `json:"stream_name"`
		StreamId   string `json:"stream_id"`
		SwitchTime int    `json:"switch_time"`
	} `json:"log_config"`
	Type                string `json:"type"`
	EnableCloudDebug    string `json:"enable_cloud_debug"`
	EnableDynamicMemory bool   `json:"enable_dynamic_memory"`
	CustomImage         struct {
	} `json:"custom_image"`
	IsStatefulFunction       bool   `json:"is_stateful_function"`
	IsBridgeFunction         bool   `json:"is_bridge_function"`
	IsReturnStream           bool   `json:"is_return_stream"`
	EnableAuthInHeader       bool   `json:"enable_auth_in_header"`
	ReservedInstanceIdleMode bool   `json:"reserved_instance_idle_mode"`
	EnableSnapshot           bool   `json:"enable_snapshot"`
	EphemeralStorage         int    `json:"ephemeral_storage"`
	IsShared                 bool   `json:"is_shared"`
	EnableClassIsolation     bool   `json:"enable_class_isolation"`
	ResourceId               string `json:"resource_id"`
}

func (self *SRegion) ListFunctions() ([]SFunction, error) {
	query := url.Values{}
	ret := []SFunction{}
	for {
		resp, err := self.list(SERVICE_FUNCTIONGRAPH, "fgs/functions", query)
		if err != nil {
			return nil, err
		}
		part := struct {
			Functions  []SFunction
			NextMarker string
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Functions...)
		if len(part.NextMarker) == 0 || len(part.Functions) == 0 {
			break
		}
		query.Set("marker", part.NextMarker)
	}
	return ret, nil
}
