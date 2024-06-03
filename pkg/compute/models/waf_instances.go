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
	"fmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SWafInstanceManager struct {
	db.SEnabledStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
}

var WafInstanceManager *SWafInstanceManager

func init() {
	WafInstanceManager = &SWafInstanceManager{
		SEnabledStatusInfrasResourceBaseManager: db.NewEnabledStatusInfrasResourceBaseManager(
			SWafInstance{},
			"waf_instances_tbl",
			"waf_instance",
			"waf_instances",
		),
	}
	WafInstanceManager.SetVirtualObject(WafInstanceManager)
}

type SWafInstance struct {
	db.SEnabledStatusInfrasResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase

	Type          cloudprovider.TWafType       `width:"20" charset:"ascii" nullable:"false" list:"domain" create:"required"`
	DefaultAction *cloudprovider.DefaultAction `charset:"ascii" nullable:"true" list:"domain" create:"domain_optional"`

	Cname string `width:"256" charset:"utf8" nullable:"true" list:"user" update:"admin"`
	// 前面是否有代理服务
	IsAccessProduct bool     `nullable:"false" default:"false" list:"user" update:"user" create:"optional"`
	AccessHeaders   []string `width:"512" charset:"utf8" nullable:"true" list:"user" update:"admin"`
	// 源站地址
	SourceIps []string `width:"512" charset:"utf8" nullable:"true" list:"user" update:"admin"`
	// 回源地址
	CcList     []string `width:"512" charset:"utf8" nullable:"true" list:"user" update:"admin"`
	HttpPorts  []int    `width:"64" charset:"utf8" nullable:"true" list:"user" update:"admin"`
	HttpsPorts []int    `width:"64" charset:"utf8" nullable:"true" list:"user" update:"admin"`

	UpstreamScheme string `width:"32" charset:"utf8" nullable:"true" list:"user" update:"admin"`
	UpstreamPort   int    `nullable:"true" list:"user" update:"admin"`
	CertId         string `width:"36" charset:"utf8" nullable:"true" list:"user" update:"admin"`
	CertName       string `width:"128" charset:"utf8" nullable:"true" list:"user" update:"admin"`
}

func (manager *SWafInstanceManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudregionManager},
	}
}

func (manager *SWafInstanceManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.WafInstanceCreateInput) (api.WafInstanceCreateInput, error) {
	_region, err := validators.ValidateModel(ctx, userCred, CloudregionManager, &input.CloudregionId)
	if err != nil {
		return input, err
	}
	region := _region.(*SCloudregion)
	_provider, err := validators.ValidateModel(ctx, userCred, CloudproviderManager, &input.CloudproviderId)
	if err != nil {
		return input, err
	}
	provider := _provider.(*SCloudprovider)
	if !provider.IsAvailable() {
		return input, httperrors.NewInputParameterError("cloudprovider %s not available", provider.Name)
	}
	for i := range input.CloudResources {
		switch input.CloudResources[i].Type {
		case LoadbalancerManager.Keyword():
			_lb, err := validators.ValidateModel(ctx, userCred, LoadbalancerManager, &input.CloudResources[i].Id)
			if err != nil {
				return input, err
			}
			lb := _lb.(*SLoadbalancer)
			if lb.ManagerId != provider.GetId() {
				return input, httperrors.NewConflictError("lb %s does not belong to account %s", lb.Name, provider.GetName())
			}
		case GuestManager.Keyword():
			_server, err := validators.ValidateModel(ctx, userCred, GuestManager, &input.CloudResources[i].Id)
			if err != nil {
				return input, err
			}
			server := _server.(*SGuest)
			host, _ := server.GetHost()
			if host.ManagerId != provider.GetId() {
				return input, httperrors.NewConflictError("server %s does not belong to account %s", server.Name, provider.GetName())
			}
		default:
			return input, httperrors.NewInputParameterError("invalid %d resource type %s", i, input.CloudResources[i].Type)
		}
	}

	input, err = region.GetDriver().ValidateCreateWafInstanceData(ctx, userCred, input)
	if err != nil {
		return input, err
	}

	input.SetEnabled()
	input.EnabledStatusInfrasResourceBaseCreateInput, err = manager.SEnabledStatusInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusInfrasResourceBaseCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (self *SWafInstance) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SEnabledStatusInfrasResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	self.StartCreateTask(ctx, userCred, data.(*jsonutils.JSONDict))
}

func (self *SWafInstance) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict) error {
	task, err := taskman.TaskManager.NewTask(ctx, "WafCreateTask", self, userCred, params, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, api.WAF_STATUS_CREATING, "")
	return task.ScheduleRun(nil)
}

