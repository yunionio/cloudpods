package manager

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/util/huawei/client/responses"
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
	// 获取资源列表 GET <base_url>/cloudservers/?<querys>
	List(querys map[string]string) (*responses.ListResult, error)
	// 根据上文获取资源列表 GET <base_url>/cloudservers/<cloudserver_id>/nics?<querys>
	ListInContext(ctx IManagerContext, querys map[string]string) (*responses.ListResult, error)
	ListInContextWithSpec(ctx IManagerContext, spec string, querys map[string]string, responseKey string) (*responses.ListResult, error)

	// 查询单个资源 GET <base_url>/cloudservers/<cloudserver_id>?<querys>
	Get(id string, querys map[string]string) (jsonutils.JSONObject, error)
	// 根据上文获取资源查询单个资源 GET <base_url>/cloudservers/<cloudserver_id>/nics/<nic_id>?<querys>
	GetInContext(ctx IManagerContext, id string, querys map[string]string) (jsonutils.JSONObject, error)

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

	// 批量执行操作 POST <base_url>/cloudservers/<action>
	// BatchPerformAction(action string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	// 执行操作 POST <base_url>/cloudservers/<cloudserver_id>/<action>
	PerformAction(action string, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
}
