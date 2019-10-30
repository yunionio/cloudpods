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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCachedLoadbalancerCertificateManager struct {
	SLoadbalancerLogSkipper
	db.SVirtualResourceBaseManager
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

	CertificateId string `width:"128" charset:"ascii" nullable:"false" create:"required"  index:"true" list:"user"` // 本地证书ID
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
	lbcertId := self.Id
	for _, man := range men {
		t := man.TableSpec().Instance()
		pdF := t.Field("pending_deleted")
		n, err := t.Query().
			Equals("certificate_id", lbcertId).
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
	name, _ := db.GenerateName(man, ownerId, certificateV.Model.GetName())
	data.Set("name", jsonutils.NewString(name))
	if _, err := man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data); err != nil {
		return nil, err
	}

	return data, nil
}

func (self *SCachedLoadbalancerCertificate) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SetStatus(userCred, api.LB_CREATING, "")
	if err := self.StartLoadbalancerCertificateCreateTask(ctx, userCred, ""); err != nil {
		log.Errorf("CachedLoadbalancerCertificate.PostCreate %s", err)
	}
	return
}

func (self *SCachedLoadbalancerCertificate) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	providerInfo := self.SManagedResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if providerInfo != nil {
		extra.Update(providerInfo)
	}
	regionInfo := self.SCloudregionResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if regionInfo != nil {
		extra.Update(regionInfo)
	}
	return extra
}

func (self *SCachedLoadbalancerCertificate) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra := self.GetCustomizeColumns(ctx, userCred, query)
	return extra, nil
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

	lbcert, err := man.getLoadbalancerCertificateByRegion(provider, lblis.CloudregionId, cert.Id)
	if err == nil {
		return &lbcert, nil
	}

	if err.Error() != "NotFound" {
		return nil, errors.Wrap(err, "cachedLoadbalancerCertificateManager.getCert")
	}

	lbcert = SCachedLoadbalancerCertificate{}
	lbcert.ManagerId = lblis.ManagerId
	lbcert.CloudregionId = lblis.CloudregionId
	lbcert.ProjectId = lblis.ProjectId
	lbcert.ProjectSrc = lblis.ProjectSrc
	lbcert.Name = cert.Name
	lbcert.Description = cert.Description
	lbcert.IsSystem = cert.IsSystem
	lbcert.CertificateId = cert.Id

	err = man.TableSpec().Insert(&lbcert)
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

	newName, err := db.GenerateName(man, projectId, extCertificate.GetName())
	if err != nil {
		return nil, err
	}
	lbcert.Name = newName
	lbcert.ExternalId = extCertificate.GetGlobalId()
	lbcert.ManagerId = provider.Id
	lbcert.CloudregionId = region.Id

	c := SLoadbalancerCertificate{}
	q1 := LoadbalancerCertificateManager.Query().IsFalse("pending_deleted").Equals("fingerprint", extCertificate.GetFingerprint())
	err = q1.First(&c)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			localcert, err := LoadbalancerCertificateManager.CreateCertificate(userCred, lbcert.Name, extCertificate.GetPublickKey(), extCertificate.GetPrivateKey(), extCertificate.GetFingerprint())
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

	err = man.TableSpec().Insert(&lbcert)
	if err != nil {
		log.Errorf("newFromCloudLoadbalancerCertificate fail %s", err)
		return nil, err
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
	// aws 所有region共用一份证书.与region无关
	if provider.GetName() != api.CLOUD_PROVIDER_AWS {
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
	// aws 所有region共用一份证书.与region无关
	if region.GetProviderName() != api.CLOUD_PROVIDER_AWS {
		q = q.Equals("cloudregion_id", region.Id)
	}

	if err := db.FetchModelObjects(man, q, &certificates); err != nil {
		log.Errorf("failed to get lb certificates for region: %v provider: %v error: %v", region, provider, err)
		return nil, err
	}
	return certificates, nil
}

func (man *SCachedLoadbalancerCertificateManager) SyncLoadbalancerCertificates(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, certificates []cloudprovider.ICloudLoadbalancerCertificate, syncRange *SSyncRange) compare.SyncResult {
	ownerProjId := provider.ProjectId

	lockman.LockClass(ctx, man, ownerProjId)
	defer lockman.ReleaseClass(ctx, man, ownerProjId)

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
