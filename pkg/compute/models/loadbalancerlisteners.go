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

package models

import (
	"context"
	"database/sql"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=loadbalancerlistener
// +onecloud:swagger-gen-model-plural=loadbalancerlisteners
type SLoadbalancerListenerManager struct {
	SLoadbalancerLogSkipper
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager

	SLoadbalancerResourceBaseManager
	SLoadbalancerAclResourceBaseManager
	SLoadbalancerCertificateResourceBaseManager
}

var LoadbalancerListenerManager *SLoadbalancerListenerManager

func init() {
	LoadbalancerListenerManager = &SLoadbalancerListenerManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SLoadbalancerListener{},
			"loadbalancerlisteners_tbl",
			"loadbalancerlistener",
			"loadbalancerlisteners",
		),
	}
	LoadbalancerListenerManager.SetVirtualObject(LoadbalancerListenerManager)
}

type SLoadbalancerHTTPRateLimiter struct {
	HTTPRequestRate       int `nullable:"true" list:"user" create:"optional" update:"user"` // 限定监听接收请示速率
	HTTPRequestRatePerSrc int `nullable:"true" list:"user" create:"optional" update:"user"` // 源IP监听请求最大速率
}

type SLoadbalancerRateLimiter struct {
	EgressMbps int `nullable:"true" list:"user" get:"user" create:"optional" update:"user" json:"egress_mbps"`
}

type SLoadbalancerHealthCheck struct {
	HealthCheck     string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"` // 健康检查开启状态 on|off
	HealthCheckType string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"` // 健康检查协议 HTTP|TCP

	HealthCheckDomain   string `charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"` // 健康检查域名 yunion.cn
	HealthCheckURI      string `charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"` // 健康检查路径 /
	HealthCheckHttpCode string `charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"` // HTTP正常状态码 http_2xx,http_3xx

	HealthCheckRise     int `nullable:"true" list:"user" create:"optional" update:"user"` //  健康检查健康阈值 3秒
	HealthCheckFall     int `nullable:"true" list:"user" create:"optional" update:"user"` //  健康检查不健康阈值 15秒
	HealthCheckTimeout  int `nullable:"true" list:"user" create:"optional" update:"user"` // 健康检查超时时间 10秒
	HealthCheckInterval int `nullable:"true" list:"user" create:"optional" update:"user"` // 健康检查间隔时间 5秒

	HealthCheckReq string `list:"user" create:"optional" update:"user"` // UDP监听健康检查的请求串
	HealthCheckExp string `list:"user" create:"optional" update:"user"` // UDP监听健康检查的响应串
}

type SLoadbalancerTCPListener struct{}
type SLoadbalancerUDPListener struct{}

// TODO sensible default for knobs
type SLoadbalancerHTTPListener struct {
	StickySession              string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`  // 会话保持开启状态 on|off
	StickySessionType          string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`  // Cookie处理方式 insert(植入cookie)|server(重写cookie)
	StickySessionCookie        string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"` // Cookie名称
	StickySessionCookieTimeout int    `nullable:"true" list:"user" create:"optional" update:"user"`                             // 会话超时时间

	XForwardedFor bool `nullable:"true" list:"user" create:"optional" update:"user"` // 获取客户端真实IP
	Gzip          bool `nullable:"true" list:"user" create:"optional" update:"user"` // Gzip数据压缩
}

type SLoadbalancerHTTPRedirect struct {
	Redirect       string `width:"16" nullable:"true" list:"user" create:"optional" update:"user" default:"off"` // 跳转类型
	RedirectCode   int    `nullable:"true" list:"user" create:"optional" update:"user"`                          // 跳转HTTP code
	RedirectScheme string `width:"16" nullable:"true" list:"user" create:"optional" update:"user"`               // 跳转uri scheme
	RedirectHost   string `nullable:"true" list:"user" create:"optional" update:"user"`                          // 跳转时变更Host
	RedirectPath   string `nullable:"true" list:"user" create:"optional" update:"user"`                          // 跳转时变更Path
}

// TODO
//
//   - CACertificate string
//   - Certificate2Id // multiple certificates for rsa, ecdsa
//   - Use certificate for tcp listener
//   - Customize ciphers?
type SLoadbalancerHTTPSListener struct {
	SLoadbalancerCertificateResourceBase

	TLSCipherPolicy string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	EnableHttp2     bool   `create:"optional" list:"user" update:"user"`
}

