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
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=webapp
// +onecloud:swagger-gen-model-plural=webapps
type SAppManager struct {
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	db.SEnabledResourceBaseManager

	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
	SNetworkResourceBaseManager
}

type SApp struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase
	db.SEnabledResourceBase

	SManagedResourceBase
	SCloudregionResourceBase
	SNetworkResourceBase

	TechStack           string `width:"64" charset:"ascii" nullable:"false" get:"user" list:"user"`
	OsType              string `width:"12" charset:"ascii" nullable:"true" get:"user" list:"user"`
	IpAddr              string `width:"32" charset:"ascii" nullable:"true" list:"user"`
	Hostname            string `width:"256" charset:"utf8" nullable:"true" list:"user"`
	ServerFarm          string `width:"64" charset:"utf8" nullable:"true" list:"user"`
	PublicNetworkAccess string `width:"32" charset:"utf8" nullable:"true" list:"user"`
}

var AppManager *SAppManager

func init() {
	AppManager = &SAppManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SApp{},
			"apps_tbl",
			"webapp",
			"webapps",
		),
	}
	AppManager.SetVirtualObject(AppManager)
}

func (am *SAppManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.AppListInput) (*sqlchemy.SQuery, error) {
	q, err := am.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = am.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}
	q, err = am.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = am.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	q, err = am.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}
	if query.TechStack != "" {
		q = q.Equals("tech_stack", query.TechStack)
	}

	return q, nil
}

func (am *SAppManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.AppListInput) (*sqlchemy.SQuery, error) {
	var err error

	q, err = am.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}
	q, err = am.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = am.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "CloudregionResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (am *SAppManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = am.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = am.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = am.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	switch field {
	case "tech_stack":
		q = am.Query("tech_stack").Distinct()
	}
	return q, nil
}

func (am *SAppManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	var err error
	q, err = am.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(am.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = am.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}

	if keys.ContainsAny(am.SCloudregionResourceBaseManager.GetExportKeys()...) {
		q, err = am.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}

func (am *SAppManager) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.AppDetails, error) {
	return api.AppDetails{}, nil
}

