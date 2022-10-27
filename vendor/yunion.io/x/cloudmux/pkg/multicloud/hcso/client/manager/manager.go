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

package manager

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/hcso/client/auth"
	"yunion.io/x/cloudmux/pkg/multicloud/hcso/client/responses"
)

type IManagerContext interface {
	GetPath() string
}

type IBaseManager interface {
	Version() string
	KeyString() string
	ServiceType() string
	GetColumns() []string
}

type IManager interface {
	IBaseManager
	// 获取资源列表 GET <base_url>/cloudservers/?<queries>
	List(queries map[string]string) (*responses.ListResult, error)
	// 根据上文获取资源列表 GET <base_url>/cloudservers/<cloudserver_id>/nics?<queries>
	ListInContext(ctx IManagerContext, queries map[string]string) (*responses.ListResult, error)
	ListInContextWithSpec(ctx IManagerContext, spec string, queries map[string]string, responseKey string) (*responses.ListResult, error)

	// 查询单个资源 GET <base_url>/cloudservers/<cloudserver_id>?<queries>
	Get(id string, queries map[string]string) (jsonutils.JSONObject, error)
	// 根据上文获取资源查询单个资源 GET <base_url>/cloudservers/<cloudserver_id>/nics/<nic_id>?<queries>
	GetInContext(ctx IManagerContext, id string, queries map[string]string) (jsonutils.JSONObject, error)

	// 创建单个资源 POST <base_url>/cloudservers
	Create(params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	// 根据上文创建单个资源 POST <base_url>/cloudservers/<cloudserver_id>/nics/<nic_id>
	CreateInContext(ctx IManagerContext, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	// 异步任务创建 POST <base_url>/cloudservers. 返回异步任务 job_id。 todo：// 后续考虑返回一个task对象
	AsyncCreate(params jsonutils.JSONObject) (string, error)

	// 更新单个资源 PUT <base_url>/cloudservers/<cloudserver_id>
	Update(id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	// 根据上文更新单个资源 PUT <base_url>/cloudservers/<cloudserver_id>/nics/<nic_id>
	UpdateInContext(ctx IManagerContext, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	// 根据上文和spec更新单个资源 PUT /v2.1/{project_id}/servers/{server_id}/os-reset-password
	UpdateInContextWithSpec(ctx IManagerContext, id string, spec string, params jsonutils.JSONObject, responseKey string) (jsonutils.JSONObject, error)

	// 删除单个资源  DELETE <base_url>/cloudservers/<cloudserver_id>
	Delete(id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	// 根据上文删除单个资源 DELETE <base_url>/cloudservers/<cloudserver_id>/nics/<nic_id>
	DeleteInContext(ctx IManagerContext, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	// 根据上文和spec删除单个资源
	DeleteInContextWithSpec(ctx IManagerContext, id string, spec string, queries map[string]string, params jsonutils.JSONObject, responseKey string) (jsonutils.JSONObject, error)
	// 批量执行操作 POST <base_url>/cloudservers/<action>
	// BatchPerformAction(action string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	// 执行操作 POST <base_url>/cloudservers/<cloudserver_id>/<action>
	PerformAction(action string, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
}

type IManagerConfig interface {
	GetSigner() auth.Signer
	GetEndpoints() *cloudprovider.SHCSOEndpoints
	GetRegionId() string
	GetDomainId() string
	GetProjectId() string
	GetDebug() bool

	GetDefaultRegion() string
}
