package modules

import (
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/util/huawei/client/manager"
	"yunion.io/x/onecloud/pkg/util/huawei/client/requests"
	"yunion.io/x/onecloud/pkg/util/huawei/client/responses"
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

)

type ResourceManager struct {
	BaseManager
	ServiceName   ServiceNameType // 服务名称： ecs
	Region        string          // 区域： cn-north-1
	ProjectId     string          // 项目ID： uuid
	version       string          // api 版本号
	Keyword       string          // 资源名称单数。构建URL时使用
	KeywordPlural string          // 资源名称复数形式。构建URL时使用

	ResourceKeyword string // 资源名称。url中使用
}

func (self *ResourceManager) Version() string {
	return self.version
}

func (self *ResourceManager) KeyString() string {
	return self.ResourceKeyword
}

func (self *ResourceManager) ServiceType() string {
	return string(self.ServiceName)
}

func (self *ResourceManager) GetColumns() []string {
	return []string{}
}

func (self *ResourceManager) getReourcePath(ctx *manager.ManagerContext, rid string, spec string) string {
	segs := []string{}
	if ctx != nil {
		segs = append(segs, ctx.GetPath())
	}

	segs = append(segs, self.KeyString())

	if len(rid) > 0 {
		segs = append(segs, url.PathEscape(rid))
	}

	if len(spec) > 0 {
		segs = append(segs, url.PathEscape(spec))
	}

	return strings.Join(segs, "/")
}

func (self *ResourceManager) newRequest(method, rid, spec string, ctx *manager.ManagerContext) *requests.SRequest {
	resourcePath := self.getReourcePath(ctx, rid, spec)
	return requests.NewResourceRequest(method, string(self.ServiceName), self.version, self.Region, self.ProjectId, resourcePath)
}

func (self *ResourceManager) List(querys map[string]string) (*responses.ListResult, error) {
	return self.ListInContext(nil, "", querys)
}

func (self *ResourceManager) ListInContext(ctx *manager.ManagerContext, spec string, querys map[string]string) (*responses.ListResult, error) {
	request := self.newRequest("GET", "", spec, ctx)
	for k, v := range querys {
		request.AddQueryParam(k, v)
	}

	//if _, ok := request.GetQueryParams()["limit"]; !ok {
	//	request.AddQueryParam("limit", "50")
	//}

	//if _, ok := request.GetQueryParams()["offset"]; !ok {
	//	request.AddQueryParam("offset", "0")
	//}

	return self._list(request, self.KeywordPlural)
}

func (self *ResourceManager) Get(id string, querys map[string]string) (jsonutils.JSONObject, error) {
	return self.GetInContext(nil, id, querys)
}

func (self *ResourceManager) GetInContext(ctx *manager.ManagerContext, id string, querys map[string]string) (jsonutils.JSONObject, error) {
	request := self.newRequest("GET", id, "", nil)
	for k, v := range querys {
		request.AddQueryParam(k, v)
	}

	return self._get(request, self.Keyword)
}

func (self *ResourceManager) Create(id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	panic("implement me")
}

func (self *ResourceManager) CreateInContext(ctx *manager.ManagerContext, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	panic("implement me")
}

func (self *ResourceManager) Update(id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	panic("implement me")
}

func (self *ResourceManager) UpdateInContext(ctx *manager.ManagerContext, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	panic("implement me")
}

func (self *ResourceManager) Delete(id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	panic("implement me")
}

func (self *ResourceManager) DeleteInContext(id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	panic("implement me")
}

func (self *ResourceManager) BatchPerformAction(action string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	panic("implement me")
}

func (self *ResourceManager) PerformAction(action string, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	panic("implement me")
}

func (self *ResourceManager) SetVersion(v string) {
	self.version = v
}

func (self *ResourceManager) versionedURL(path string) string {
	return ""
}

// todo: Init a manager with environment variables
func (self *ResourceManager) Init() error {
	return nil
}
