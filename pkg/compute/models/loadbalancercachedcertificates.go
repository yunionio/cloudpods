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
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SCachedLoadbalancerCertificateManager struct {
	SLoadbalancerLogSkipper
	db.SVirtualResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
	SLoadbalancerCertificateResourceBaseManager
}

var CachedLoadbalancerCertificateManager *SCachedLoadbalancerCertificateManager

func init() {
	CachedLoadbalancerCertificateManager = &SCachedLoadbalancerCertificateManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SCachedLoadbalancerCertificate{},
			"cachedloadbalancercertificates_tbl",
			"cachedloadbalancercertificate",
			"cachedloadbalancercertificates",
		),
	}

	CachedLoadbalancerCertificateManager.SetVirtualObject(CachedLoadbalancerCertificateManager)
}

type SCachedLoadbalancerCertificate struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase
	SManagedResourceBase     // 云账号ID
	SCloudregionResourceBase // Region ID

	SLoadbalancerCertificateResourceBase `width:"128" charset:"ascii" nullable:"false" create:"required"  index:"true" list:"user"`
	// CertificateId string `width:"128" charset:"ascii" nullable:"false" create:"required"  index:"true" list:"user" json:"certificate_id"` // 本地证书ID
}

func (self *SCachedLoadbalancerCertificate) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (self *SCachedLoadbalancerCertificate) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (self *SCachedLoadbalancerCertificate) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return false
}

func (self *SCachedLoadbalancerCertificate) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SCachedLoadbalancerCertificate) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (self *SCachedLoadbalancerCertificate) ValidateDeleteCondition(ctx context.Context) error {
	men := []db.IModelManager{
		LoadbalancerListenerManager,
	}
	lbcertId := self.CertificateId
	for _, man := range men {
		t := man.TableSpec().Instance()
		pdF := t.Field("pending_deleted")
		n, err := t.Query().
			Equals("domain_id", self.DomainId).
			Equals("certificate_id", lbcertId).
			Equals("cached_certificate_id", self.GetId()).
			Filter(sqlchemy.OR(sqlchemy.IsNull(pdF), sqlchemy.IsFalse(pdF))).
			CountWithError()
		if err != nil {
			return httperrors.NewInternalServerError("get certificate refcount fail %s", err)
		}
		if n > 0 {
			return httperrors.NewResourceBusyError("certificate %s is still referred to by %d %s",
				lbcertId, n, man.KeywordPlural())
		}
	}
	return nil
}

func (self *SCachedLoadbalancerCertificate) ValidatePurgeCondition(ctx context.Context) error {
	return nil
}

func (self *SCachedLoadbalancerCertificate) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SCachedLoadbalancerCertificate) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	self.SetStatus(userCred, api.LB_STATUS_DELETING, "")
	return self.StartLoadBalancerCertificateDeleteTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (lbcert *SCachedLoadbalancerCertificate) StartLoadBalancerCertificateDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerCertificateDeleteTask", lbcert, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (man *SCachedLoadbalancerCertificateManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	certificateV := validators.NewModelIdOrNameValidator("certificate", "loadbalancercertificate", ownerId)
	regionV := validators.NewModelIdOrNameValidator("cloudregion", "cloudregion", ownerId)
	providerV := validators.NewModelIdOrNameValidator("cloudprovider", "cloudprovider", ownerId)
	keyV := map[string]validators.IValidator{
		"certificate":   certificateV,
		"cloudregion":   regionV,
		"cloudprovider": providerV,
	}

	for _, v := range keyV {
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}

	// validate local cert
	cert := certificateV.Model.(*SLoadbalancerCertificate)
	if len(cert.PrivateKey) == 0 {
		return nil, httperrors.NewResourceNotReadyError("invalid local certificate, private key is empty.")
	} else if len(cert.Certificate) == 0 {
		return nil, httperrors.NewResourceNotReadyError("invalid local certificate, certificate is empty.")
	} else {
		data.Set("certificate", jsonutils.NewString(cert.Certificate))
		data.Set("private_key", jsonutils.NewString(cert.PrivateKey))
	}

	count, err := man.Query().Equals("certificate_id", certificateV.Model.GetId()).Equals("cloudregion_id", regionV.Model.GetId()).IsFalse("deleted").CountWithError()
	if err != nil {
		return nil, err
	}

	if count > 0 {
		return nil, httperrors.NewDuplicateResourceError("the certificate cache in region %s aready exists.", regionV.Model.GetId())
	}

	provider := providerV.Model.(*SCloudprovider)
	data.Set("manager_id", jsonutils.NewString(provider.Id))
	name, _ := db.GenerateName(ctx, man, ownerId, certificateV.Model.GetName())
	data.Set("name", jsonutils.NewString(name))

	input := apis.VirtualResourceCreateInput{}
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal VirtualResourceCreateInput fail %s", err)
	}
	input, err = man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))

	return data, nil
}

