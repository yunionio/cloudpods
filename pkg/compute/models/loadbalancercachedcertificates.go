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
	"yunion.io/x/pkg/util/rand"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SCachedLoadbalancerCertificateManager struct {
	SLoadbalancerLogSkipper
	db.SStatusStandaloneResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
	SLoadbalancerCertificateResourceBaseManager
}

var CachedLoadbalancerCertificateManager *SCachedLoadbalancerCertificateManager

func init() {
	CachedLoadbalancerCertificateManager = &SCachedLoadbalancerCertificateManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SCachedLoadbalancerCertificate{},
			"cachedloadbalancercertificates_tbl",
			"cachedloadbalancercertificate",
			"cachedloadbalancercertificates",
		),
	}

	CachedLoadbalancerCertificateManager.SetVirtualObject(CachedLoadbalancerCertificateManager)
}

type SCachedLoadbalancerCertificate struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SManagedResourceBase     // 云账号ID
	SCloudregionResourceBase // Region ID

	SLoadbalancerCertificateResourceBase `width:"128" charset:"ascii" nullable:"false" create:"required"  index:"true" list:"user"`
}

func (manager *SCachedLoadbalancerCertificateManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeProject
}

func (self *SCachedLoadbalancerCertificate) GetOwnerId() mcclient.IIdentityProvider {
	cert, err := self.GetCertificate()
	if err != nil {
		return nil
	}
	return cert.GetOwnerId()
}

func (manager *SCachedLoadbalancerCertificateManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	certId, _ := data.GetString("certificate_id")
	if len(certId) > 0 {
		cert, err := db.FetchById(LoadbalancerCertificateManager, certId)
		if err != nil {
			return nil, errors.Wrapf(err, "db.FetchById(LoadbalancerCertificateManager, %s)", certId)
		}
		return cert.(*SLoadbalancerCertificate).GetOwnerId(), nil
	}
	return db.FetchProjectInfo(ctx, data)
}

func (manager *SCachedLoadbalancerCertificateManager) FilterByOwner(q *sqlchemy.SQuery, userCred mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if userCred != nil {
		sq := LoadbalancerCertificateManager.Query("id")
		switch scope {
		case rbacscope.ScopeProject:
			sq = sq.Equals("tenant_id", userCred.GetProjectId())
			return q.In("certificate_id", sq.SubQuery())
		case rbacscope.ScopeDomain:
			sq = sq.Equals("domain_id", userCred.GetProjectDomainId())
			return q.In("certificate_id", sq.SubQuery())
		}
	}
	return q
}

func (self *SCachedLoadbalancerCertificate) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SCachedLoadbalancerCertificate) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SCachedLoadbalancerCertificate) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	self.SetStatus(userCred, api.LB_STATUS_DELETING, "")
	return self.StartLoadBalancerCertificateDeleteTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (lbcert *SCachedLoadbalancerCertificate) StartLoadBalancerCertificateDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	err := func() error {
		task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerCertificateDeleteTask", lbcert, userCred, params, parentTaskId, "", nil)
		if err != nil {
			return errors.Wrapf(err, "NewTask")
		}
		return task.ScheduleRun(nil)
	}()
	if err != nil {
		lbcert.SetStatus(userCred, api.LB_STATUS_DELETE_FAILED, err.Error())
	}
	return err
}

func (man *SCachedLoadbalancerCertificateManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CachedLoadbalancerCertificateDetails {
	rows := make([]api.CachedLoadbalancerCertificateDetails, len(objs))

	stdRows := man.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := man.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := man.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	certRows := man.SLoadbalancerCertificateResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.CachedLoadbalancerCertificateDetails{
			StatusStandaloneResourceDetails:     stdRows[i],
			ManagedResourceInfo:                 manRows[i],
			CloudregionResourceInfo:             regionRows[i],
			LoadbalancerCertificateResourceInfo: certRows[i],
		}
	}

	return rows
}

func (lbcert *SCachedLoadbalancerCertificate) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	provider, err := lbcert.GetDriver(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDriver")
	}
	region, err := lbcert.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegion")
	}
	return provider.GetIRegionById(region.ExternalId)
}

func (lbcert *SCachedLoadbalancerCertificate) GetRegion() (*SCloudregion, error) {
	return lbcert.SCloudregionResourceBase.GetRegion()
}