type SLoadbalancerListener struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase

	SLoadbalancerResourceBase `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	ListenerType      string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"`
	ListenerPort      int    `nullable:"false" list:"user" create:"required"`
	BackendGroupId    string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	BackendServerPort int    `nullable:"false" get:"user" list:"user" default:"0" create:"optional"`

	Scheduler string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user"`

	SendProxy string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user" default:"off"`

	ClientRequestTimeout  int `nullable:"true" list:"user" create:"optional" update:"user"` // 连接请求超时时间
	ClientIdleTimeout     int `nullable:"true" list:"user" create:"optional" update:"user"` // 连接空闲超时时间
	BackendConnectTimeout int `nullable:"true" list:"user" create:"optional" update:"user"` // 后端连接超时时间
	BackendIdleTimeout    int `nullable:"true" list:"user" create:"optional" update:"user"` // 后端连接空闲时间

	AclStatus                    string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	AclType                      string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	SLoadbalancerAclResourceBase `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	SLoadbalancerRateLimiter

	SLoadbalancerTCPListener
	SLoadbalancerUDPListener
	SLoadbalancerHTTPListener
	SLoadbalancerHTTPSListener

	SLoadbalancerHealthCheck
	SLoadbalancerHTTPRateLimiter
	SLoadbalancerHTTPRedirect
}

func (manager *SLoadbalancerListenerManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeProject
}

func (self *SLoadbalancerListener) GetOwnerId() mcclient.IIdentityProvider {
	lb, err := self.GetLoadbalancer()
	if err != nil {
		return nil
	}
	return lb.GetOwnerId()
}

func (manager *SLoadbalancerListenerManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	lbId, _ := data.GetString("loadbalancer_id")
	if len(lbId) > 0 {
		lb, err := db.FetchById(LoadbalancerManager, lbId)
		if err != nil {
			return nil, errors.Wrapf(err, "db.FetchById(LoadbalancerManager, %s)", lbId)
		}
		return lb.(*SLoadbalancer).GetOwnerId(), nil
	}
	return db.FetchProjectInfo(ctx, data)
}

func (man *SLoadbalancerListenerManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, manager db.FilterByOwnerProvider, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if ownerId != nil {
		sq := LoadbalancerManager.Query("id")
		switch scope {
		case rbacscope.ScopeProject:
			sq = sq.Equals("tenant_id", ownerId.GetProjectId())
			return q.In("loadbalancer_id", sq.SubQuery())
		case rbacscope.ScopeDomain:
			sq = sq.Equals("domain_id", ownerId.GetProjectDomainId())
			return q.In("loadbalancer_id", sq.SubQuery())
		}
	}
	return q
}

// 负载均衡监听器Listener列表
func (man *SLoadbalancerListenerManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerListenerListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = man.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = man.SLoadbalancerResourceBaseManager.ListItemFilter(ctx, q, userCred, query.LoadbalancerFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLoadbalancerResourceBaseManager.ListItemFilter")
	}

	if len(query.BackendGroup) > 0 {
		_, err := validators.ValidateModel(ctx, userCred, LoadbalancerBackendGroupManager, &query.BackendGroup)
		if err != nil {
			return nil, err
		}
		q = q.Equals("backend_group_id", query.BackendGroup)
	}

	if len(query.ListenerType) > 0 {
		q = q.In("listener_type", query.ListenerType)
	}
	if len(query.ListenerPort) > 0 {
		q = q.In("listener_port", query.ListenerPort)
	}
	if len(query.Scheduler) > 0 {
		q = q.In("scheduler", query.Scheduler)
	}
	if len(query.Certificate) > 0 {
		q = q.In("certificate_id", query.Certificate)
	}
	if len(query.SendProxy) > 0 {
		q = q.In("send_proxy", query.SendProxy)
	}
	if len(query.AclStatus) > 0 {
		q = q.In("acl_status", query.AclStatus)
	}
	if len(query.AclType) > 0 {
		q = q.In("acl_type", query.AclType)
	}
	if len(query.AclId) > 0 {
		q = q.Equals("acl_id", query.AclId)
	} else if len(query.Acl) > 0 {
		q = q.Equals("acl_id", query.Acl)
	}

	return q, nil
}

func (man *SLoadbalancerListenerManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerListenerListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SLoadbalancerResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.LoadbalancerFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLoadbalancerResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (man *SLoadbalancerListenerManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SLoadbalancerResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

type sListener struct {
	Name           string
	LoadbalancerId string
}

func (self *SLoadbalancerListener) GetUniqValues() jsonutils.JSONObject {
	return jsonutils.Marshal(sListener{Name: self.Name, LoadbalancerId: self.LoadbalancerId})
}

func (manager *SLoadbalancerListenerManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	info := sListener{}
	data.Unmarshal(&info)
	return jsonutils.Marshal(info)
}

func (manager *SLoadbalancerListenerManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	info := sListener{}
	values.Unmarshal(&info)
	if len(info.LoadbalancerId) > 0 {
		q = q.Equals("loadbalancer_id", info.LoadbalancerId)
	}
	if len(info.Name) > 0 {
		q = q.Equals("name", info.Name)
	}
	return q
}

func (man *SLoadbalancerListenerManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.LoadbalancerListenerCreateInput) (*api.LoadbalancerListenerCreateInput, error) {
	lbObj, err := validators.ValidateModel(ctx, userCred, LoadbalancerManager, &input.LoadbalancerId)
	if err != nil {
		return nil, err
	}
	lb := lbObj.(*SLoadbalancer)
	lbbgObj, err := validators.ValidateModel(ctx, userCred, LoadbalancerBackendGroupManager, &input.BackendGroupId)
	if err != nil {
		return nil, err
	}
	lbbg := lbbgObj.(*SLoadbalancerBackendGroup)
	if lbbg.LoadbalancerId != lb.Id {
		return nil, httperrors.NewConflictError("backendgroup_id not same with listener's loadbalancer")
	}
	region, err := lb.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegion")
	}
	if region.Provider == api.CLOUD_PROVIDER_AWS {
		input.Scheduler = api.LB_SCHEDULER_NOP
	}
	err = input.Validate()
	if err != nil {
		return nil, err
	}
	if utils.IsInStringArray(input.ListenerType, []string{api.LB_LISTENER_TYPE_TCP, api.LB_LISTENER_TYPE_UDP}) {

	}
	if input.AclStatus == api.LB_BOOL_ON {
		if len(input.AclId) == 0 {
			return nil, httperrors.NewMissingParameterError("acl_id")
		}
		_, err := validators.ValidateModel(ctx, userCred, LoadbalancerAclManager, &input.AclId)
		if err != nil {
			return nil, err
		}
	}
	if input.ListenerType == api.LB_LISTENER_TYPE_HTTPS {
		if len(input.CertificateId) == 0 {
			return nil, httperrors.NewMissingParameterError("certificate_id")
		}
		_, err := validators.ValidateModel(ctx, userCred, LoadbalancerCertificateManager, &input.CertificateId)
		if err != nil {
			return nil, err
		}
	}
	input, err = region.GetDriver().ValidateCreateLoadbalancerListenerData(ctx, userCred, ownerId, input, lb, lbbg)
	if err != nil {
		return nil, err
	}

	input.StatusStandaloneResourceCreateInput, err = man.SStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusStandaloneResourceCreateInput)
	if err != nil {
		return nil, err
	}
	return input, nil
}

func (man *SLoadbalancerListenerManager) CheckTypeV(listenerType string) validators.IValidator {
	switch listenerType {
	case api.LB_LISTENER_TYPE_HTTP, api.LB_LISTENER_TYPE_HTTPS:
		return validators.NewStringChoicesValidator("health_check_type", api.LB_HEALTH_CHECK_TYPES_TCP).Default(api.LB_HEALTH_CHECK_HTTP)
	case api.LB_LISTENER_TYPE_TCP:
		return validators.NewStringChoicesValidator("health_check_type", api.LB_HEALTH_CHECK_TYPES_TCP).Default(api.LB_HEALTH_CHECK_TCP)
	case api.LB_LISTENER_TYPE_UDP:
		return validators.NewStringChoicesValidator("health_check_type", api.LB_HEALTH_CHECK_TYPES_UDP).Default(api.LB_HEALTH_CHECK_UDP)
	}
	// should it happen, panic then
	return nil
}

func (lblis *SLoadbalancerListener) PerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformStatusInput) (jsonutils.JSONObject, error) {
	if _, err := lblis.SStatusStandaloneResourceBase.PerformStatus(ctx, userCred, query, input); err != nil {
		return nil, err
	}
	if lblis.Status == api.LB_STATUS_ENABLED {
		return nil, lblis.StartLoadBalancerListenerStartTask(ctx, userCred, "")
	}
	return nil, lblis.StartLoadBalancerListenerStopTask(ctx, userCred, "")
}

func (lblis *SLoadbalancerListener) StartLoadBalancerListenerStartTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerListenerStartTask", lblis, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lblis *SLoadbalancerListener) StartLoadBalancerListenerStopTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerListenerStopTask", lblis, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lblis *SLoadbalancerListener) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	if utils.IsInStringArray(lblis.Status, []string{api.LB_STATUS_ENABLED, api.LB_STATUS_DISABLED}) {
		params.Add(jsonutils.NewString(lblis.Status), "origin_status")
	}
	return nil, lblis.StartLoadBalancerListenerSyncstatusTask(ctx, userCred, params, "")
}

func (lblis *SLoadbalancerListener) StartLoadBalancerListenerSyncstatusTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerListenerSyncstatusTask", lblis, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (lblis *SLoadbalancerListener) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.LoadbalancerListenerUpdateInput) (*api.LoadbalancerListenerUpdateInput, error) {
	err := input.Validate()
	if err != nil {
		return nil, err
	}
	if input.AclStatus != nil && *input.AclStatus == api.LB_BOOL_ON {
		if input.AclId == nil {
			return nil, httperrors.NewMissingParameterError("acl_id")
		}
		_, err = validators.ValidateModel(ctx, userCred, LoadbalancerAclManager, input.AclId)
		if err != nil {
			return nil, err
		}
	}
	if lblis.ListenerType == api.LB_LISTENER_TYPE_HTTPS && input.CertificateId != nil && len(*input.CertificateId) > 0 {
		_, err = validators.ValidateModel(ctx, userCred, LoadbalancerCertificateManager, input.CertificateId)
		if err != nil {
			return nil, err
		}
	}
	input.StatusStandaloneResourceBaseUpdateInput, err = lblis.SStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.StatusStandaloneResourceBaseUpdateInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBase.ValidateUpdateData")
	}
	region, err := lblis.GetRegion()
	if err != nil {
		return nil, err
	}
	return region.GetDriver().ValidateUpdateLoadbalancerListenerData(ctx, userCred, lblis, input)
}

func (lblis *SLoadbalancerListener) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lblis.SStatusStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)

	if account := lblis.GetCloudaccount(); account != nil && !account.IsOnPremise {
		lblis.StartLoadBalancerListenerSyncTask(ctx, userCred, data, "")
	}
}

func (lblis *SLoadbalancerListener) StartLoadBalancerListenerSyncTask(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject, parentTaskId string) error {
	params := data.(*jsonutils.JSONDict)
	lblis.SetStatus(ctx, userCred, api.LB_SYNC_CONF, "")
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerListenerSyncTask", lblis, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	return task.ScheduleRun(nil)
}

func (manager *SLoadbalancerListenerManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LoadbalancerListenerDetails {
	rows := make([]api.LoadbalancerListenerDetails, len(objs))

	stdRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	lbRows := manager.SLoadbalancerResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	lbaclRows := manager.SLoadbalancerAclResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	lbcertRows := manager.SLoadbalancerCertificateResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	lbbgIds := make([]string, len(objs))
	lbIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.LoadbalancerListenerDetails{
			StatusStandaloneResourceDetails:     stdRows[i],
			LoadbalancerResourceInfo:            lbRows[i],
			LoadbalancerAclResourceInfo:         lbaclRows[i],
			LoadbalancerCertificateResourceInfo: lbcertRows[i],
		}
		lis := objs[i].(*SLoadbalancerListener)
		lbIds[i] = lis.LoadbalancerId
		lbbgIds[i] = lis.BackendGroupId
	}

	lbs := map[string]SLoadbalancer{}
	err := db.FetchStandaloneObjectsByIds(LoadbalancerManager, lbIds, &lbs)
	if err != nil {
		return rows
	}

	virObjs := make([]interface{}, len(objs))
	for i := range rows {
		if lb, ok := lbs[lbIds[i]]; ok {
			virObjs[i] = &lb
			rows[i].ProjectId = lb.ProjectId
		}
	}

	lbbgs := map[string]SLoadbalancerBackendGroup{}
	err = db.FetchModelObjectsByIds(LoadbalancerBackendGroupManager, "id", lbbgIds, &lbbgs)
	if err != nil {
		return rows
	}
	for i := range rows {
		if lbbg, ok := lbbgs[lbbgIds[i]]; ok {
			rows[i].BackendGroup = lbbg.Name
		}
	}

	return rows
}

func (lblis *SLoadbalancerListener) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lblis.SStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	lblis.StartLoadBalancerListenerCreateTask(ctx, userCred, data.(*jsonutils.JSONDict), "")
}

func (lblis *SLoadbalancerListener) StartLoadBalancerListenerCreateTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) {
	err := func() error {
		lblis.SetStatus(ctx, userCred, api.LB_CREATING, "")
		task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerListenerCreateTask", lblis, userCred, data, parentTaskId, "", nil)
		if err != nil {
			return errors.Wrapf(err, "NewTask")
		}
		return task.ScheduleRun(nil)
	}()
	if err != nil {
		lblis.SetStatus(ctx, userCred, api.LB_CREATE_FAILED, err.Error())
	}
}

func (lblis *SLoadbalancerListener) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, lblis.RealDelete(ctx, userCred)
}

func (lblis *SLoadbalancerListener) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, lblis.StartLoadBalancerListenerSyncTask(ctx, userCred, nil, "")
}

func (lblis *SLoadbalancerListener) StartLoadBalancerListenerDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	err := func() error {
		task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerListenerDeleteTask", lblis, userCred, params, parentTaskId, "", nil)
		if err != nil {
			return errors.Wrapf(err, "NewTask")
		}
		return task.ScheduleRun(nil)
	}()
	if err != nil {
		lblis.SetStatus(ctx, userCred, api.LB_STATUS_DELETE_FAILED, err.Error())
	}
	return err
}

func (lblis *SLoadbalancerListener) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	lblis.SetStatus(ctx, userCred, api.LB_STATUS_DELETING, "")
	return lblis.StartLoadBalancerListenerDeleteTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (self *SLoadbalancerListener) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	rules, err := self.GetLoadbalancerListenerRules()
	if err != nil {
		return errors.Wrapf(err, "GetLoadbalancerListenerRules")
	}
	for i := range rules {
		err := rules[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "RealDelete rule %s", rules[i].Id)
		}
	}
	return self.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (lblis *SLoadbalancerListener) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (lblis *SLoadbalancerListener) GetLoadbalancerListenerParams() (*cloudprovider.SLoadbalancerListenerCreateOptions, error) {
	listener := &cloudprovider.SLoadbalancerListenerCreateOptions{
		Name:                    lblis.Name,
		Description:             lblis.Description,
		ListenerType:            lblis.ListenerType,
		ListenerPort:            lblis.ListenerPort,
		Scheduler:               lblis.Scheduler,
		EnableHTTP2:             lblis.EnableHttp2,
		EgressMbps:              lblis.EgressMbps,
		EstablishedTimeout:      lblis.BackendConnectTimeout,
		AccessControlListStatus: lblis.AclStatus,

		ClientRequestTimeout:  lblis.ClientRequestTimeout,
		ClientIdleTimeout:     lblis.ClientIdleTimeout,
		BackendIdleTimeout:    lblis.BackendIdleTimeout,
		BackendConnectTimeout: lblis.BackendConnectTimeout,

		ListenerHealthCheckOptions: cloudprovider.ListenerHealthCheckOptions{
			HealthCheckReq: lblis.HealthCheckReq,
			HealthCheckExp: lblis.HealthCheckExp,

			HealthCheck:         lblis.HealthCheck,
			HealthCheckType:     lblis.HealthCheckType,
			HealthCheckTimeout:  lblis.HealthCheckTimeout,
			HealthCheckDomain:   lblis.HealthCheckDomain,
			HealthCheckHttpCode: lblis.HealthCheckHttpCode,
			HealthCheckURI:      lblis.HealthCheckURI,
			HealthCheckInterval: lblis.HealthCheckInterval,

			HealthCheckRise: lblis.HealthCheckRise,
			HealthCheckFail: lblis.HealthCheckFall,
		},

		ListenerStickySessionOptions: cloudprovider.ListenerStickySessionOptions{
			StickySession:              lblis.StickySession,
			StickySessionType:          lblis.StickySessionType,
			StickySessionCookie:        lblis.StickySessionCookie,
			StickySessionCookieTimeout: lblis.StickySessionCookieTimeout,
		},

		BackendServerPort: lblis.BackendServerPort,
		XForwardedFor:     lblis.XForwardedFor,
		TLSCipherPolicy:   lblis.TLSCipherPolicy,
		Gzip:              lblis.Gzip,
	}

	if backendgroup, _ := lblis.GetLoadbalancerBackendGroup(); backendgroup != nil {
		listener.BackendGroupId = backendgroup.ExternalId
		listener.BackendGroupType = backendgroup.Type
	}

	return listener, nil
}

func (lblis *SLoadbalancerListener) GetLoadbalancerListenerRules() ([]SLoadbalancerListenerRule, error) {
	q := LoadbalancerListenerRuleManager.Query().Equals("listener_id", lblis.Id)
	rules := []SLoadbalancerListenerRule{}
	err := db.FetchModelObjects(LoadbalancerListenerRuleManager, q, &rules)
	if err != nil {
		return nil, err
	}
	return rules, nil
}

func (lblis *SLoadbalancerListener) GetDefaultRule() (*SLoadbalancerListenerRule, error) {
	q := LoadbalancerListenerRuleManager.Query().Equals("listener_id", lblis.Id).IsTrue("is_default")
	rules := []SLoadbalancerListenerRule{}
	err := db.FetchModelObjects(LoadbalancerListenerRuleManager, q, &rules)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, errors.Wrap(err, "LoadbalancerListener.GetLoadbalancerDefaultRule")
	}

	if len(rules) >= 1 {
		return &rules[0], nil
	}

	return nil, nil
}

func (lblis *SLoadbalancerListener) GetLoadbalancerAcl() *SLoadbalancerAcl {
	if len(lblis.AclId) == 0 {
		return nil
	}

	acl, err := LoadbalancerAclManager.FetchById(lblis.AclId)
	if err != nil {
		return nil
	}
	return acl.(*SLoadbalancerAcl)
}

func (lblis *SLoadbalancerListener) GetLoadbalancerBackendGroup() (*SLoadbalancerBackendGroup, error) {
	groupObj, err := LoadbalancerBackendGroupManager.FetchById(lblis.BackendGroupId)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchById(%s)", lblis.BackendGroupId)
	}
	return groupObj.(*SLoadbalancerBackendGroup), nil
}

func (lblis *SLoadbalancerListener) GetLoadbalancer() (*SLoadbalancer, error) {
	lbObj, err := LoadbalancerManager.FetchById(lblis.LoadbalancerId)
	if err != nil {
		return nil, err
	}
	return lbObj.(*SLoadbalancer), nil
}

func (lblis *SLoadbalancerListener) GetRegion() (*SCloudregion, error) {
	loadbalancer, err := lblis.GetLoadbalancer()
	if err != nil {
		return nil, err
	}
	return loadbalancer.GetRegion()
}

func (lblis *SLoadbalancerListener) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	loadbalancer, err := lblis.GetLoadbalancer()
	if err != nil {
		return nil, err
	}
	return loadbalancer.GetIRegion(ctx)
}

func (lblis *SLoadbalancerListener) GetILoadbalancer(ctx context.Context) (cloudprovider.ICloudLoadbalancer, error) {
	lb, err := lblis.GetLoadbalancer()
	if err != nil {
		return nil, errors.Wrapf(err, "GetLoadbalancer")
	}
	return lb.GetILoadbalancer(ctx)
}

func (lblis *SLoadbalancerListener) GetILoadbalancerListener(ctx context.Context) (cloudprovider.ICloudLoadbalancerListener, error) {
	if len(lblis.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	iLb, err := lblis.GetILoadbalancer(ctx)
	if err != nil {
		return nil, err
	}
	return iLb.GetILoadBalancerListenerById(lblis.ExternalId)
}

func (man *SLoadbalancerListenerManager) getLoadbalancerListenersByLoadbalancer(lb *SLoadbalancer) ([]SLoadbalancerListener, error) {
	listeners := []SLoadbalancerListener{}
	q := man.Query().Equals("loadbalancer_id", lb.Id)
	if err := db.FetchModelObjects(man, q, &listeners); err != nil {
		return nil, err
	}
	return listeners, nil
}

func (man *SLoadbalancerListenerManager) SyncLoadbalancerListeners(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, lb *SLoadbalancer, listeners []cloudprovider.ICloudLoadbalancerListener) ([]SLoadbalancerListener, []cloudprovider.ICloudLoadbalancerListener, compare.SyncResult) {
	syncOwnerId := provider.GetOwnerId()

	lockman.LockRawObject(ctx, "listeners", lb.Id)
	defer lockman.ReleaseRawObject(ctx, "listeners", lb.Id)

	localListeners := []SLoadbalancerListener{}
	remoteListeners := []cloudprovider.ICloudLoadbalancerListener{}
	result := compare.SyncResult{}

	dbListeners, err := man.getLoadbalancerListenersByLoadbalancer(lb)
	if err != nil {
		result.Error(err)
		return nil, nil, result
	}

	removed := []SLoadbalancerListener{}
	commondb := []SLoadbalancerListener{}
	commonext := []cloudprovider.ICloudLoadbalancerListener{}
	added := []cloudprovider.ICloudLoadbalancerListener{}

	err = compare.CompareSets(dbListeners, listeners, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return nil, nil, result
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].syncRemoveCloudLoadbalancerListener(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}
	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudLoadbalancerListener(ctx, userCred, lb, commonext[i], syncOwnerId, provider)
		if err != nil {
			result.UpdateError(err)
			continue
		}
		localListeners = append(localListeners, commondb[i])
		remoteListeners = append(remoteListeners, commonext[i])
		result.Update()
	}
	for i := 0; i < len(added); i++ {
		new, err := man.newFromCloudLoadbalancerListener(ctx, userCred, lb, added[i], syncOwnerId, provider)
		if err != nil {
			result.AddError(err)
			continue
		}
		localListeners = append(localListeners, *new)
		remoteListeners = append(remoteListeners, added[i])
		result.Add()
	}
	return localListeners, remoteListeners, result
}

func (lblis *SLoadbalancerListener) constructFieldsFromCloudListener(userCred mcclient.TokenCredential, lb *SLoadbalancer, extListener cloudprovider.ICloudLoadbalancerListener) {
	// lblis.ManagerId = lb.ManagerId
	// lblis.CloudregionId = lb.CloudregionId
	// lblis.Name = extListener.GetName()
	lblis.ListenerType = extListener.GetListenerType()
	lblis.EgressMbps = extListener.GetEgressMbps()
	lblis.ListenerPort = extListener.GetListenerPort()
	lblis.Status = extListener.GetStatus()

	// default off
	if extListener.GetAclStatus() == "" {
		lblis.AclStatus = api.LB_BOOL_OFF
	} else {
		lblis.AclStatus = extListener.GetAclStatus()
	}

	lblis.AclType = extListener.GetAclType()
	if aclId := extListener.GetAclId(); len(aclId) > 0 {
		if _acl, err := db.FetchByExternalIdAndManagerId(LoadbalancerAclManager, aclId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			return q.Equals("manager_id", lb.ManagerId)
		}); err == nil {
			acl := _acl.(*SLoadbalancerAcl)
			lblis.AclId = acl.Id
		}
	} else {
		lblis.AclId = ""
	}

	if scheduler := extListener.GetScheduler(); len(scheduler) > 0 {
		lblis.Scheduler = scheduler
	}

	if len(extListener.GetHealthCheckType()) > 0 {
		lblis.HealthCheck = extListener.GetHealthCheck()
		if lblis.HealthCheck == api.LB_BOOL_ON {
			lblis.HealthCheckType = extListener.GetHealthCheckType()
			lblis.HealthCheckTimeout = extListener.GetHealthCheckTimeout()
			lblis.HealthCheckInterval = extListener.GetHealthCheckInterval()
			lblis.HealthCheckRise = extListener.GetHealthCheckRise()
			lblis.HealthCheckFall = extListener.GetHealthCheckFail()
			lblis.HealthCheckDomain = extListener.GetHealthCheckDomain()
			lblis.HealthCheckURI = extListener.GetHealthCheckURI()
			lblis.HealthCheckExp = extListener.GetHealthCheckExp()
			lblis.HealthCheckHttpCode = extListener.GetHealthCheckCode()
		}
	}

	if utils.IsInStringArray(extListener.GetRedirect(), []string{api.LB_REDIRECT_OFF, api.LB_REDIRECT_RAW}) {
		lblis.Redirect = extListener.GetRedirect()
		lblis.RedirectCode = int(extListener.GetRedirectCode())
		lblis.RedirectScheme = extListener.GetRedirectScheme()
		lblis.RedirectHost = extListener.GetRedirectHost()
		lblis.RedirectPath = extListener.GetRedirectPath()
	}

	lblis.ClientIdleTimeout = extListener.GetClientIdleTimeout()
	lblis.BackendConnectTimeout = extListener.GetBackendConnectTimeout()
	lblis.BackendServerPort = extListener.GetBackendServerPort()

	switch lblis.ListenerType {
	case api.LB_LISTENER_TYPE_UDP:
		if !utils.IsInStringArray(lblis.GetProviderName(), []string{api.CLOUD_PROVIDER_HUAWEI, api.CLOUD_PROVIDER_HCSO, api.CLOUD_PROVIDER_HCS}) {
			lblis.HealthCheckExp = extListener.GetHealthCheckExp()
			lblis.HealthCheckReq = extListener.GetHealthCheckReq()
		}
	case api.LB_LISTENER_TYPE_HTTPS:
		lblis.TLSCipherPolicy = extListener.GetTLSCipherPolicy()
		lblis.EnableHttp2 = extListener.HTTP2Enabled()
		if certificateId := extListener.GetCertificateId(); len(certificateId) > 0 {
			cert, err := db.FetchByExternalIdAndManagerId(LoadbalancerCertificateManager, certificateId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				return q.Equals("manager_id", lb.ManagerId)
			})
			if err != nil {
				log.Errorf("fetch cert %s error: %v", certificateId, err)
			} else {
				lblis.CertificateId = cert.GetId()
			}
		}
	case api.LB_LISTENER_TYPE_HTTP:
		if len(extListener.GetStickySessionType()) > 0 {
			if lblis.GetProviderName() == api.CLOUD_PROVIDER_QCLOUD && utils.IsInStringArray(lblis.ListenerType, []string{api.LB_LISTENER_TYPE_HTTP, api.LB_LISTENER_TYPE_HTTPS}) {
				// 腾讯云http&https监听， 没有会话保持，不需要同步
			} else {
				// deprecated ???
				lblis.StickySession = extListener.GetStickySession()
				lblis.StickySessionType = extListener.GetStickySessionType()
				lblis.StickySessionCookie = extListener.GetStickySessionCookie()
				lblis.StickySessionCookieTimeout = extListener.GetStickySessionCookieTimeout()
			}
		}
		lblis.EnableHttp2 = extListener.HTTP2Enabled()
		lblis.XForwardedFor = extListener.XForwardedForEnabled()
		lblis.Gzip = extListener.GzipEnabled()
	}

	if utils.IsInStringArray(extListener.GetStickySession(), []string{api.LB_BOOL_ON, api.LB_BOOL_OFF}) {
		lblis.StickySession = extListener.GetStickySession()
		lblis.StickySessionType = extListener.GetStickySessionType()
		lblis.StickySessionCookie = extListener.GetStickySessionCookie()
		lblis.StickySessionCookieTimeout = extListener.GetStickySessionCookieTimeout()
	}

}

func (lblis *SLoadbalancerListener) updateBackendGroupId(ctx context.Context, ext cloudprovider.ICloudLoadbalancerListener, managerId string) error {
	extId := ext.GetBackendGroupId()
	if len(extId) == 0 {
		return nil
	}
	q := LoadbalancerBackendGroupManager.Query().Equals("external_id", extId).Equals("loadbalancer_id", lblis.LoadbalancerId)
	groups := []SLoadbalancerBackendGroup{}
	err := db.FetchModelObjects(LoadbalancerBackendGroupManager, q, &groups)
	if err != nil {
		return errors.Wrapf(err, "db.FetchModelObjects")
	}
	if len(groups) == 1 {
		_, err := db.Update(lblis, func() error {
			lblis.BackendGroupId = groups[0].Id
			return nil
		})
		return err
	}
	return nil
}

func (lblis *SLoadbalancerListener) syncRemoveCloudLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lblis)
	defer lockman.ReleaseObject(ctx, lblis)

	err := lblis.ValidateDeleteCondition(ctx, nil)
	if err != nil { // cannot delete
		return lblis.SetStatus(ctx, userCred, api.LB_STATUS_UNKNOWN, "sync to delete")
	}
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    lblis,
		Action: notifyclient.ActionSyncDelete,
	})
	return lblis.RealDelete(ctx, userCred)
}

func (lblis *SLoadbalancerListener) SyncWithCloudLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, extListener cloudprovider.ICloudLoadbalancerListener, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider) error {
	diff, err := db.UpdateWithLock(ctx, lblis, func() error {
		lblis.constructFieldsFromCloudListener(userCred, lb, extListener)
		return nil
	})
	if err != nil {
		return err
	}
	if account, _ := provider.GetCloudaccount(); account != nil {
		syncMetadata(ctx, userCred, lblis, extListener, account.ReadOnly)
	}

	if len(diff) > 0 {
		notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
			Obj:    lblis,
			Action: notifyclient.ActionSyncUpdate,
		})
	}

	err = lblis.updateBackendGroupId(ctx, extListener, lb.ManagerId)
	if err != nil {
		return errors.Wrap(err, "updateBackendGroupId")
	}

	db.OpsLog.LogSyncUpdate(lblis, diff, userCred)

	return nil
}

func (man *SLoadbalancerListenerManager) newFromCloudLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, extListener cloudprovider.ICloudLoadbalancerListener, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider) (*SLoadbalancerListener, error) {
	lblis := &SLoadbalancerListener{}
	lblis.SetModelManager(man, lblis)

	lblis.LoadbalancerId = lb.Id
	lblis.ExternalId = extListener.GetGlobalId()

	lblis.constructFieldsFromCloudListener(userCred, lb, extListener)

	var err = func() error {
		lockman.LockRawObject(ctx, man.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, man.Keyword(), "name")

		newName, err := db.GenerateName(ctx, man, syncOwnerId, extListener.GetName())
		if err != nil {
			return err
		}
		lblis.Name = newName

		return man.TableSpec().Insert(ctx, lblis)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}
	syncMetadata(ctx, userCred, lblis, extListener, false)

	err = lblis.updateBackendGroupId(ctx, extListener, lb.ManagerId)
	if err != nil {
		return nil, errors.Wrap(err, "updateBackendGroupId")
	}

	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    lblis,
		Action: notifyclient.ActionSyncCreate,
	})

	db.OpsLog.LogEvent(lblis, db.ACT_CREATE, lblis.GetShortDesc(ctx), userCred)

	return lblis, nil
}

func (manager *SLoadbalancerListenerManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SLoadbalancerResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SLoadbalancerResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, ".SLoadbalancerResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}

func (manager *SLoadbalancerListenerManager) InitializeData() error {
	return nil
}