func (self *SCachedLoadbalancerCertificate) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SetStatus(userCred, api.LB_CREATING, "")
	if err := self.StartLoadbalancerCertificateCreateTask(ctx, userCred, ""); err != nil {
		log.Errorf("CachedLoadbalancerCertificate.PostCreate %s", err)
	}
	return
}

func (self *SCachedLoadbalancerCertificate) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.CachedLoadbalancerCertificateDetails, error) {
	return api.CachedLoadbalancerCertificateDetails{}, nil
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

	virtRows := man.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := man.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := man.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	certRows := man.SLoadbalancerCertificateResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.CachedLoadbalancerCertificateDetails{
			VirtualResourceDetails:              virtRows[i],
			ManagedResourceInfo:                 manRows[i],
			CloudregionResourceInfo:             regionRows[i],
			LoadbalancerCertificateResourceInfo: certRows[i],
		}
	}

	return rows
}

func (lbcert *SCachedLoadbalancerCertificate) GetIRegion() (cloudprovider.ICloudRegion, error) {
	provider, err := lbcert.GetDriver()
	if err != nil {
		return nil, fmt.Errorf("No cloudprovider for lbcert %s: %s", lbcert.Name, err)
	}
	region := lbcert.GetRegion()
	if region == nil {
		return nil, fmt.Errorf("failed to find region for lbcert %s", lbcert.Name)
	}
	return provider.GetIRegionById(region.ExternalId)
}

func (lbcert *SCachedLoadbalancerCertificate) GetRegion() *SCloudregion {
	region, err := CloudregionManager.FetchById(lbcert.CloudregionId)
	if err != nil {
		log.Errorf("failed to find region for loadbalancer certificate %s", lbcert.Name)
		return nil
	}
	return region.(*SCloudregion)
}

func (man *SCachedLoadbalancerCertificateManager) GetOrCreateCachedCertificate(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, lblis *SLoadbalancerListener, cert *SLoadbalancerCertificate) (*SCachedLoadbalancerCertificate, error) {
	ownerProjId := provider.ProjectId

	lockman.LockClass(ctx, man, ownerProjId)
	defer lockman.ReleaseClass(ctx, man, ownerProjId)

	region := lblis.GetRegion()
	if region == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "loadbalancer listener is not attached to any region?")
	}
	lbcert, err := man.getLoadbalancerCertificateByRegion(provider, region.Id, cert.Id)
	if err == nil {
		return &lbcert, nil
	}

	if err.Error() != "NotFound" {
		return nil, errors.Wrap(err, "cachedLoadbalancerCertificateManager.getCert")
	}

	lbcert = SCachedLoadbalancerCertificate{}
	lbcert.ManagerId = provider.Id
	lbcert.CloudregionId = region.Id
	lbcert.ProjectId = lblis.ProjectId
	lbcert.ProjectSrc = lblis.ProjectSrc
	lbcert.Name = cert.Name
	lbcert.Description = cert.Description
	lbcert.IsSystem = cert.IsSystem
	lbcert.CertificateId = cert.Id

	err = man.TableSpec().Insert(ctx, &lbcert)
	if err != nil {
		return nil, errors.Wrap(err, "cachedLoadbalancerCertificateManager.create")
	}

	return &lbcert, nil
}