func (man *SCachedLoadbalancerCertificateManager) GetOrCreateCachedCertificate(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, lblis *SLoadbalancerListener, cert *SLoadbalancerCertificate) (*SCachedLoadbalancerCertificate, error) {
	ownerProjId := provider.ProjectId

	lockman.LockClass(ctx, man, ownerProjId)
	defer lockman.ReleaseClass(ctx, man, ownerProjId)

	cache, err := func() (*SCachedLoadbalancerCertificate, error) {
		region, err := lblis.GetRegion()
		if err != nil {
			return nil, err
		}
		lbcert, err := man.getLoadbalancerCertificateByRegion(provider, region.Id, cert.Id)
		if err == nil {
			return lbcert, nil
		}

		if errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, errors.Wrap(err, "getLoadbalancerCertificateByRegion")
		}

		lbcert = &SCachedLoadbalancerCertificate{}
		lbcert.SetModelManager(CachedLoadbalancerCertificateManager, lbcert)
		lbcert.ManagerId = provider.Id
		lbcert.CloudregionId = region.Id
		lbcert.Name = cert.Name
		lbcert.Description = cert.Description
		lbcert.CertificateId = cert.Id

		err = man.TableSpec().Insert(ctx, lbcert)
		if err != nil {
			return nil, errors.Wrap(err, "Insert")
		}

		return lbcert, nil
	}()
	if err != nil {
		return nil, err
	}
	if len(cache.ExternalId) > 0 {
		return cache, nil
	}
	err = cache.CreateICertificate(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateICertificate")
	}
	return cache, nil
}

func (lbcert *SCachedLoadbalancerCertificate) CreateICertificate(ctx context.Context) error {
	iRegion, err := lbcert.GetIRegion(ctx)
	if err != nil {
		return errors.Wrapf(err, "lbcert.GetIRegion")
	}

	localCert, err := lbcert.GetCertificate()
	if err != nil {
		return errors.Wrapf(err, "GetCertificate")
	}

	certificate := &cloudprovider.SLoadbalancerCertificate{
		Name:        fmt.Sprintf("%s-%s", lbcert.Name, rand.String(4)),
		PrivateKey:  localCert.PrivateKey,
		Certificate: localCert.Certificate,
	}
	iLoadbalancerCert, err := iRegion.CreateILoadBalancerCertificate(certificate)
	if err != nil {
		return errors.Wrap(err, "iRegion.CreateILoadBalancerCertificate")
	}
	_, err = db.Update(lbcert, func() error {
		lbcert.ExternalId = iLoadbalancerCert.GetGlobalId()
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "db.SetExternalId")
	}
	return nil
}

func (lbcert *SCachedLoadbalancerCertificate) StartLoadbalancerCertificateCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerCertificateCreateTask", lbcert, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (self *SCloudprovider) newFromCloudLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudLoadbalancerCertificate, region *SCloudregion) error {
	lbcert := &SCachedLoadbalancerCertificate{}
	lbcert.SetModelManager(CachedLoadbalancerCertificateManager, lbcert)

	lbcert.ExternalId = ext.GetGlobalId()
	lbcert.ManagerId = self.Id
	if region.GetDriver().IsCertificateBelongToRegion() {
		lbcert.CloudregionId = region.Id
	}

	tenantIds := []string{self.GetOwnerId().GetProjectDomainId()}
	c := &SLoadbalancerCertificate{}
	c.SetModelManager(LoadbalancerCertificateManager, c)
	for _, tenantId := range tenantIds {
		q1 := LoadbalancerCertificateManager.Query()
		q1 = q1.Equals("fingerprint", ext.GetFingerprint())
		q1 = q1.Equals("tenant_id", tenantId)
		cnt, err := q1.CountWithError()
		if err != nil {
			return errors.Wrapf(err, "CountWithError")
		}
		if cnt > 0 {
			err = q1.First(c)
			if err != nil {
				return errors.Wrapf(err, "q1.First")
			}
			break
		}
	}
	if len(c.Id) == 0 {
		// other information's
		c.Name = ext.GetName()
		c.Certificate = ext.GetPublickKey()
		c.PrivateKey = ext.GetPrivateKey()
		c.Fingerprint = ext.GetFingerprint()
		c.ProjectId = tenantIds[0]
		c.CommonName = ext.GetCommonName()
		c.SubjectAlternativeNames = ext.GetSubjectAlternativeNames()
		c.NotAfter = ext.GetExpireTime()
		c.PublicScope = string(rbacscope.ScopeDomain)
		c.IsPublic = true

		err := LoadbalancerCertificateManager.TableSpec().Insert(ctx, c)
		if err != nil {
			return errors.Wrapf(err, "Insert lbcert")
		}

		SyncCloudProject(ctx, userCred, c, self.GetOwnerId(), ext, self.GetId())
	}
	lbcert.CertificateId = c.Id
	lbcert.Name = ext.GetName()

	err := CachedLoadbalancerCertificateManager.TableSpec().Insert(ctx, lbcert)
	if err != nil {
		return errors.Wrapf(err, "Insert cache lbert")
	}

	syncMetadata(ctx, userCred, lbcert, ext)
	db.OpsLog.LogEvent(lbcert, db.ACT_CREATE, lbcert.GetShortDesc(ctx), userCred)
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    lbcert,
		Action: notifyclient.ActionSyncCreate,
	})

	return nil
}