func (manager *SWafInstanceManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.WafInstanceDetails {
	rows := make([]api.WafInstanceDetails, len(objs))
	stdRows := manager.SEnabledStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	insIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.WafInstanceDetails{
			EnabledStatusInfrasResourceBaseDetails: stdRows[i],
			ManagedResourceInfo:                    managerRows[i],
			CloudregionResourceInfo:                regionRows[i],
		}
		ins := objs[i].(*SWafInstance)
		insIds[i] = ins.Id
	}
	type WafRule struct {
		api.SWafRule
		WafInstanceId string
	}
	rules := []WafRule{}
	q := WafRuleManager.Query().In("waf_instance_id", insIds)
	err := q.All(&rules)
	if err != nil {
		return rows
	}
	ruleMaps := map[string][]api.SWafRule{}
	for _, rule := range rules {
		_, ok := ruleMaps[rule.WafInstanceId]
		if !ok {
			ruleMaps[rule.WafInstanceId] = []api.SWafRule{}
		}
		ruleMaps[rule.WafInstanceId] = append(ruleMaps[rule.WafInstanceId], rule.SWafRule)
	}
	for i := range rows {
		rows[i].Rules, _ = ruleMaps[insIds[i]]
	}
	return rows
}

// 列出WAF实例
func (manager *SWafInstanceManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.WafInstanceListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SWafInstanceManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SEnabledStatusInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	q, err = manager.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SWafInstanceManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.WafInstanceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SWafInstanceManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusInfrasResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}

func (self *SCloudregion) GetWafInstances(managerId string) ([]SWafInstance, error) {
	q := WafInstanceManager.Query().Equals("cloudregion_id", self.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	wafs := []SWafInstance{}
	err := db.FetchModelObjects(WafInstanceManager, q, &wafs)
	return wafs, err
}

func (self *SCloudregion) SyncWafInstances(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	exts []cloudprovider.ICloudWafInstance,
	xor bool,
) ([]SWafInstance, []cloudprovider.ICloudWafInstance, compare.SyncResult) {
	lockman.LockRawObject(ctx, WafInstanceManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, provider.Id))
	defer lockman.ReleaseRawObject(ctx, WafInstanceManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, provider.Id))

	result := compare.SyncResult{}

	localWafs := []SWafInstance{}
	remoteWafs := []cloudprovider.ICloudWafInstance{}

	dbWafs, err := self.GetWafInstances(provider.Id)
	if err != nil {
		result.Error(err)
		return nil, nil, result
	}

	removed := make([]SWafInstance, 0)
	commondb := make([]SWafInstance, 0)
	commonext := make([]cloudprovider.ICloudWafInstance, 0)
	added := make([]cloudprovider.ICloudWafInstance, 0)
	if err := compare.CompareSets(dbWafs, exts, &removed, &commondb, &commonext, &added); err != nil {
		result.Error(err)
		return nil, nil, result
	}

	for i := 0; i < len(removed); i++ {
		err := removed[i].syncRemove(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	if !xor {
		for i := 0; i < len(commondb); i++ {
			err := commondb[i].SyncWithCloudWafInstance(ctx, userCred, commonext[i])
			if err != nil {
				result.UpdateError(err)
				continue
			}
			localWafs = append(localWafs, commondb[i])
			remoteWafs = append(remoteWafs, commonext[i])
			result.Update()
		}
	}

	for i := 0; i < len(added); i++ {
		newWaf, err := self.newFromCloudWafInstance(ctx, userCred, provider, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		localWafs = append(localWafs, *newWaf)
		remoteWafs = append(remoteWafs, added[i])
		result.Add()
	}

	return localWafs, remoteWafs, result
}

func (self *SWafInstance) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteTask(ctx, userCred)
}

func (self *SWafInstance) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	task, err := taskman.TaskManager.NewTask(ctx, "WafDeleteTask", self, userCred, nil, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, api.WAF_STATUS_DELETING, "")
	return task.ScheduleRun(nil)
}

func (self *SWafInstance) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SWafInstance) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	rules, err := self.GetWafRules()
	if err != nil {
		return errors.Wrapf(err, "GetWafRules")
	}
	for i := range rules {
		err = rules[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "Delete Rule %s", rules[i].Name)
		}
	}
	return self.SEnabledStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SWafInstance) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := self.RealDelete(ctx, userCred)
	if err != nil {
		return err
	}
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    self,
		Action: notifyclient.ActionSyncDelete,
	})
	return nil
}

func (self *SWafInstance) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	region, err := self.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegion")
	}
	provider, err := self.GetDriver(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDriver")
	}
	return provider.GetIRegionById(region.ExternalId)
}

func (self *SWafInstance) GetICloudWafInstance(ctx context.Context) (cloudprovider.ICloudWafInstance, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	iRegion, err := self.GetIRegion(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetIRegion")
	}
	return iRegion.GetICloudWafInstanceById(self.ExternalId)
}

