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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SLoadbalancerListenerManager struct {
	SLoadbalancerLogSkipper
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager

	SLoadbalancerResourceBaseManager
	SLoadbalancerAclResourceBaseManager
	SLoadbalancerCertificateResourceBaseManager
}

var LoadbalancerListenerManager *SLoadbalancerListenerManager

func init() {
	LoadbalancerListenerManager = &SLoadbalancerListenerManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
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
//  - CACertificate string
//  - Certificate2Id // multiple certificates for rsa, ecdsa
//  - Use certificate for tcp listener
//  - Customize ciphers?
type SLoadbalancerHTTPSListener struct {
	SLoadbalancerCertificateResourceBase

	CachedCertificateId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	TLSCipherPolicy     string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	EnableHttp2         bool   `create:"optional" list:"user" update:"user"`
}

type SLoadbalancerListener struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	SLoadbalancerResourceBase `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	//LoadbalancerId    string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	ListenerType      string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"`
	ListenerPort      int    `nullable:"false" list:"user" create:"required"`
	BackendGroupId    string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
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
	CachedAclId                  string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`

	SLoadbalancerRateLimiter

	SLoadbalancerTCPListener
	SLoadbalancerUDPListener
	SLoadbalancerHTTPListener
	SLoadbalancerHTTPSListener

	SLoadbalancerHealthCheck
	SLoadbalancerHTTPRateLimiter
	SLoadbalancerHTTPRedirect
}

func (man *SLoadbalancerListenerManager) CheckListenerUniqueness(ctx context.Context, lb *SLoadbalancer, listenerType string, listenerPort int64) error {
	q := man.Query().
		IsFalse("pending_deleted").
		Equals("loadbalancer_id", lb.Id).
		Equals("listener_port", listenerPort)
	switch listenerType {
	case api.LB_LISTENER_TYPE_TCP, api.LB_LISTENER_TYPE_HTTP, api.LB_LISTENER_TYPE_HTTPS:
		q = q.NotEquals("listener_type", api.LB_LISTENER_TYPE_UDP)
	case api.LB_LISTENER_TYPE_UDP:
		q = q.Equals("listener_type", api.LB_LISTENER_TYPE_UDP)
	default:
		return fmt.Errorf("unexpected listener type: %s", listenerType)
	}
	var listener SLoadbalancerListener
	q.First(&listener)
	if len(listener.Id) > 0 {
		return httperrors.NewConflictError("%s listener port %d is already taken by listener %s(%s)",
			listenerType, listenerPort, listener.Name, listener.Id)
	}
	return nil
}

func (man *SLoadbalancerListenerManager) CheckAwsListenerUniqueness(ctx context.Context, lb *SLoadbalancer, lblis *SLoadbalancerListener, listenerType string, listenerPort int64) error {
	q := man.Query().
		IsFalse("pending_deleted").
		Equals("loadbalancer_id", lb.Id).
		Equals("listener_port", listenerPort)

	if lblis != nil {
		q = q.NotEquals("id", lblis.GetId())
	}
	var listener SLoadbalancerListener
	q.First(&listener)
	if len(listener.Id) > 0 {
		return httperrors.NewConflictError("%s listener port %d is already taken by listener %s(%s)",
			listenerType, listenerPort, listener.Name, listener.Id)
	}
	return nil
}

func (man *SLoadbalancerListenerManager) pendingDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential, q *sqlchemy.SQuery) {
	subs := []SLoadbalancerListener{}
	db.FetchModelObjects(man, q, &subs)
	for _, sub := range subs {
		sub.LBPendingDelete(ctx, userCred)
	}
}

func (man *SLoadbalancerListenerManager) FilterByOwner(q *sqlchemy.SQuery, userCred mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if userCred != nil {
		sq := LoadbalancerManager.Query("id")
		switch scope {
		case rbacutils.ScopeProject:
			sq = sq.Equals("tenant_id", userCred.GetProjectId())
			return q.In("loadbalancer_id", sq.SubQuery())
		case rbacutils.ScopeDomain:
			sq = sq.Equals("domain_id", userCred.GetProjectDomainId())
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
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = man.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = man.SLoadbalancerResourceBaseManager.ListItemFilter(ctx, q, userCred, query.LoadbalancerFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLoadbalancerResourceBaseManager.ListItemFilter")
	}

	// userProjId := userCred.GetProjectId()
	data := jsonutils.Marshal(query).(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		// {Key: "loadbalancer", ModelKeyword: "loadbalancer", OwnerId: userCred},
		{Key: "backend_group", ModelKeyword: "loadbalancerbackendgroup", OwnerId: userCred},
		// {Key: "acl", ModelKeyword: "cachedloadbalanceracl", OwnerId: userCred},
		// {Key: "cloudregion", ModelKeyword: "cloudregion", OwnerId: userCred},
		// {Key: "manager", ModelKeyword: "cloudprovider", OwnerId: userCred},
	})
	if err != nil {
		return nil, err
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

	q, err = man.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SLoadbalancerResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.LoadbalancerFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLoadbalancerResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (man *SLoadbalancerListenerManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SLoadbalancerResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (man *SLoadbalancerListenerManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	lbV := validators.NewModelIdOrNameValidator("loadbalancer", "loadbalancer", nil)
	if err := lbV.Validate(data.(*jsonutils.JSONDict)); err == nil {
		return lbV.Model.GetOwnerId(), nil
	}
	return man.SVirtualResourceBaseManager.FetchOwnerId(ctx, data)
}

func (man *SLoadbalancerListenerManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	lbV := validators.NewModelIdOrNameValidator("loadbalancer", "loadbalancer", ownerId)
	if err := lbV.Validate(data); err != nil {
		return nil, err
	}

	backendGroupV := validators.NewModelIdOrNameValidator("backend_group", "loadbalancerbackendgroup", ownerId)
	if err := backendGroupV.Optional(true).Validate(data); err != nil {
		return nil, err
	}

	input := apis.VirtualResourceCreateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal VirtualResourceCreateInput fail %s", err)
	}
	input, err = man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))

	lb := lbV.Model.(*SLoadbalancer)
	region, err := lb.GetRegion()
	if err != nil {
		return nil, err
	}

	return region.GetDriver().ValidateCreateLoadbalancerListenerData(ctx, userCred, ownerId, data, lb, backendGroupV.Model)
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

func (man *SLoadbalancerListenerManager) ValidateAcl(aclStatusV *validators.ValidatorStringChoices, aclTypeV *validators.ValidatorStringChoices, aclV *validators.ValidatorModelIdOrName, data *jsonutils.JSONDict, providerName string) error {
	if aclStatusV.Value == api.LB_BOOL_ON {
		if aclV.Model == nil {
			return httperrors.NewMissingParameterError("acl")
		}
		if len(aclTypeV.Value) == 0 {
			return httperrors.NewMissingParameterError("acl_type")
		}
	} else {
		if !utils.IsInStringArray(providerName, []string{api.CLOUD_PROVIDER_HUAWEI, api.CLOUD_PROVIDER_HCSO, api.CLOUD_PROVIDER_HCS}) {
			data.Set("acl_id", jsonutils.NewString(""))
			data.Set("cached_acl_id", jsonutils.NewString(""))
		}
	}
	return nil
}

func (lblis *SLoadbalancerListener) PerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformStatusInput) (jsonutils.JSONObject, error) {
	if _, err := lblis.SVirtualResourceBase.PerformStatus(ctx, userCred, query, input); err != nil {
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
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lblis *SLoadbalancerListener) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	ownerId := lblis.GetOwnerId()
	backendGroupV := validators.NewModelIdOrNameValidator("backend_group", "loadbalancerbackendgroup", ownerId)
	backendGroupV.AllowEmpty(true).Default(lblis.BackendGroupId)
	if err := backendGroupV.Validate(data); err != nil {
		return nil, err
	}

	input := apis.VirtualResourceBaseUpdateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	input, err = lblis.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBase.ValidateUpdateData")
	}
	data.Update(jsonutils.Marshal(input))

	region, err := lblis.GetRegion()
	if err != nil {
		return nil, err
	}

	return region.GetDriver().ValidateUpdateLoadbalancerListenerData(ctx, userCred, data, lblis, backendGroupV.Model)
}

func (lblis *SLoadbalancerListener) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lblis.SVirtualResourceBase.PostUpdate(ctx, userCred, query, data)

	if account := lblis.GetCloudaccount(); account != nil && !account.IsOnPremise {
		lblis.StartLoadBalancerListenerSyncTask(ctx, userCred, data, "")
	}
}

func (lblis *SLoadbalancerListener) StartLoadBalancerListenerSyncTask(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject, parentTaskId string) error {
	params := jsonutils.NewDict()
	if utils.IsInStringArray(lblis.Status, []string{api.LB_STATUS_ENABLED, api.LB_STATUS_DISABLED}) {
		params.Add(jsonutils.NewString(lblis.Status), "origin_status")
	}

	if data != nil {
		if certId, err := data.GetString("certificate_id"); err == nil && len(certId) > 0 {
			params.Add(jsonutils.NewString(certId), "certificate_id")
		}

		if aclId, err := data.GetString("acl_id"); err == nil && len(aclId) > 0 {
			params.Add(jsonutils.NewString(aclId), "acl_id")
		}
	}

	lblis.SetStatus(userCred, api.LB_SYNC_CONF, "")
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerListenerSyncTask", lblis, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lblis *SLoadbalancerListener) getMoreDetails(out api.LoadbalancerListenerDetails) (api.LoadbalancerListenerDetails, error) {
	{
		if lblis.BackendGroupId != "" {
			lbbg, err := LoadbalancerBackendGroupManager.FetchById(lblis.BackendGroupId)
			if err != nil {
				log.Errorf("loadbalancer listener %s(%s): fetch backend group (%s) error: %s",
					lblis.Name, lblis.Id, lblis.BackendGroupId, err)
				return out, err
			}
			out.BackendGroup = lbbg.GetName()
		}
	}

	if len(lblis.CertificateId) > 0 {
		if cert, _ := lblis.GetLoadbalancerCertificate(); cert != nil {
			out.CertificateName = cert.Name
			out.OriginCertificateId = cert.CertificateId
		}
	}

	return out, nil
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

	virtRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	lbRows := manager.SLoadbalancerResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	lbaclRows := manager.SLoadbalancerAclResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	lbcertRows := manager.SLoadbalancerCertificateResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.LoadbalancerListenerDetails{
			VirtualResourceDetails:              virtRows[i],
			LoadbalancerResourceInfo:            lbRows[i],
			LoadbalancerAclResourceInfo:         lbaclRows[i],
			LoadbalancerCertificateResourceInfo: lbcertRows[i],
		}
		rows[i], _ = objs[i].(*SLoadbalancerListener).getMoreDetails(rows[i])
	}

	return rows
}

func (lblis *SLoadbalancerListener) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lblis.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	lblis.SetStatus(userCred, api.LB_CREATING, "")
	if err := lblis.StartLoadBalancerListenerCreateTask(ctx, userCred, data.(*jsonutils.JSONDict), ""); err != nil {
		log.Errorf("Failed to create loadbalancer listener error: %v", err)
	}
}

func (lblis *SLoadbalancerListener) StartLoadBalancerListenerCreateTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerListenerCreateTask", lblis, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lblis *SLoadbalancerListener) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	parasm := jsonutils.NewDict()
	parasm.Add(jsonutils.JSONTrue, "purge")
	return nil, lblis.StartLoadBalancerListenerDeleteTask(ctx, userCred, parasm, "")
}

func (lblis *SLoadbalancerListener) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, lblis.StartLoadBalancerListenerSyncTask(ctx, userCred, nil, "")
}

func (lblis *SLoadbalancerListener) StartLoadBalancerListenerDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerListenerDeleteTask", lblis, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lblis *SLoadbalancerListener) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	lblis.SetStatus(userCred, api.LB_STATUS_DELETING, "")
	return lblis.StartLoadBalancerListenerDeleteTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (lblis *SLoadbalancerListener) LBPendingDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	lblis.pendingDeleteSubs(ctx, userCred)
	lblis.DoPendingDelete(ctx, userCred)
}

func (lblis *SLoadbalancerListener) pendingDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential) {
	subMan := LoadbalancerListenerRuleManager
	ownerId := lblis.GetOwnerId()

	lockman.LockClass(ctx, subMan, db.GetLockClassKey(subMan, ownerId))
	defer lockman.ReleaseClass(ctx, subMan, db.GetLockClassKey(subMan, ownerId))
	q := subMan.Query().IsFalse("pending_deleted").Equals("listener_id", lblis.Id)
	subMan.pendingDeleteSubs(ctx, userCred, q)
}

func (lblis *SLoadbalancerListener) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (lblis *SLoadbalancerListener) GetLoadbalancerListenerParams() (*cloudprovider.SLoadbalancerListener, error) {
	listener := &cloudprovider.SLoadbalancerListener{
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

		StickySession:              lblis.StickySession,
		StickySessionType:          lblis.StickySessionType,
		StickySessionCookie:        lblis.StickySessionCookie,
		StickySessionCookieTimeout: lblis.StickySessionCookieTimeout,

		BackendServerPort: lblis.BackendServerPort,
		XForwardedFor:     lblis.XForwardedFor,
		TLSCipherPolicy:   lblis.TLSCipherPolicy,
		Gzip:              lblis.Gzip,
	}
	if acl := lblis.GetCachedLoadbalancerAcl(); acl != nil {
		listener.AccessControlListID = acl.ExternalId
		listener.AccessControlListType = lblis.AclType
	}
	if certificate, err := lblis.GetLoadbalancerCertificate(); err != nil {
		return nil, errors.Wrap(err, "SLoadbalancerListener.GetLoadbalancerListenerParams.certificate")
	} else if certificate != nil && lblis.ListenerType == api.LB_LISTENER_TYPE_HTTPS {
		listener.CertificateID = certificate.ExternalId
	}

	if backendgroup := lblis.GetLoadbalancerBackendGroup(); backendgroup != nil {
		listener.BackendGroupID = backendgroup.ExternalId
		listener.BackendGroupType = backendgroup.Type
	}

	if loadbalancer, _ := lblis.GetLoadbalancer(); loadbalancer != nil {
		listener.LoadbalancerID = loadbalancer.ExternalId
	}

	return listener, nil
}

func (lblis *SLoadbalancerListener) GetLoadbalancerListenerRules() ([]SLoadbalancerListenerRule, error) {
	q := LoadbalancerListenerRuleManager.Query().Equals("listener_id", lblis.Id).IsFalse("pending_deleted")
	rules := []SLoadbalancerListenerRule{}
	err := db.FetchModelObjects(LoadbalancerListenerRuleManager, q, &rules)
	if err != nil {
		return nil, err
	}
	return rules, nil
}

func (lblis *SLoadbalancerListener) GetDefaultRule() (*SLoadbalancerListenerRule, error) {
	q := LoadbalancerListenerRuleManager.Query().Equals("listener_id", lblis.Id).IsFalse("pending_deleted").IsTrue("is_default")
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

func (lblis *SLoadbalancerListener) GetHuaweiLoadbalancerListenerParams() (*cloudprovider.SLoadbalancerListener, error) {
	listener, err := lblis.GetLoadbalancerListenerParams()
	if err != nil {
		return nil, err
	}

	if backendgroup := lblis.GetLoadbalancerBackendGroup(); backendgroup != nil {
		cachedLbbg, err := HuaweiCachedLbbgManager.GetCachedBackendGroupByAssociateId(lblis.GetId())
		if err != nil {
			if err != sql.ErrNoRows {
				return nil, errors.Wrap(err, "loadbalancerListener.GetCachedBackendGroupByAssociateId")
			} else {
				log.Debugf("loadbalancerListener.GetCachedBackendGroupByAssociateId %s not found", lblis.GetId())
			}
		} else {
			listener.BackendGroupID = cachedLbbg.ExternalId
			listener.BackendGroupType = backendgroup.Type
		}
	}

	return listener, nil
}

func (lblis *SLoadbalancerListener) GetAwsLoadbalancerListenerParams() (*cloudprovider.SLoadbalancerListener, error) {
	listener, err := lblis.GetLoadbalancerListenerParams()
	if err != nil {
		return nil, err
	}

	lb, _ := lblis.GetLoadbalancer()
	if lb != nil {
		listener.LoadbalancerID = lb.ExternalId
	}

	if backendgroup := lblis.GetLoadbalancerBackendGroup(); backendgroup != nil {
		cachedLbbg, err := AwsCachedLbbgManager.GetUsableCachedBackendGroup(lb.GetId(), lblis.BackendGroupId, listener.ListenerType, listener.HealthCheckType, listener.HealthCheckInterval)
		if err != nil {
			return nil, err
		}

		if cachedLbbg == nil {
			return nil, fmt.Errorf("backendgroup %s related cached loadbalancer backendgroup not found", backendgroup.GetId())
		}

		listener.BackendGroupID = cachedLbbg.ExternalId
		listener.BackendGroupType = backendgroup.Type
	}

	return listener, nil
}

func (lblis *SLoadbalancerListener) GetQcloudLoadbalancerListenerParams() (*cloudprovider.SLoadbalancerListener, error) {
	listener, err := lblis.GetLoadbalancerListenerParams()
	if err != nil {
		return nil, err
	}

	if backendgroup := lblis.GetLoadbalancerBackendGroup(); backendgroup != nil {
		cachedLbbg, err := QcloudCachedLbbgManager.GetCachedBackendGroupByAssociateId(lblis.GetId())
		if err != nil {
			if err != sql.ErrNoRows {
				return nil, errors.Wrap(err, "loadbalancerListener.GetCachedBackendGroupByAssociateId")
			} else {
				log.Debugf("loadbalancerListener.GetCachedBackendGroupByAssociateId %s not found", lblis.GetId())
			}
		} else {
			listener.BackendGroupID = cachedLbbg.ExternalId
			listener.BackendGroupType = backendgroup.Type
		}
	}

	return listener, nil
}

func (lblis *SLoadbalancerListener) GetOpenstackLoadbalancerListenerParams() (*cloudprovider.SLoadbalancerListener, error) {
	listener, err := lblis.GetLoadbalancerListenerParams()
	if err != nil {
		return nil, err
	}

	if backendgroup := lblis.GetLoadbalancerBackendGroup(); backendgroup != nil {
		cachedLbbg, err := OpenstackCachedLbbgManager.GetCachedBackendGroupByAssociateId(lblis.GetId())
		if err != nil {
			if errors.Cause(err) != sql.ErrNoRows {
				return nil, errors.Wrap(err, "loadbalancerListener.GetCachedBackendGroupByAssociateId")
			} else {
				log.Debugf("loadbalancerListener.GetCachedBackendGroupByAssociateId %s not found", lblis.GetId())
			}
		} else {
			listener.BackendGroupID = cachedLbbg.ExternalId
			listener.BackendGroupType = backendgroup.Type
		}
	}

	return listener, nil
}

func (lblis *SLoadbalancerListener) GetLoadbalancerCertificate() (*SCachedLoadbalancerCertificate, error) {
	if len(lblis.CachedCertificateId) == 0 {
		return nil, nil
	}

	ret := &SCachedLoadbalancerCertificate{}
	err := CachedLoadbalancerCertificateManager.Query().Equals("id", lblis.CachedCertificateId).IsFalse("pending_deleted").First(ret)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return ret, nil
}

func (lblis *SLoadbalancerListener) GetCachedLoadbalancerAcl() *SCachedLoadbalancerAcl {
	if len(lblis.CachedAclId) == 0 {
		return nil
	}

	acl, err := CachedLoadbalancerAclManager.FetchById(lblis.CachedAclId)
	if err != nil {
		return nil
	}
	return acl.(*SCachedLoadbalancerAcl)
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

func (lblis *SLoadbalancerListener) GetLoadbalancerBackendGroup() *SLoadbalancerBackendGroup {
	_group, err := LoadbalancerBackendGroupManager.FetchById(lblis.BackendGroupId)
	if err != nil {
		return nil
	}
	group := _group.(*SLoadbalancerBackendGroup)
	if group.PendingDeleted {
		log.Errorf("backendgroup %s(%s) has been deleted", group.Name, group.Id)
		return nil
	}
	return group
}

func (lblis *SLoadbalancerListener) GetLoadbalancer() (*SLoadbalancer, error) {
	_loadbalancer, err := LoadbalancerManager.FetchById(lblis.LoadbalancerId)
	if err != nil {
		return nil, err
	}
	loadbalancer := _loadbalancer.(*SLoadbalancer)
	if loadbalancer.PendingDeleted {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "pending deleted")
	}
	return loadbalancer, nil
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

func (man *SLoadbalancerListenerManager) getLoadbalancerListenersByLoadbalancer(lb *SLoadbalancer) ([]SLoadbalancerListener, error) {
	listeners := []SLoadbalancerListener{}
	q := man.Query().Equals("loadbalancer_id", lb.Id).IsFalse("pending_deleted")
	if err := db.FetchModelObjects(man, q, &listeners); err != nil {
		return nil, err
	}
	return listeners, nil
}

func (man *SLoadbalancerListenerManager) SyncLoadbalancerListeners(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, lb *SLoadbalancer, listeners []cloudprovider.ICloudLoadbalancerListener, syncRange *SSyncRange) ([]SLoadbalancerListener, []cloudprovider.ICloudLoadbalancerListener, compare.SyncResult) {
	syncOwnerId := provider.GetOwnerId()

	lockman.LockRawObject(ctx, "listeners", lb.Id)
	defer lockman.ReleaseRawObject(ctx, "listeners", lb.Id)

	localListeners := []SLoadbalancerListener{}
	remoteListeners := []cloudprovider.ICloudLoadbalancerListener{}
	syncResult := compare.SyncResult{}

	dbListeners, err := man.getLoadbalancerListenersByLoadbalancer(lb)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	removed := []SLoadbalancerListener{}
	commondb := []SLoadbalancerListener{}
	commonext := []cloudprovider.ICloudLoadbalancerListener{}
	added := []cloudprovider.ICloudLoadbalancerListener{}

	err = compare.CompareSets(dbListeners, listeners, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].syncRemoveCloudLoadbalancerListener(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudLoadbalancerListener(ctx, userCred, lb, commonext[i], syncOwnerId, provider)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			localListeners = append(localListeners, commondb[i])
			remoteListeners = append(remoteListeners, commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		new, err := man.newFromCloudLoadbalancerListener(ctx, userCred, lb, added[i], syncOwnerId, provider)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, new, added[i])
			localListeners = append(localListeners, *new)
			remoteListeners = append(remoteListeners, added[i])
			syncResult.Add()
		}
	}
	return localListeners, remoteListeners, syncResult
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
	if aclID := extListener.GetAclId(); len(aclID) > 0 {
		if _acl, err := db.FetchByExternalIdAndManagerId(CachedLoadbalancerAclManager, aclID, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			return q.Equals("manager_id", lb.ManagerId)
		}); err == nil {
			acl := _acl.(*SCachedLoadbalancerAcl)
			lblis.CachedAclId = acl.GetId()
			lblis.AclId = acl.AclId
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
			if _cert, err := db.FetchByExternalIdAndManagerId(CachedLoadbalancerCertificateManager, certificateId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				return q.Equals("manager_id", lb.ManagerId)
			}); err == nil {
				cert := _cert.(*SCachedLoadbalancerCertificate)
				lblis.CachedCertificateId = cert.GetId()
				lblis.CertificateId = cert.CertificateId
			}
		}
		fallthrough
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

	groupId := extListener.GetBackendGroupId()
	switch lblis.GetProviderName() {
	case api.CLOUD_PROVIDER_HUAWEI, api.CLOUD_PROVIDER_HCSO, api.CLOUD_PROVIDER_HCS:
		if len(groupId) > 0 {
			group, err := db.FetchByExternalIdAndManagerId(HuaweiCachedLbbgManager, groupId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				return q.Equals("manager_id", lb.ManagerId)
			})
			if err != nil {
				if err == sql.ErrNoRows {
					lblis.BackendGroupId = ""
				}
				log.Errorf("Fetch huawei loadbalancer backendgroup by external id %s failed: %s", groupId, err)
			} else {
				lblis.BackendGroupId = group.(*SHuaweiCachedLbbg).BackendGroupId
			}
		}
	case api.CLOUD_PROVIDER_AWS:
		if len(groupId) > 0 {
			group, err := db.FetchByExternalIdAndManagerId(AwsCachedLbbgManager, groupId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				return q.Equals("manager_id", lb.ManagerId)
			})
			if err != nil {
				log.Errorf("Fetch aws loadbalancer backendgroup by external id %s failed: %s", groupId, err)
			} else {
				lblis.BackendGroupId = group.(*SAwsCachedLbbg).BackendGroupId
				if rule, err := lblis.GetDefaultRule(); err != nil || rule == nil {
					log.Warningf("LoadbalancerListener %s default rule not found %s", lblis.GetId(), err)
				} else {
					_, err = db.Update(rule, func() error {
						rule.BackendGroupId = lblis.BackendGroupId
						return nil
					})
					if err != nil {
						log.Errorf("Update default rule %s backendgroup failed %s", rule.GetId(), err)
					}
				}
			}
		}
	case api.CLOUD_PROVIDER_QCLOUD:
		if len(groupId) > 0 {
			lb, _ := lblis.GetLoadbalancer()
			if forward, _ := lb.LBInfo.Int("Forward"); forward == 1 {
				// 应用型负载均衡
				group, err := db.FetchByExternalIdAndManagerId(QcloudCachedLbbgManager, groupId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
					return q.Equals("manager_id", lb.ManagerId)
				})
				if err != nil {
					log.Errorf("Fetch qcloud loadbalancer backendgroup by external id %s failed: %s", groupId, err)
				} else {
					lblis.BackendGroupId = group.(*SQcloudCachedLbbg).BackendGroupId
				}
			} else {
				// 传统型负载均衡
				if group, err := db.FetchByExternalIdAndManagerId(LoadbalancerBackendGroupManager, groupId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
					sq := LoadbalancerManager.Query().SubQuery()
					return q.Join(sq, sqlchemy.Equals(sq.Field("id"), q.Field("loadbalancer_id"))).Filter(sqlchemy.Equals(sq.Field("manager_id"), lb.ManagerId))
				}); err == nil {
					lblis.BackendGroupId = group.GetId()
				}
			}
		}
	case api.CLOUD_PROVIDER_OPENSTACK:
		if len(groupId) > 0 {
			group, err := db.FetchByExternalIdAndManagerId(OpenstackCachedLbbgManager, groupId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				return q.Equals("manager_id", lb.ManagerId)
			})
			if err != nil {
				if errors.Cause(err) == sql.ErrNoRows {
					lblis.BackendGroupId = ""
				}
				log.Errorf("Fetch openstack loadbalancer backendgroup by external id %s failed: %s", groupId, err)
			} else {
				lblis.BackendGroupId = group.(*SOpenstackCachedLbbg).BackendGroupId
			}
		}
	default:
		if len(lblis.BackendGroupId) == 0 && len(groupId) == 0 {
			lblis.BackendGroupId = lb.BackendGroupId
		} else if group, err := db.FetchByExternalIdAndManagerId(LoadbalancerBackendGroupManager, groupId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			sq := LoadbalancerManager.Query().SubQuery()
			q = q.Join(sq, sqlchemy.Equals(sq.Field("id"), q.Field("loadbalancer_id"))).Filter(sqlchemy.Equals(sq.Field("manager_id"), lb.ManagerId))
			return q.IsFalse("pending_deleted")
		}); err == nil {
			lblis.BackendGroupId = group.GetId()
		}
	}
}

func (lblis *SLoadbalancerListener) updateCachedLoadbalancerBackendGroupAssociate(ctx context.Context, extListener cloudprovider.ICloudLoadbalancerListener, managerId string) error {
	exteralLbbgId := extListener.GetBackendGroupId()
	if len(exteralLbbgId) == 0 {
		return nil
	}

	switch lblis.GetProviderName() {
	case api.CLOUD_PROVIDER_HUAWEI, api.CLOUD_PROVIDER_HCSO, api.CLOUD_PROVIDER_HCS:
		_group, err := db.FetchByExternalIdAndManagerId(HuaweiCachedLbbgManager, exteralLbbgId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			return q.Equals("manager_id", managerId)
		})
		if err != nil {
			if err == sql.ErrNoRows {
				lblis.BackendGroupId = ""
			} else {
				return fmt.Errorf("Fetch huawei loadbalancer backendgroup by external id %s failed: %s", exteralLbbgId, err)
			}
		}

		if _group != nil {
			group := _group.(*SHuaweiCachedLbbg)
			if group.AssociatedId != lblis.Id {
				_, err := db.UpdateWithLock(ctx, group, func() error {
					group.AssociatedId = lblis.Id
					group.AssociatedType = api.LB_ASSOCIATE_TYPE_LISTENER
					return nil
				})
				if err != nil {
					return errors.Wrap(err, "LoadbalancerListener.updateCachedLoadbalancerBackendGroupAssociate.huawei")
				}
			}
		}
	case api.CLOUD_PROVIDER_QCLOUD:
		lb, _ := lblis.GetLoadbalancer()
		if forward, _ := lb.LBInfo.Int("Forward"); forward == 1 {
			_group, err := db.FetchByExternalIdAndManagerId(QcloudCachedLbbgManager, exteralLbbgId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				return q.Equals("manager_id", managerId)
			})
			if err != nil {
				if err == sql.ErrNoRows {
					lblis.BackendGroupId = ""
				} else {
					return fmt.Errorf("Fetch qcloud loadbalancer backendgroup by external id %s failed: %s", exteralLbbgId, err)
				}
			}

			if _group != nil {
				group := _group.(*SQcloudCachedLbbg)
				if group.AssociatedId != lblis.Id {
					_, err := db.UpdateWithLock(ctx, group, func() error {
						group.AssociatedId = lblis.Id
						group.AssociatedType = api.LB_ASSOCIATE_TYPE_LISTENER
						return nil
					})
					if err != nil {
						return errors.Wrap(err, "LoadbalancerListener.updateCachedLoadbalancerBackendGroupAssociate.qcloud")
					}
				}
			}
		}
	case api.CLOUD_PROVIDER_OPENSTACK:
		_group, err := db.FetchByExternalIdAndManagerId(OpenstackCachedLbbgManager, exteralLbbgId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			return q.Equals("manager_id", managerId)
		})
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				lblis.BackendGroupId = ""
			} else {
				return fmt.Errorf("Fetch openstack loadbalancer backendgroup by external id %s failed: %s", exteralLbbgId, err)
			}
		}

		if _group != nil {
			group := _group.(*SOpenstackCachedLbbg)
			if group.AssociatedId != lblis.Id {
				_, err := db.UpdateWithLock(ctx, group, func() error {
					group.AssociatedId = lblis.Id
					group.AssociatedType = api.LB_ASSOCIATE_TYPE_LISTENER
					return nil
				})
				if err != nil {
					return errors.Wrap(err, "LoadbalancerListener.updateCachedLoadbalancerBackendGroupAssociate.openstack")
				}
			}
		}
	default:
		return nil
	}

	return nil
}

func (lblis *SLoadbalancerListener) syncRemoveCloudLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lblis)
	defer lockman.ReleaseObject(ctx, lblis)

	err := lblis.ValidateDeleteCondition(ctx, nil)
	if err != nil { // cannot delete
		err = lblis.SetStatus(userCred, api.LB_STATUS_UNKNOWN, "sync to delete")
	} else {
		notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
			Obj:    lblis,
			Action: notifyclient.ActionSyncDelete,
		})
		lblis.LBPendingDelete(ctx, userCred)
	}
	return err
}

func (lblis *SLoadbalancerListener) SyncWithCloudLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, extListener cloudprovider.ICloudLoadbalancerListener, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider) error {
	diff, err := db.UpdateWithLock(ctx, lblis, func() error {
		lblis.constructFieldsFromCloudListener(userCred, lb, extListener)
		return nil
	})
	if err != nil {
		return err
	}

	if len(diff) > 0 {
		notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
			Obj:    lblis,
			Action: notifyclient.ActionSyncUpdate,
		})
	}

	err = lblis.updateCachedLoadbalancerBackendGroupAssociate(ctx, extListener, lb.ManagerId)
	if err != nil {
		return errors.Wrap(err, "LoadbalancerListener.SyncWithCloudLoadbalancerListener")
	}

	db.OpsLog.LogSyncUpdate(lblis, diff, userCred)

	SyncCloudProject(userCred, lblis, syncOwnerId, extListener, provider.Id)

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

	err = lblis.updateCachedLoadbalancerBackendGroupAssociate(ctx, extListener, lb.ManagerId)
	if err != nil {
		return nil, errors.Wrap(err, "LoadbalancerListener.newFromCloudLoadbalancerListener")
	}

	SyncCloudProject(userCred, lblis, syncOwnerId, extListener, provider.Id)

	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    lblis,
		Action: notifyclient.ActionSyncCreate,
	})

	db.OpsLog.LogEvent(lblis, db.ACT_CREATE, lblis.GetShortDesc(ctx), userCred)

	return lblis, nil
}

func (manager *SLoadbalancerListenerManager) InitializeData() error {
	/*listeners := []SLoadbalancerListener{}
	q := manager.Query()
	q = q.Filter(sqlchemy.IsNullOrEmpty(q.Field("cloudregion_id")))
	if err := db.FetchModelObjects(manager, q, &listeners); err != nil {
		return err
	}
	for i := 0; i < len(listeners); i++ {
		listener := &listeners[i]
		if lb := listener.GetLoadbalancer(); lb != nil && len(lb.CloudregionId) > 0 {
			_, err := db.Update(listener, func() error {
				listener.CloudregionId = lb.CloudregionId
				listener.ManagerId = lb.ManagerId
				return nil
			})
			if err != nil {
				log.Errorf("failed to update loadbalancer listener %s cloudregion_id", listener.Name)
			}
		}
	}*/
	return nil
}

func (manager *SLoadbalancerListenerManager) GetResourceCount() ([]db.SScopeResourceCount, error) {
	virts := manager.Query().IsFalse("pending_deleted")
	return db.CalculateResourceCount(virts, "tenant_id")
}

func (manager *SLoadbalancerListenerManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SLoadbalancerResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SLoadbalancerResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, ".SLoadbalancerResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}