func (lbcert *SCachedLoadbalancerCertificate) StartLoadbalancerCertificateCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerCertificateCreateTask", lbcert, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (man *SCachedLoadbalancerCertificateManager) newFromCloudLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extCertificate cloudprovider.ICloudLoadbalancerCertificate, region *SCloudregion, projectId mcclient.IIdentityProvider) (*SCachedLoadbalancerCertificate, error) {
	lbcert := SCachedLoadbalancerCertificate{}
	lbcert.SetModelManager(man, &lbcert)

	lbcert.ExternalId = extCertificate.GetGlobalId()
	lbcert.ManagerId = provider.Id
	lbcert.CloudregionId = region.Id

	c := SLoadbalancerCertificate{}
	q1 := LoadbalancerCertificateManager.Query().IsFalse("pending_deleted")
	q1 = q1.Equals("fingerprint", extCertificate.GetFingerprint())
	q1 = q1.Equals("tenant_id", provider.ProjectId)
	err := q1.First(&c)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			localcert, err := LoadbalancerCertificateManager.CreateCertificate(ctx, userCred, provider, lbcert.Name, extCertificate)
			if err != nil {
				return nil, fmt.Errorf("newFromCloudLoadbalancerCertificate CreateCertificate %s", err)
			}

			lbcert.CertificateId = localcert.Id
		default:
			return nil, fmt.Errorf("newFromCloudLoadbalancerCertificate.QueryCachedLoadbalancerCertificate %s", err)
		}
	} else {
		lbcert.CertificateId = c.Id
	}

	err = func() error {
		lockman.LockRawObject(ctx, man.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, man.Keyword(), "name")

		newName, err := db.GenerateName(ctx, man, projectId, extCertificate.GetName())
		if err != nil {
			return err
		}
		lbcert.Name = newName

		return man.TableSpec().Insert(ctx, &lbcert)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	SyncCloudProject(userCred, &lbcert, projectId, extCertificate, lbcert.ManagerId)

	db.OpsLog.LogEvent(&lbcert, db.ACT_CREATE, lbcert.GetShortDesc(ctx), userCred)

	return &lbcert, nil
}

func (lbcert *SCachedLoadbalancerCertificate) SyncWithCloudLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, extCertificate cloudprovider.ICloudLoadbalancerCertificate, projectId mcclient.IIdentityProvider) error {
	diff, err := db.UpdateWithLock(ctx, lbcert, func() error {
		lbcert.ExternalId = extCertificate.GetGlobalId()
		lbcert.Name = extCertificate.GetName()
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(lbcert, diff, userCred)

	SyncCloudProject(userCred, lbcert, projectId, extCertificate, lbcert.ManagerId)

	return nil
}

func (lbcert *SCachedLoadbalancerCertificate) syncRemoveCloudLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbcert)
	defer lockman.ReleaseObject(ctx, lbcert)

	err := lbcert.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		err = lbcert.SetStatus(userCred, api.LB_STATUS_UNKNOWN, "sync to delete")
	} else {
		err = lbcert.DoPendingDelete(ctx, userCred)
	}
	return err
}