func (self *SWafInstance) SyncWithCloudWafInstance(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudWafInstance) error {
	diff, err := db.Update(self, func() error {
		self.ExternalId = ext.GetGlobalId()
		self.SetEnabled(ext.GetEnabled())
		self.DefaultAction = ext.GetDefaultAction()
		self.Status = ext.GetStatus()
		self.IsAccessProduct = ext.GetIsAccessProduct()
		self.Type = ext.GetWafType()
		self.HttpsPorts = ext.GetHttpsPorts()
		self.HttpPorts = ext.GetHttpPorts()
		self.Cname = ext.GetCname()
		self.SourceIps = ext.GetSourceIps()
		if ccList := ext.GetCcList(); len(ccList) > 0 {
			self.CcList = ccList
		}
		if certId := ext.GetCertId(); len(certId) > 0 {
			self.CertId = certId
		}
		if certName := ext.GetCertName(); len(certName) > 0 {
			self.CertName = certName
		}
		self.UpstreamScheme = ext.GetUpstreamScheme()
		self.UpstreamPort = ext.GetUpstreamPort()
		self.AccessHeaders = ext.GetAccessHeaders()
		return nil
	})
	if len(diff) > 0 {
		notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
			Obj:    self,
			Action: notifyclient.ActionSyncUpdate,
		})
	}
	if account := self.GetCloudaccount(); account != nil {
		syncMetadata(ctx, userCred, self, ext, account.ReadOnly)
	}
	return err
}

func (self *SCloudregion) newFromCloudWafInstance(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudWafInstance) (*SWafInstance, error) {
	waf := &SWafInstance{}
	waf.SetModelManager(WafInstanceManager, waf)
	waf.SetEnabled(ext.GetEnabled())
	waf.CloudregionId = self.Id
	waf.ManagerId = provider.Id
	waf.Status = ext.GetStatus()
	waf.DefaultAction = ext.GetDefaultAction()
	waf.Type = ext.GetWafType()
	waf.ExternalId = ext.GetGlobalId()
	waf.IsAccessProduct = ext.GetIsAccessProduct()
	waf.HttpsPorts = ext.GetHttpsPorts()
	waf.HttpPorts = ext.GetHttpPorts()
	waf.Cname = ext.GetCname()
	waf.UpstreamScheme = ext.GetUpstreamScheme()
	waf.UpstreamPort = ext.GetUpstreamPort()
	waf.SourceIps = ext.GetSourceIps()
	waf.CcList = ext.GetCcList()
	waf.CertId = ext.GetCertId()
	waf.CertName = ext.GetCertName()
	waf.AccessHeaders = ext.GetAccessHeaders()
	var err = func() error {
		lockman.LockRawObject(ctx, WafInstanceManager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, WafInstanceManager.Keyword(), "name")

		var err error
		waf.Name, err = db.GenerateName(ctx, WafInstanceManager, userCred, ext.GetName())
		if err != nil {
			return errors.Wrapf(err, "db.GenerateName")
		}

		return WafInstanceManager.TableSpec().Insert(ctx, waf)
	}()
	if err != nil {
		return nil, err
	}
	syncMetadata(ctx, userCred, waf, ext, false)
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    waf,
		Action: notifyclient.ActionSyncCreate,
	})
	return waf, nil
}

// 获取WAF绑定的资源列表
func (self *SWafInstance) GetDetailsCloudResources(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (cloudprovider.SCloudResources, error) {
	ret := cloudprovider.SCloudResources{}
	iWaf, err := self.GetICloudWafInstance(ctx)
	if err != nil {
		return ret, httperrors.NewGeneralError(errors.Wrapf(err, "GetICloudWafInstance"))
	}
	ret.Data, err = iWaf.GetCloudResources()
	if err != nil {
		return ret, errors.Wrapf(err, "GetCloudResources")
	}
	ret.Total = len(ret.Data)
	return ret, nil
}

// 同步WAF状态
func (self *SWafInstance) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.WafSyncstatusInput) (jsonutils.JSONObject, error) {
	return nil, StartResourceSyncStatusTask(ctx, userCred, self, "WafSyncstatusTask", "")
}

func (self *SWafInstance) PerformRemoteUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.MongoDBRemoteUpdateInput) (jsonutils.JSONObject, error) {
	err := self.StartRemoteUpdateTask(ctx, userCred, (input.ReplaceTags != nil && *input.ReplaceTags), "")
	if err != nil {
		return nil, errors.Wrap(err, "StartRemoteUpdateTask")
	}
	return nil, nil
}

func (self *SWafInstance) StartRemoteUpdateTask(ctx context.Context, userCred mcclient.TokenCredential, replaceTags bool, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewBool(replaceTags), "replace_tags")
	task, err := taskman.TaskManager.NewTask(ctx, "WafInstanceRemoteUpdateTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_UPDATE_TAGS, "StartRemoteUpdateTask")
	return task.ScheduleRun(nil)
}

func (self *SWafInstance) OnMetadataUpdated(ctx context.Context, userCred mcclient.TokenCredential) {
	if len(self.ExternalId) == 0 || options.Options.KeepTagLocalization {
		return
	}
	if account := self.GetCloudaccount(); account != nil && account.ReadOnly {
		return
	}
	err := self.StartRemoteUpdateTask(ctx, userCred, true, "")
	if err != nil {
		log.Errorf("StartRemoteUpdateTask fail: %s", err)
	}
}