func (lbcert *SCachedLoadbalancerCertificate) SyncWithCloudLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudLoadbalancerCertificate) error {
	diff, err := db.Update(lbcert, func() error {
		lbcert.Name = ext.GetName()
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}

	syncMetadata(ctx, userCred, lbcert, ext)
	db.OpsLog.LogSyncUpdate(lbcert, diff, userCred)
	if len(diff) > 0 {
		notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
			Obj:    lbcert,
			Action: notifyclient.ActionSyncUpdate,
		})
	}
	return nil
}

func (lbcert *SCachedLoadbalancerCertificate) syncRemoveCloudLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbcert)
	defer lockman.ReleaseObject(ctx, lbcert)

	err := lbcert.RealDelete(ctx, userCred)
	if err != nil {
		return errors.Wrapf(err, "lbcert.RealDelete")
	}
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    lbcert,
		Action: notifyclient.ActionSyncDelete,
	})
	return nil
}

func (man *SCachedLoadbalancerCertificateManager) getLoadbalancerCertificateByRegion(provider *SCloudprovider, regionId string, localCertificateId string) (*SCachedLoadbalancerCertificate, error) {
	certificates := []SCachedLoadbalancerCertificate{}
	q := man.Query().Equals("manager_id", provider.Id).Equals("certificate_id", localCertificateId)
	regionDriver, err := provider.GetRegionDriver()
	if err != nil {
		return nil, errors.Wrap(err, "GetRegionDriver")
	}

	if regionDriver.IsCertificateBelongToRegion() {
		q = q.Equals("cloudregion_id", regionId)
	}

	if err := db.FetchModelObjects(man, q, &certificates); err != nil {
		log.Errorf("failed to get lb certificate for region: %v provider: %v error: %v", regionId, provider, err)
		return nil, err
	}

	if len(certificates) >= 1 {
		return &certificates[0], nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SCloudprovider) getLoadbalancerCertificatesByRegion(region *SCloudregion) ([]SCachedLoadbalancerCertificate, error) {
	q := CachedLoadbalancerCertificateManager.Query().Equals("manager_id", self.Id)
	if region.GetDriver().IsCertificateBelongToRegion() {
		q = q.Equals("cloudregion_id", region.Id)
	}

	ret := []SCachedLoadbalancerCertificate{}
	err := db.FetchModelObjects(CachedLoadbalancerCertificateManager, q, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return ret, nil
}

func (self *SCloudprovider) SyncLoadbalancerCertificates(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	region *SCloudregion,
	certificates []cloudprovider.ICloudLoadbalancerCertificate,
	xor bool,
) compare.SyncResult {
	lockman.LockRawObject(ctx, CachedLoadbalancerCertificateManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, region.Id))
	defer lockman.ReleaseRawObject(ctx, CachedLoadbalancerCertificateManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, region.Id))

	syncResult := compare.SyncResult{}

	dbCertificates, err := self.getLoadbalancerCertificatesByRegion(region)
	if err != nil {
		syncResult.Error(errors.Wrapf(err, "getLoadbalancerCertificatesByRegion"))
		return syncResult
	}

	removed := []SCachedLoadbalancerCertificate{}
	commondb := []SCachedLoadbalancerCertificate{}
	commonext := []cloudprovider.ICloudLoadbalancerCertificate{}
	added := []cloudprovider.ICloudLoadbalancerCertificate{}

	err = compare.CompareSets(dbCertificates, certificates, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(errors.Wrapf(err, "compare.CompareSets"))
		return syncResult
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].syncRemoveCloudLoadbalancerCertificate(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
			continue
		}
		syncResult.Delete()
	}
	if !xor {
		for i := 0; i < len(commondb); i++ {
			err = commondb[i].SyncWithCloudLoadbalancerCertificate(ctx, userCred, commonext[i])
			if err != nil {
				syncResult.UpdateError(err)
				continue
			}
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		err := self.newFromCloudLoadbalancerCertificate(ctx, userCred, added[i], region)
		if err != nil {
			syncResult.AddError(err)
			continue
		}
		syncResult.Add()
	}
	return syncResult
}

func (man *SCachedLoadbalancerCertificateManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CachedLoadbalancerCertificateListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = man.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	q, err = man.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}
	q, err = man.SLoadbalancerCertificateResourceBaseManager.ListItemFilter(ctx, q, userCred, query.LoadbalancerCertificateFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLoadbalancerCertificateResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (man *SCachedLoadbalancerCertificateManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CachedLoadbalancerCertificateListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SLoadbalancerCertificateResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.LoadbalancerCertificateFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLoadbalancerCertificateResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (man *SCachedLoadbalancerCertificateManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SLoadbalancerCertificateResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SCachedLoadbalancerCertificateManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SLoadbalancerCertificateResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SLoadbalancerCertificateResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SLoadbalancerCertificateResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}

func (man *SCachedLoadbalancerCertificateManager) InitializeData() error {
	return nil
}
