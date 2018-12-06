package manager

import (
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/util/huawei/client/responses"
)

type ManagerContext struct {
	InstanceManager IManager
	InstanceId      string
}

func (self *ManagerContext) GetPath() string {
	path := self.InstanceManager.KeyString()
	if len(self.InstanceId) > 0 {
		path += fmt.Sprintf("/%s", url.PathEscape(self.InstanceId))
	}

	return path
}

type IBaseManager interface {
	Version() string
	KeyString() string
	ServiceType() string
	GetColumns() []string
}

type IManager interface {
	IBaseManager
	// 获取资源列表 GET <base_url>/cloudservers/?<querys>
	List(querys map[string]string) (*responses.ListResult, error)
	// 根据上文获取资源列表 GET <base_url>/cloudservers/<cloudserver_id>/nics?<querys>
	ListInContext(ctx *ManagerContext, spec string, querys map[string]string) (*responses.ListResult, error)

	// 查询单个资源 GET <base_url>/cloudservers/<cloudserver_id>?<querys>
	Get(id string, querys map[string]string) (jsonutils.JSONObject, error)
	// 根据上文获取资源查询单个资源 GET <base_url>/cloudservers/<cloudserver_id>/nics/<nic_id>?<querys>
	GetInContext(ctx *ManagerContext, id string, querys map[string]string) (jsonutils.JSONObject, error)

	// 创建单个资源 POST <base_url>/cloudservers
	Create(id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	// 根据上文创建单个资源 POST <base_url>/cloudservers/<cloudserver_id>/nics/<nic_id>
	CreateInContext(ctx *ManagerContext, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)

	// 更新单个资源 PUT <base_url>/cloudservers/<cloudserver_id>
	Update(id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	// 根据上文更新单个资源 PUT <base_url>/cloudservers/<cloudserver_id>/nics/<nic_id>
	UpdateInContext(ctx *ManagerContext, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)

	// 删除单个资源  DELETE <base_url>/cloudservers/<cloudserver_id>
	Delete(id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	// 根据上文删除单个资源 DELETE <base_url>/cloudservers/<cloudserver_id>/nics/<nic_id>
	DeleteInContext(id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)

	// 批量执行操作 POST <base_url>/cloudservers/<action>
	BatchPerformAction(action string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	// 执行操作 POST <base_url>/cloudservers/<cloudserver_id>/<action>
	PerformAction(action string, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
}
