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
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/multicloud/huawei/client/manager"
	"yunion.io/x/onecloud/pkg/multicloud/huawei/client/requests"
	"yunion.io/x/onecloud/pkg/multicloud/huawei/client/responses"
)

type ServiceNameType string

const HuaWeiDefaultDomain = "myhuaweicloud.com"

const (
	ServiceNameECS  ServiceNameType = "ecs"  // 弹性云服务
	ServiceNameCCE  ServiceNameType = "cce"  // 云容器服务
	ServiceNameAS   ServiceNameType = "as"   // 弹性伸缩服务
	ServiceNameIAM  ServiceNameType = "iam"  //  统一身份认证服务
	ServiceNameIMS  ServiceNameType = "ims"  // 镜像服务
	ServiceNameCSBS ServiceNameType = "csbs" // 云服务器备份服务
	ServiceNameCCI  ServiceNameType = "cci"  // 云容器实例 CCI
	ServiceNameBMS  ServiceNameType = "bms"  // 裸金属服务器
	ServiceNameEVS  ServiceNameType = "evs"  // 云硬盘 EVS
	ServiceNameVBS  ServiceNameType = "vbs"  // 云硬盘备份 VBS
	ServiceNameOBS  ServiceNameType = "obs"  // 对象存储服务 OBS
	ServiceNameVPC  ServiceNameType = "vpc"  // 虚拟私有云 VPC
	ServiceNameELB  ServiceNameType = "elb"  // 弹性负载均衡 ELB
	ServiceNameBSS  ServiceNameType = "bss"  // 合作伙伴运营能力
	ServiceNameNAT  ServiceNameType = "nat"  // Nat网关 NAT
	ServiceNameDCS  ServiceNameType = "dcs"  // 分布式缓存服务
)

type SManagerContext struct {
	InstanceManager manager.IManager
	InstanceId      string
}

func (self *SManagerContext) GetPath() string {
	path := self.InstanceManager.KeyString()
	if len(self.InstanceId) > 0 {
		path += fmt.Sprintf("/%s", url.PathEscape(self.InstanceId))
	}

	return path
}

type SResourceManager struct {
	SBaseManager
	ctx           manager.IManagerContext
	ServiceName   ServiceNameType // 服务名称： ecs
	Region        string          // 区域： cn-north-1
	ProjectId     string          // 项目ID： uuid
	version       string          // api 版本号
	Keyword       string          // 资源名称单数。构建URL时使用
	KeywordPlural string          // 资源名称复数形式。构建URL时使用

	ResourceKeyword string // 资源名称。url中使用
}

func getContent(params jsonutils.JSONObject) string {
	if params == nil {
		return ""
	}

	return params.String()
}

func (self *SResourceManager) Version() string {
	return self.version
}

func (self *SResourceManager) KeyString() string {
	return self.ResourceKeyword
}

func (self *SResourceManager) ServiceType() string {
	return string(self.ServiceName)
}

func (self *SResourceManager) GetColumns() []string {
	return []string{}
}

func (self *SResourceManager) getReourcePath(ctx manager.IManagerContext, rid string, spec string) string {
	segs := []string{}
	if ctx != nil {
		segs = append(segs, ctx.GetPath())
	}

	segs = append(segs, self.KeyString())

	if len(rid) > 0 {
		segs = append(segs, url.PathEscape(rid))
	}

	if len(spec) > 0 {
		specSegs := strings.Split(spec, "/")
		for _, specSeg := range specSegs {
			segs = append(segs, url.PathEscape(specSeg))
		}
	}

	return strings.Join(segs, "/")
}

func (self *SResourceManager) newRequest(method, rid, spec string, ctx manager.IManagerContext) *requests.SRequest {
	resourcePath := self.getReourcePath(ctx, rid, spec)
	return requests.NewResourceRequest(method, string(self.ServiceName), self.version, self.Region, self.ProjectId, resourcePath)
}

func (self *SResourceManager) List(queries map[string]string) (*responses.ListResult, error) {
	return self.ListInContext(self.ctx, queries)
}