func (a *SApp) GetAppEnvironments() ([]SAppEnvironment, error) {
	q := AppEnvironmentManager.Query().Equals("app_id", a.Id)
	ret := []SAppEnvironment{}
	err := db.FetchModelObjects(AppEnvironmentManager, q, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return ret, nil
}

func (am *SAppManager) GetApps(providerId string) ([]SApp, error) {
	q := AppManager.Query().Equals("manager_id", providerId)
	ret := []SApp{}
	err := db.FetchModelObjects(AppManager, q, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return ret, nil
}

func (self *SCloudregion) GetApps(managerId string) ([]SApp, error) {
	q := AppManager.Query().Equals("cloudregion_id", self.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	ret := []SApp{}
	err := db.FetchModelObjects(AppManager, q, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return ret, nil
}

func (self *SCloudregion) SyncApps(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	exts []cloudprovider.ICloudApp,
	xor bool,
) compare.SyncResult {
	lockman.LockRawObject(ctx, AppManager.KeywordPlural(), fmt.Sprintf("%s-%s", provider.Id, self.Id))
	defer lockman.ReleaseRawObject(ctx, AppManager.KeywordPlural(), fmt.Sprintf("%s-%s", provider.Id, self.Id))
	result := compare.SyncResult{}
	apps, err := self.GetApps(provider.Id)
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SApp, 0)
	commondb := make([]SApp, 0)
	commonext := make([]cloudprovider.ICloudApp, 0)
	added := make([]cloudprovider.ICloudApp, 0)
	// compare
	err = compare.CompareSets(apps, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	// remove
	for i := 0; i < len(removed); i++ {
		err := removed[i].syncRemoveCloudApp(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	if !xor {
		// sync with cloud app
		for i := 0; i < len(commondb); i++ {
			err := commondb[i].SyncWithCloudApp(ctx, userCred, provider, commonext[i])
			if err != nil {
				result.UpdateError(err)
				continue
			}
			result.Update()
		}
	}

	// new one
	for i := 0; i < len(added); i++ {
		_, err := self.newFromCloudApp(ctx, userCred, provider, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}

func (self *SCloudregion) newFromCloudApp(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudApp) (*SApp, error) {
	app := SApp{}
	app.SetModelManager(AppManager, &app)

	app.ExternalId = ext.GetGlobalId()
	app.CloudregionId = self.Id
	app.ManagerId = provider.Id
	app.IsEmulated = ext.IsEmulated()
	app.Status = ext.GetStatus()
	app.TechStack = ext.GetTechStack()
	app.Name = ext.GetName()
	app.Enabled = tristate.True
	app.OsType = string(ext.GetOsType())
	app.IpAddr = ext.GetIpAddress()
	app.Hostname = ext.GetHostname()
	app.ServerFarm = ext.GetServerFarm()
	app.PublicNetworkAccess = ext.GetPublicNetworkAccess()

	if netId := ext.GetNetworkId(); len(netId) > 0 {
		network, err := db.FetchByExternalIdAndManagerId(NetworkManager, netId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			wire := WireManager.Query().SubQuery()
			vpc := VpcManager.Query().SubQuery()
			return q.Join(wire, sqlchemy.Equals(wire.Field("id"), q.Field("wire_id"))).
				Join(vpc, sqlchemy.Equals(vpc.Field("id"), wire.Field("vpc_id"))).
				Filter(sqlchemy.Equals(vpc.Field("manager_id"), provider.Id))
		})
		if err != nil {
			log.Errorf("fetch network %s error: %v", netId, err)
		} else {
			app.NetworkId = network.GetId()
		}
	}

	err := AppManager.TableSpec().Insert(ctx, &app)
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert app")
	}
	aes, err := ext.GetEnvironments()
	if err != nil {
		return &app, errors.Wrap(err, "unable to GetEnvironments")
	}
	result := app.SyncAppEnvironments(ctx, userCred, provider, aes)
	if result.IsError() {
		return &app, errors.Wrap(result.AllError(), "unable to SyncAppEnvironments")
	}
	SyncCloudProject(ctx, userCred, &app, provider.GetOwnerId(), ext, provider)
	syncVirtualResourceMetadata(ctx, userCred, &app, ext, false)

	db.OpsLog.LogEvent(&app, db.ACT_CREATE, app.GetShortDesc(ctx), userCred)
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    &app,
		Action: notifyclient.ActionSyncCreate,
	})

	return &app, nil
}

func (am *SAppManager) purgeAll(ctx context.Context, userCred mcclient.TokenCredential, providerId string) error {
	apps, err := am.GetApps(providerId)
	if err != nil {
		return errors.Wrapf(err, "unable to GetApps of provider id %s", providerId)
	}
	for i := range apps {
		err = apps[i].purge(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *SApp) GetChangeOwnerCandidateDomainIds() []string {
	return nil
}

func (a *SApp) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	aes, err := a.GetAppEnvironments()
	if err != nil {
		return errors.Wrap(err, "unable to GetAppEnvironments")
	}
	for i := range aes {
		err = aes[i].Delete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "unable to Delete AppEnvironment %s", aes[i].Id)
		}
	}
	return a.Delete(ctx, userCred)
}

func (a *SApp) syncRemoveCloudApp(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := a.purge(ctx, userCred)
	if err != nil {
		return err
	}
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    a,
		Action: notifyclient.ActionSyncDelete,
	})
	return nil
}

func (a *SApp) SyncWithCloudApp(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudApp) error {
	diff, err := db.UpdateWithLock(ctx, a, func() error {
		a.ExternalId = ext.GetGlobalId()
		a.Status = ext.GetStatus()
		a.TechStack = ext.GetTechStack()
		a.OsType = string(ext.GetOsType())
		a.IpAddr = ext.GetIpAddress()
		a.Hostname = ext.GetHostname()
		a.ServerFarm = ext.GetServerFarm()
		a.PublicNetworkAccess = ext.GetPublicNetworkAccess()

		if netId := ext.GetNetworkId(); len(netId) > 0 && len(a.NetworkId) == 0 {
			network, err := db.FetchByExternalIdAndManagerId(NetworkManager, netId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				wire := WireManager.Query().SubQuery()
				vpc := VpcManager.Query().SubQuery()
				return q.Join(wire, sqlchemy.Equals(wire.Field("id"), q.Field("wire_id"))).
					Join(vpc, sqlchemy.Equals(vpc.Field("id"), wire.Field("vpc_id"))).
					Filter(sqlchemy.Equals(vpc.Field("manager_id"), provider.Id))
			})
			if err != nil {
				log.Errorf("fetch network %s error: %v", netId, err)
			} else {
				a.NetworkId = network.GetId()
			}
		}

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "db.Update")
	}
	aes, err := ext.GetEnvironments()
	if err != nil {
		return errors.Wrapf(err, "unable to GetAppEnvironments for ICloudApp %s", ext.GetGlobalId())
	}
	result := a.SyncAppEnvironments(ctx, userCred, provider, aes)
	if result.IsError() {
		return result.AllError()
	}
	if len(diff) > 0 {
		notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
			Obj:    a,
			Action: notifyclient.ActionSyncUpdate,
		})
	}
	if account, _ := provider.GetCloudaccount(); account != nil {
		syncVirtualResourceMetadata(ctx, userCred, a, ext, account.ReadOnly)
	}
	return nil
}

func (am *SAppManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []api.AppDetails {
	rows := make([]api.AppDetails, len(objs))
	virtRows := am.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := am.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regRows := am.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	netRows := am.SNetworkResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i].VirtualResourceDetails = virtRows[i]
		rows[i].ManagedResourceInfo = manRows[i]
		rows[i].CloudregionResourceInfo = regRows[i]
		rows[i].Network = netRows[i].Network
		rows[i].VpcId = netRows[i].VpcId
		rows[i].Vpc = netRows[i].Vpc
	}
	return rows
}

func (a *SApp) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var openTask = true
	count, err := taskman.TaskManager.QueryTasksOfObject(a, time.Now().Add(-3*time.Minute), &openTask).CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, httperrors.NewBadRequestError("WebApp has %d task active, can't sync status", count)
	}

	return nil, StartResourceSyncStatusTask(ctx, userCred, a, "AppSyncstatusTask", "")
}