func (man *SCachedLoadbalancerCertificateManager) getLoadbalancerCertificateByRegion(provider *SCloudprovider, regionId string, localCertificateId string) (SCachedLoadbalancerCertificate, error) {
	certificates := []SCachedLoadbalancerCertificate{}
	q := man.Query().Equals("manager_id", provider.Id).Equals("certificate_id", localCertificateId).IsFalse("pending_deleted")
	regionDriver, err := provider.GetRegionDriver()
	if err != nil {
		return SCachedLoadbalancerCertificate{}, errors.Wrap(err, "GetRegionDriver")
	}

	if regionDriver.IsCertificateBelongToRegion() {
		q = q.Equals("cloudregion_id", regionId)
	}

	if err := db.FetchModelObjects(man, q, &certificates); err != nil {
		log.Errorf("failed to get lb certificate for region: %v provider: %v error: %v", regionId, provider, err)
		return SCachedLoadbalancerCertificate{}, err
	}

	if len(certificates) >= 1 {
		return certificates[0], nil
	} else {
		return SCachedLoadbalancerCertificate{}, fmt.Errorf("NotFound")
	}
}

func (man *SCachedLoadbalancerCertificateManager) getLoadbalancerCertificatesByRegion(region *SCloudregion, provider *SCloudprovider) ([]SCachedLoadbalancerCertificate, error) {
	certificates := []SCachedLoadbalancerCertificate{}
	q := man.Query().Equals("manager_id", provider.Id).IsFalse("pending_deleted")
	if region.GetDriver().IsCertificateBelongToRegion() {
		q = q.Equals("cloudregion_id", region.Id)
	}

	if err := db.FetchModelObjects(man, q, &certificates); err != nil {
		log.Errorf("failed to get lb certificates for region: %v provider: %v error: %v", region, provider, err)
		return nil, err
	}
	return certificates, nil
}

func (man *SCachedLoadbalancerCertificateManager) SyncLoadbalancerCertificates(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, certificates []cloudprovider.ICloudLoadbalancerCertificate, syncRange *SSyncRange) compare.SyncResult {
	lockman.LockRawObject(ctx, "certificates", fmt.Sprintf("%s-%s", provider.Id, region.Id))
	defer lockman.ReleaseRawObject(ctx, "certificates", fmt.Sprintf("%s-%s", provider.Id, region.Id))

	syncResult := compare.SyncResult{}

	dbCertificates, err := man.getLoadbalancerCertificatesByRegion(region, provider)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := []SCachedLoadbalancerCertificate{}
	commondb := []SCachedLoadbalancerCertificate{}
	commonext := []cloudprovider.ICloudLoadbalancerCertificate{}
	added := []cloudprovider.ICloudLoadbalancerCertificate{}

	err = compare.CompareSets(dbCertificates, certificates, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].syncRemoveCloudLoadbalancerCertificate(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudLoadbalancerCertificate(ctx, userCred, commonext[i], provider.GetOwnerId())
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		local, err := man.newFromCloudLoadbalancerCertificate(ctx, userCred, provider, added[i], region, provider.GetOwnerId())
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, local, added[i])
			syncResult.Add()
		}
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

	q, err = man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
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

	q, err = man.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
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

	q, err = man.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
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

	q, err = manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemExportKeys")
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
	certs := man.Query().SubQuery()
	sq := certs.Query(certs.Field("id"), certs.Field("external_id"), sqlchemy.COUNT("external_id").Label("total"))
	sq2 := sq.GroupBy("external_id").SubQuery()
	sq3 := sq2.Query(sq2.Field("external_id")).GT("total", 1).SubQuery()

	duplicates := []SCachedLoadbalancerCertificate{}
	q := man.Query().In("external_id", sq3)
	err := db.FetchModelObjects(man, q, &duplicates)
	if err != nil {
		return errors.Wrap(err, "clean duplicated cached loadbalancer certificates")
	}

	for i := range duplicates {
		cache := duplicates[i]
		_, err := db.Update(&cache, func() error {
			cache.MarkDelete()
			return nil
		})
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("clean duplicated cached loadbalancer certificate %s", cache.GetId()))
		}
	}

	log.Infof("%d duplicated cached loadbalancer certificate cleaned", len(duplicates))
	return nil
}