func (self *SResourceManager) ListInContext(ctx manager.IManagerContext, queries map[string]string) (*responses.ListResult, error) {
	return self.ListInContextWithSpec(ctx, "", queries, self.KeywordPlural)
}

func (self *SResourceManager) ListInContextWithSpec(ctx manager.IManagerContext, spec string, queries map[string]string, responseKey string) (*responses.ListResult, error) {
	request := self.newRequest("GET", "", spec, ctx)
	for k, v := range queries {
		request.AddQueryParam(k, v)
	}

	return self._list(request, responseKey)
}

func (self *SResourceManager) Get(id string, queries map[string]string) (jsonutils.JSONObject, error) {
	return self.GetInContext(self.ctx, id, queries)
}

func (self *SResourceManager) GetInContext(ctx manager.IManagerContext, id string, queries map[string]string) (jsonutils.JSONObject, error) {
	return self.GetInContextWithSpec(ctx, id, "", queries, self.Keyword)
}

func (self *SResourceManager) GetInContextWithSpec(ctx manager.IManagerContext, id string, spec string, queries map[string]string, responseKey string) (jsonutils.JSONObject, error) {
	request := self.newRequest("GET", id, spec, ctx)
	for k, v := range queries {
		request.AddQueryParam(k, v)
	}

	return self._get(request, responseKey)
}

func (self *SResourceManager) Create(params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return self.CreateInContext(self.ctx, params)
}

func (self *SResourceManager) CreateInContext(ctx manager.IManagerContext, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return self.CreateInContextWithSpec(ctx, "", params, self.Keyword)
}

func (self *SResourceManager) CreateInContextWithSpec(ctx manager.IManagerContext, spec string, params jsonutils.JSONObject, responseKey string) (jsonutils.JSONObject, error) {
	request := self.newRequest("POST", "", spec, ctx)
	request.SetContent([]byte(params.String()))

	return self._do(request, responseKey)
}

func (self *SResourceManager) AsyncCreate(params jsonutils.JSONObject) (string, error) {
	return "", fmt.Errorf("not supported")
}

func (self *SResourceManager) Update(id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return self.UpdateInContext(self.ctx, id, params)
}

func (self *SResourceManager) UpdateInContext(ctx manager.IManagerContext, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return self.UpdateInContextWithSpec(ctx, id, "", params, self.Keyword)
}

func (self *SResourceManager) UpdateInContextWithSpec(ctx manager.IManagerContext, id string, spec string, params jsonutils.JSONObject, responseKey string) (jsonutils.JSONObject, error) {
	request := self.newRequest("PUT", id, spec, ctx)
	content := getContent(params)
	if len(content) > 0 {
		request.SetContent([]byte(content))
	}

	return self._do(request, responseKey)
}

func (self *SResourceManager) Delete(id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return self.DeleteInContext(self.ctx, id, params)
}

func (self *SResourceManager) DeleteInContext(ctx manager.IManagerContext, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return self.DeleteInContextWithSpec(ctx, id, "", nil, params, self.Keyword)
}

func (self *SResourceManager) DeleteInContextWithSpec(ctx manager.IManagerContext, id string, spec string, queries map[string]string, params jsonutils.JSONObject, responseKey string) (jsonutils.JSONObject, error) {
	request := self.newRequest("DELETE", id, spec, ctx)
	for k, v := range queries {
		request.AddQueryParam(k, v)
	}

	content := getContent(params)
	if len(content) > 0 {
		request.SetContent([]byte(content))
	}

	return self._do(request, responseKey)
}

func (self *SResourceManager) PerformAction(action string, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return self.PerformAction2(action, id, params, self.Keyword)
}

func (self *SResourceManager) PerformAction2(action string, id string, params jsonutils.JSONObject, responseKey string) (jsonutils.JSONObject, error) {
	request := self.newRequest("POST", id, action, nil)
	request.SetContent([]byte(getContent(params)))

	return self._do(request, responseKey)
}

func (self *SResourceManager) SetVersion(v string) {
	self.version = v
}

func (self *SResourceManager) versionedURL(path string) string {
	return ""
}

// todo: Init a manager with environment variables
func (self *SResourceManager) Init() error {
	return nil
}