func (a *SApp) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	region, err := a.SCloudregionResourceBase.GetRegion()
	if err != nil {
		return nil, errors.Wrap(err, "GetRegion")
	}
	provider, err := a.GetDriver(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "GetDriver")
	}
	return provider.GetIRegionById(region.GetExternalId())
}

func (a *SApp) GetIApp(ctx context.Context) (cloudprovider.ICloudApp, error) {
	if len(a.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty externalId")
	}
	iRegion, err := a.GetIRegion(ctx)
	if err != nil {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "GetIRegion")
	}
	return iRegion.GetICloudAppById(a.ExternalId)
}

func (self *SApp) PerformRemoteUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.MongoDBRemoteUpdateInput) (jsonutils.JSONObject, error) {
	err := self.StartRemoteUpdateTask(ctx, userCred, (input.ReplaceTags != nil && *input.ReplaceTags), "")
	if err != nil {
		return nil, errors.Wrap(err, "StartRemoteUpdateTask")
	}
	return nil, nil
}

func (self *SApp) StartRemoteUpdateTask(ctx context.Context, userCred mcclient.TokenCredential, replaceTags bool, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewBool(replaceTags), "replace_tags")
	task, err := taskman.TaskManager.NewTask(ctx, "AppRemoteUpdateTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_UPDATE_TAGS, "StartRemoteUpdateTask")
	return task.ScheduleRun(nil)
}

func (self *SApp) OnMetadataUpdated(ctx context.Context, userCred mcclient.TokenCredential) {
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

// 获取混合连接
func (self *SApp) GetDetailsHybirdConnections(ctx context.Context, userCred mcclient.TokenCredential, input jsonutils.JSONObject) (*api.AppHybirdConnectionOutput, error) {
	iApp, err := self.GetIApp(ctx)
	if err != nil {
		return nil, err
	}
	ret := &api.AppHybirdConnectionOutput{
		Data: []api.AppHybirdConnection{},
	}
	connections, err := iApp.GetHybirdConnections()
	if err != nil {
		return nil, err
	}
	for _, conn := range connections {
		ret.Data = append(ret.Data, api.AppHybirdConnection{
			Id:        conn.GetGlobalId(),
			Name:      conn.GetName(),
			Hostname:  conn.GetHostname(),
			Namespace: conn.GetNamespace(),
			Port:      conn.GetPort(),
		})
	}
	return ret, nil
}

// 获取备份列表
func (self *SApp) GetDetailsBackups(ctx context.Context, userCred mcclient.TokenCredential, input jsonutils.JSONObject) (*api.AppBackupOutput, error) {
	iApp, err := self.GetIApp(ctx)
	if err != nil {
		return nil, err
	}
	ret := &api.AppBackupOutput{
		Data: []api.AppBackup{},
	}

	backups, err := iApp.GetBackups()
	if err != nil {
		return nil, err
	}
	for _, backup := range backups {
		ret.Data = append(ret.Data, api.AppBackup{
			Id:   backup.GetGlobalId(),
			Name: backup.GetName(),
			Type: backup.GetType(),
		})
	}
	opts := iApp.GetBackupConfig()
	jsonutils.Update(&ret.BackupConfig, opts)
	return ret, nil
}

// 获取证书列表
func (self *SApp) GetDetailsCertificates(ctx context.Context, userCred mcclient.TokenCredential, input jsonutils.JSONObject) (*api.AppCertificateOutput, error) {
	iApp, err := self.GetIApp(ctx)
	if err != nil {
		return nil, err
	}
	ret := &api.AppCertificateOutput{
		Data: []api.AppCertificate{},
	}

	certs, err := iApp.GetCertificates()
	if err != nil {
		return nil, err
	}
	for _, cert := range certs {
		ret.Data = append(ret.Data, api.AppCertificate{
			Id:          cert.GetGlobalId(),
			Name:        cert.GetName(),
			SubjectName: cert.GetSubjectName(),
			Issuer:      cert.GetIssuer(),
			IssueDate:   cert.GetIssueDate(),
			Thumbprint:  cert.GetThumbprint(),
			ExpireTime:  cert.GetExpireTime(),
		})
	}
	return ret, nil
}

// 获取自定义域列表
func (self *SApp) GetDetailsCustomDomains(ctx context.Context, userCred mcclient.TokenCredential, input jsonutils.JSONObject) (*api.AppDomainOutput, error) {
	iApp, err := self.GetIApp(ctx)
	if err != nil {
		return nil, err
	}
	ret := &api.AppDomainOutput{
		Data: []api.AppDomain{},
	}

	domains, err := iApp.GetDomains()
	if err != nil {
		return nil, err
	}
	for _, domain := range domains {
		ret.Data = append(ret.Data, api.AppDomain{
			Id:       domain.GetGlobalId(),
			Name:     domain.GetName(),
			Status:   domain.GetStatus(),
			SslState: domain.GetSslState(),
		})
	}
	return ret, nil
}
