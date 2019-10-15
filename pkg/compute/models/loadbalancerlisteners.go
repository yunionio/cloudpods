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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SLoadbalancerListenerManager struct {
	SLoadbalancerLogSkipper
	db.SVirtualResourceBaseManager
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
	HTTPRequestRate       int `nullable:"true" list:"user" create:"optional" update:"user"`
	HTTPRequestRatePerSrc int `nullable:"true" list:"user" create:"optional" update:"user"`
}

type SLoadbalancerRateLimiter struct {
	EgressMbps int `nullable:"true" list:"user" get:"user" create:"optional" update:"user"`
}

type SLoadbalancerHealthCheck struct {
	HealthCheck     string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	HealthCheckType string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`

	HealthCheckDomain   string `charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	HealthCheckURI      string `charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	HealthCheckHttpCode string `charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`

	HealthCheckRise     int `nullable:"true" list:"user" create:"optional" update:"user"`
	HealthCheckFall     int `nullable:"true" list:"user" create:"optional" update:"user"`
	HealthCheckTimeout  int `nullable:"true" list:"user" create:"optional" update:"user"`
	HealthCheckInterval int `nullable:"true" list:"user" create:"optional" update:"user"`

	HealthCheckReq string `list:"user" create:"optional" update:"user"`
	HealthCheckExp string `list:"user" create:"optional" update:"user"`
}

type SLoadbalancerTCPListener struct{}
type SLoadbalancerUDPListener struct{}

// TODO sensible default for knobs
type SLoadbalancerHTTPListener struct {
	StickySession              string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	StickySessionType          string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	StickySessionCookie        string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	StickySessionCookieTimeout int    `nullable:"true" list:"user" create:"optional" update:"user"`

	XForwardedFor bool `nullable:"true" list:"user" create:"optional" update:"user"`
	Gzip          bool `nullable:"true" list:"user" create:"optional" update:"user"`
}

// TODO
//
//  - CACertificate string
//  - Certificate2Id // multiple certificates for rsa, ecdsa
//  - Use certificate for tcp listener
//  - Customize ciphers?
type SLoadbalancerHTTPSListener struct {
	CertificateId       string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	CachedCertificateId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	TLSCipherPolicy     string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	EnableHttp2         bool   `create:"optional" list:"user" update:"user"`
}

type SLoadbalancerListener struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase

	LoadbalancerId    string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	ListenerType      string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"`
	ListenerPort      int    `nullable:"false" list:"user" create:"required"`
	BackendGroupId    string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	BackendServerPort int    `nullable:"false" get:"user" list:"user" default:"0" create:"optional"`

	Scheduler string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`

	SendProxy string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user" default:"off"`

	ClientRequestTimeout  int `nullable:"true" list:"user" create:"optional" update:"user"`
	ClientIdleTimeout     int `nullable:"true" list:"user" create:"optional" update:"user"`
	BackendConnectTimeout int `nullable:"true" list:"user" create:"optional" update:"user"`
	BackendIdleTimeout    int `nullable:"true" list:"user" create:"optional" update:"user"`

	AclStatus   string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	AclType     string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	AclId       string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	CachedAclId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`

	SLoadbalancerRateLimiter

	SLoadbalancerTCPListener
	SLoadbalancerUDPListener
	SLoadbalancerHTTPListener
	SLoadbalancerHTTPSListener

	SLoadbalancerHealthCheck
	SLoadbalancerHTTPRateLimiter
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

func (man *SLoadbalancerListenerManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	// userProjId := userCred.GetProjectId()
	data := query.(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "loadbalancer", ModelKeyword: "loadbalancer", OwnerId: userCred},
		{Key: "backend_group", ModelKeyword: "loadbalancerbackendgroup", OwnerId: userCred},
		{Key: "acl", ModelKeyword: "cachedloadbalanceracl", OwnerId: userCred},
		{Key: "cloudregion", ModelKeyword: "cloudregion", OwnerId: userCred},
		{Key: "manager", ModelKeyword: "cloudprovider", OwnerId: userCred},
	})
	if err != nil {
		return nil, err
	}
	return q, nil
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

	if _, err := man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data); err != nil {
		return nil, err
	}

	lb := lbV.Model.(*SLoadbalancer)
	region := lb.GetRegion()
	if region == nil {
		return nil, httperrors.NewResourceNotFoundError("failed to find region for loadbalancer %s", lb.Name)
	}

	if len(lb.ManagerId) > 0 {
		data.Set("manager_id", jsonutils.NewString(lb.ManagerId))
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
		if providerName != api.CLOUD_PROVIDER_HUAWEI {
			data.Set("acl_id", jsonutils.NewString(""))
			data.Set("cached_acl_id", jsonutils.NewString(""))
		}
	}
	return nil
}

func (lblis *SLoadbalancerListener) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return lblis.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, lblis, "status")
}

func (lblis *SLoadbalancerListener) PerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if _, err := lblis.SVirtualResourceBase.PerformStatus(ctx, userCred, query, data); err != nil {
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

func (lblis *SLoadbalancerListener) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, lblis, "syncstatus")
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
	backendGroupV.Optional(true)
	if err := backendGroupV.Validate(data); err != nil {
		return nil, err
	}

	if _, err := lblis.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, data); err != nil {
		return nil, err
	}

	region := lblis.GetRegion()
	if region == nil {
		return nil, httperrors.NewResourceNotFoundError("failed to find region for loadbalancer listener %s", lblis.Name)
	}

	return region.GetDriver().ValidateUpdateLoadbalancerListenerData(ctx, userCred, data, lblis, backendGroupV.Model)
}

func (lblis *SLoadbalancerListener) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lblis.SVirtualResourceBase.PostUpdate(ctx, userCred, query, data)
	lblis.StartLoadBalancerListenerSyncTask(ctx, userCred, data, "")
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

func (lblis *SLoadbalancerListener) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := lblis.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	{
		lb, err := LoadbalancerManager.FetchById(lblis.LoadbalancerId)
		if err != nil {
			log.Errorf("loadbalancer listener %s(%s): fetch loadbalancer (%s) error: %s",
				lblis.Name, lblis.Id, lblis.LoadbalancerId, err)
			return extra
		}
		extra.Set("loadbalancer", jsonutils.NewString(lb.GetName()))
	}
	{
		if lblis.BackendGroupId != "" {
			lbbg, err := LoadbalancerBackendGroupManager.FetchById(lblis.BackendGroupId)
			if err != nil {
				log.Errorf("loadbalancer listener %s(%s): fetch backend group (%s) error: %s",
					lblis.Name, lblis.Id, lblis.BackendGroupId, err)
				return extra
			}
			extra.Set("backend_group", jsonutils.NewString(lbbg.GetName()))
		}
	}
	if len(lblis.AclId) > 0 {
		if acl := lblis.GetCachedLoadbalancerAcl(); acl != nil {
			extra.Set("acl_name", jsonutils.NewString(acl.Name))
		}
	}

	if len(lblis.CertificateId) > 0 {
		if cert, _ := lblis.GetLoadbalancerCertificate(); cert != nil {
			extra.Set("certificate_name", jsonutils.NewString(cert.Name))
			extra.Set("origin_certificate_id", jsonutils.NewString(cert.CertificateId))
		}
	}
	regionInfo := lblis.SCloudregionResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if regionInfo != nil {
		extra.Update(regionInfo)
	}
	return extra
}

func (lblis *SLoadbalancerListener) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra := lblis.GetCustomizeColumns(ctx, userCred, query)
	return extra, nil
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

func (lblis *SLoadbalancerListener) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, lblis, "purge")
}

func (lblis *SLoadbalancerListener) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	parasm := jsonutils.NewDict()
	parasm.Add(jsonutils.JSONTrue, "purge")
	return nil, lblis.StartLoadBalancerListenerDeleteTask(ctx, userCred, parasm, "")
}

func (lblis *SLoadbalancerListener) AllowPerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, lblis, "sync")
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

	if loadbalancer := lblis.GetLoadbalancer(); loadbalancer != nil {
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

	lb := lblis.GetLoadbalancer()
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

func (lblis *SLoadbalancerListener) GetLoadbalancerCertificate() (*SCachedLoadbalancerCertificate, error) {
	if len(lblis.CachedCertificateId) == 0 {
		return nil, nil
	}

	ret := &SCachedLoadbalancerCertificate{}
	err := CachedLoadbalancerCertificateManager.Query().Equals("id", lblis.CachedCertificateId).Equals("cloudregion_id", lblis.CloudregionId).IsFalse("pending_deleted").First(ret)
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

func (lblis *SLoadbalancerListener) GetLoadbalancer() *SLoadbalancer {
	_loadbalancer, err := LoadbalancerManager.FetchById(lblis.LoadbalancerId)
	if err != nil {
		log.Errorf("failed to find loadbalancer for loadbalancer listener %s", lblis.Name)
		return nil
	}
	loadbalancer := _loadbalancer.(*SLoadbalancer)
	if loadbalancer.PendingDeleted {
		log.Errorf("loadbalancer %s(%s) has been deleted", loadbalancer.Name, loadbalancer.Id)
		return nil
	}
	return loadbalancer
}

func (lblis *SLoadbalancerListener) GetRegion() *SCloudregion {
	if loadbalancer := lblis.GetLoadbalancer(); loadbalancer != nil {
		return loadbalancer.GetRegion()
	}
	return nil
}

func (lblis *SLoadbalancerListener) GetIRegion() (cloudprovider.ICloudRegion, error) {
	if loadbalancer := lblis.GetLoadbalancer(); loadbalancer != nil {
		return loadbalancer.GetIRegion()
	}
	return nil, fmt.Errorf("failed to find loadbalancer for lblis %s", lblis.Name)
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

	lockman.LockClass(ctx, man, db.GetLockClassKey(man, syncOwnerId))
	defer lockman.ReleaseClass(ctx, man, db.GetLockClassKey(man, syncOwnerId))

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
		err = commondb[i].SyncWithCloudLoadbalancerListener(ctx, userCred, lb, commonext[i], syncOwnerId)
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
		new, err := man.newFromCloudLoadbalancerListener(ctx, userCred, lb, added[i], syncOwnerId)
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
	lblis.ManagerId = lb.ManagerId
	lblis.CloudregionId = lb.CloudregionId
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
		if _acl, err := db.FetchByExternalId(CachedLoadbalancerAclManager, aclID); err == nil {
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
		}
	}

	lblis.BackendServerPort = extListener.GetBackendServerPort()

	switch lblis.ListenerType {
	case api.LB_LISTENER_TYPE_UDP:
		if lblis.GetProviderName() != api.CLOUD_PROVIDER_HUAWEI {
			lblis.HealthCheckExp = extListener.GetHealthCheckExp()
			lblis.HealthCheckReq = extListener.GetHealthCheckReq()
		}
	case api.LB_LISTENER_TYPE_HTTPS:
		lblis.TLSCipherPolicy = extListener.GetTLSCipherPolicy()
		lblis.EnableHttp2 = extListener.HTTP2Enabled()
		if certificateId := extListener.GetCertificateId(); len(certificateId) > 0 {
			if _cert, err := db.FetchByExternalId(CachedLoadbalancerCertificateManager, certificateId); err == nil {
				cert := _cert.(*SCachedLoadbalancerCertificate)
				lblis.CachedCertificateId = cert.GetId()
				lblis.CertificateId = cert.CertificateId
			}
		}
		fallthrough
	case api.LB_LISTENER_TYPE_HTTP:
		if len(extListener.GetStickySessionType()) > 0 {
			lblis.StickySession = extListener.GetStickySession()
			lblis.StickySessionType = extListener.GetStickySessionType()
			lblis.StickySessionCookie = extListener.GetStickySessionCookie()
			lblis.StickySessionCookieTimeout = extListener.GetStickySessionCookieTimeout()
		}
		lblis.XForwardedFor = extListener.XForwardedForEnabled()
		lblis.Gzip = extListener.GzipEnabled()
	}
	groupId := extListener.GetBackendGroupId()
	// 腾讯云兼容代码。主要目的是在关联listen时回写一个fake的backend group external id
	if lblis.GetProviderName() == api.CLOUD_PROVIDER_QCLOUD && len(groupId) > 0 && len(lblis.BackendGroupId) > 0 {
		ilbbg, err := LoadbalancerBackendGroupManager.FetchById(lblis.BackendGroupId)
		lbbg := ilbbg.(*SLoadbalancerBackendGroup)
		if err == nil && (len(lbbg.ExternalId) == 0 || lbbg.ExternalId != groupId) {
			err = db.SetExternalId(lbbg, userCred, groupId)
			if err != nil {
				log.Errorf("Update loadbalancer BackendGroup(%s) external id failed: %s", lbbg.GetId(), err)
			}
		}
	}

	if len(lblis.BackendGroupId) == 0 && len(groupId) == 0 {
		lblis.BackendGroupId = lb.BackendGroupId
	} else if lblis.GetProviderName() == api.CLOUD_PROVIDER_HUAWEI {
		if len(groupId) > 0 {
			group, err := db.FetchByExternalId(HuaweiCachedLbbgManager, groupId)
			if err != nil {
				log.Errorf("Fetch huawei loadbalancer backendgroup by external id %s failed: %s", groupId, err)
			}

			lblis.BackendGroupId = group.(*SHuaweiCachedLbbg).BackendGroupId
		}
	} else if lblis.GetProviderName() == api.CLOUD_PROVIDER_AWS {
		if len(groupId) > 0 {
			group, err := db.FetchByExternalId(AwsCachedLbbgManager, groupId)
			if err != nil {
				log.Errorf("Fetch aws loadbalancer backendgroup by external id %s failed: %s", groupId, err)
			} else {
				lblis.BackendGroupId = group.(*SAwsCachedLbbg).BackendGroupId
			}
		}
	} else if group, err := db.FetchByExternalId(LoadbalancerBackendGroupManager, groupId); err == nil {
		lblis.BackendGroupId = group.GetId()
	}
}

func (lblis *SLoadbalancerListener) syncRemoveCloudLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lblis)
	defer lockman.ReleaseObject(ctx, lblis)

	err := lblis.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		err = lblis.SetStatus(userCred, api.LB_STATUS_UNKNOWN, "sync to delete")
	} else {
		lblis.LBPendingDelete(ctx, userCred)
	}
	return err
}

func (lblis *SLoadbalancerListener) SyncWithCloudLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, extListener cloudprovider.ICloudLoadbalancerListener, syncOwnerId mcclient.IIdentityProvider) error {
	diff, err := db.UpdateWithLock(ctx, lblis, func() error {
		lblis.constructFieldsFromCloudListener(userCred, lb, extListener)
		return nil
	})
	if err != nil {
		return err
	}

	db.OpsLog.LogSyncUpdate(lblis, diff, userCred)

	SyncCloudProject(userCred, lblis, syncOwnerId, extListener, lblis.ManagerId)

	return nil
}

func (man *SLoadbalancerListenerManager) newFromCloudLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, extListener cloudprovider.ICloudLoadbalancerListener, syncOwnerId mcclient.IIdentityProvider) (*SLoadbalancerListener, error) {
	lblis := &SLoadbalancerListener{}
	lblis.SetModelManager(man, lblis)

	lblis.LoadbalancerId = lb.Id
	lblis.ExternalId = extListener.GetGlobalId()

	newName, err := db.GenerateName(man, syncOwnerId, extListener.GetName())
	if err != nil {
		return nil, err
	}
	lblis.Name = newName

	lblis.constructFieldsFromCloudListener(userCred, lb, extListener)

	err = man.TableSpec().Insert(lblis)
	if err != nil {
		return nil, err
	}

	groupId := extListener.GetBackendGroupId()
	if lblis.GetProviderName() == api.CLOUD_PROVIDER_HUAWEI && len(groupId) > 0 {
		group, err := db.FetchByExternalId(HuaweiCachedLbbgManager, groupId)
		if err != nil {
			log.Errorf("Fetch huawei loadbalancer backendgroup by external id %s failed: %s", groupId, err)
		}

		cachedGroup := group.(*SHuaweiCachedLbbg)
		_, err = db.UpdateWithLock(context.Background(), cachedGroup, func() error {
			cachedGroup.AssociatedId = lblis.GetId()
			cachedGroup.AssociatedType = api.LB_ASSOCIATE_TYPE_LISTENER
			return nil
		})
		if err != nil {
			log.Errorf("Update huawei loadbalancer backendgroup cache %s failed: %s", groupId, err)
		}
	}

	SyncCloudProject(userCred, lblis, syncOwnerId, extListener, lblis.ManagerId)

	db.OpsLog.LogEvent(lblis, db.ACT_CREATE, lblis.GetShortDesc(ctx), userCred)

	return lblis, nil
}

func (manager *SLoadbalancerListenerManager) InitializeData() error {
	listeners := []SLoadbalancerListener{}
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
	}
	return nil
}

func (manager *SLoadbalancerListenerManager) GetResourceCount() ([]db.SProjectResourceCount, error) {
	virts := manager.Query().IsFalse("pending_deleted")
	return db.CalculateProjectResourceCount(virts)
}
