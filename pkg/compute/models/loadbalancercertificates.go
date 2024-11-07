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
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strings"

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
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=loadbalancercertificate
// +onecloud:swagger-gen-model-plural=loadbalancercertificates
type SLoadbalancerCertificateManager struct {
	SLoadbalancerLogSkipper
	db.SSharableVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager

	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
}

var LoadbalancerCertificateManager *SLoadbalancerCertificateManager

func init() {
	LoadbalancerCertificateManager = &SLoadbalancerCertificateManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SLoadbalancerCertificate{},
			"loadbalancercertificates_tbl",
			"loadbalancercertificate",
			"loadbalancercertificates",
		),
	}
	LoadbalancerCertificateManager.SetVirtualObject(LoadbalancerCertificateManager)
}

type SLoadbalancerCertificate struct {
	db.SSharableVirtualResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase

	db.SCertificateResourceBase
}

func (lbcert *SLoadbalancerCertificate) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.LoadbalancerCertificateUpdateInput) (*api.LoadbalancerCertificateUpdateInput, error) {
	var err error
	input.SharableVirtualResourceBaseUpdateInput, err = lbcert.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.SharableVirtualResourceBaseUpdateInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (nm *SLoadbalancerCertificateManager) query(manager db.IModelManager, field string, certIds []string, filter func(*sqlchemy.SQuery) *sqlchemy.SQuery) *sqlchemy.SSubQuery {
	q := manager.Query()

	if filter != nil {
		q = filter(q)
	}

	sq := q.SubQuery()

	return sq.Query(
		sq.Field("certificate_id"),
		sqlchemy.COUNT(field),
	).In("certificate_id", certIds).GroupBy(sq.Field("certificate_id")).SubQuery()
}

type SCertUsageCount struct {
	Id string
	api.LoadbalancerCertificateUsage
}

func (manager *SLoadbalancerCertificateManager) TotalResourceCount(certIds []string) (map[string]api.LoadbalancerCertificateUsage, error) {
	// listener
	listenerSQ := manager.query(LoadbalancerListenerManager, "listener_cnt", certIds, nil)

	certs := manager.Query().SubQuery()
	certQ := certs.Query(
		sqlchemy.SUM("lb_listener_count", listenerSQ.Field("listener_cnt")),
	)

	certQ.AppendField(certQ.Field("id"))

	certQ = certQ.LeftJoin(listenerSQ, sqlchemy.Equals(certQ.Field("id"), listenerSQ.Field("certificate_id")))
	certQ = certQ.Filter(sqlchemy.In(certQ.Field("id"), certIds)).GroupBy(certQ.Field("id"))

	certCount := []SCertUsageCount{}
	err := certQ.All(&certCount)
	if err != nil {
		return nil, errors.Wrapf(err, "certQ.All")
	}

	result := map[string]api.LoadbalancerCertificateUsage{}
	for i := range certCount {
		result[certCount[i].Id] = certCount[i].LoadbalancerCertificateUsage
	}

	return result, nil
}

func (manager *SLoadbalancerCertificateManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LoadbalancerCertificateDetails {
	rows := make([]api.LoadbalancerCertificateDetails, len(objs))

	virtRows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	certIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.LoadbalancerCertificateDetails{
			SharableVirtualResourceDetails: virtRows[i],
			ManagedResourceInfo:            managerRows[i],
			CloudregionResourceInfo:        regionRows[i],
		}
	}

	usage, err := manager.TotalResourceCount(certIds)
	if err != nil {
		log.Errorf("TotalResourceCount error: %v", err)
		return rows
	}

	for i := range rows {
		rows[i].LoadbalancerCertificateUsage, _ = usage[certIds[i]]
	}

	return rows
}

func (lbcert *SLoadbalancerCertificate) ValidateDeleteCondition(ctx context.Context, info *api.LoadbalancerCertificateDetails) error {
	if info != nil && info.ListenerCount > 0 {
		return httperrors.NewNotEmptyError("cert %s with %d listeners", lbcert.Name, info.ListenerCount)
	}
	return lbcert.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx, jsonutils.Marshal(info))
}

func (lbcert *SLoadbalancerCertificate) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (lbcert *SLoadbalancerCertificate) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return lbcert.SSharableVirtualResourceBase.Delete(ctx, userCred)
}

func (self *SLoadbalancerCertificate) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteTask(ctx, userCred, "")
}

func (lbcert *SLoadbalancerCertificate) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerCertificateDeleteTask", lbcert, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	lbcert.SetStatus(ctx, userCred, apis.STATUS_DELETING, "")
	return task.ScheduleRun(nil)
}

func (man *SLoadbalancerCertificateManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerCertificateListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = man.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = man.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	q, err = man.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}

	if query.Usable != nil && *query.Usable {
		q = q.Filter(
			sqlchemy.OR(
				sqlchemy.AND(
					sqlchemy.Equals(q.Field("cloudregion_id"), api.DEFAULT_REGION_ID),
					sqlchemy.IsNotEmpty(q.Field("certificate")),
					sqlchemy.IsNotEmpty(q.Field("private_key")),
				),
				sqlchemy.AND(
					sqlchemy.NotEquals(q.Field("cloudregion_id"), api.DEFAULT_REGION_ID),
					sqlchemy.IsNotEmpty(q.Field("external_id")),
				),
			),
		)
	}

	if len(query.CommonName) > 0 {
		q = q.In("common_name", query.CommonName)
	}
	if len(query.SubjectAlternativeNames) > 0 {
		q = q.In("subject_alternative_names", query.SubjectAlternativeNames)
	}

	return q, nil
}

func (man *SLoadbalancerCertificateManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerCertificateListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SSharableVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.OrderByExtraFields")
	}

	q, err = man.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (man *SLoadbalancerCertificateManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SSharableVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
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

	return q, httperrors.ErrNotFound
}

func (manager *SLoadbalancerCertificateManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemExportKeys")
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

	return q, nil
}

func (self *SLoadbalancerCertificate) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
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

func (self *SLoadbalancerCertificate) GetILoadbalancerCertificate(ctx context.Context) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	iRegion, err := self.GetIRegion(ctx)
	if err != nil {
		return nil, err
	}
	return iRegion.GetILoadBalancerCertificateById(self.ExternalId)
}

func (lbcert *SLoadbalancerCertificate) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, StartResourceSyncStatusTask(ctx, userCred, lbcert, "LoadbalancerCertificateSyncstatusTask", "")
}

func (man *SLoadbalancerCertificateManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input *api.LoadbalancerCertificateCreateInput,
) (*api.LoadbalancerCertificateCreateInput, error) {
	if len(input.Certificate) == 0 {
		return nil, httperrors.NewMissingParameterError("certificate")
	}
	if len(input.PrivateKey) == 0 {
		return nil, httperrors.NewMissingParameterError("private_key")
	}
	_, err := tls.X509KeyPair([]byte(input.Certificate), []byte(input.PrivateKey))
	if err != nil {
		return nil, err
	}
	p, _ := pem.Decode([]byte(input.Certificate))
	c, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		return nil, err
	}
	input.SubjectAlternativeNames = strings.Join(c.DNSNames, " ")
	input.SignatureAlgorithm = c.SignatureAlgorithm.String()
	d := sha256.Sum256(c.Raw)
	input.Fingerprint = api.LB_TLS_CERT_FINGERPRINT_ALGO_SHA256 + ":" + hex.EncodeToString(d[:])
	input.CommonName = c.Subject.CommonName
	input.NotBefore = c.NotBefore
	input.NotAfter = c.NotAfter
	switch pub := c.PublicKey.(type) {
	case *rsa.PublicKey:
		input.PublicKeyBitLen = pub.N.BitLen()
	case *ecdsa.PublicKey:
		input.PublicKeyBitLen = pub.X.BitLen()
	}
	input.SharableVirtualResourceCreateInput, err = man.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.Status = apis.STATUS_CREATING

	if len(input.CloudregionId) == 0 {
		input.CloudregionId = api.DEFAULT_REGION_ID
	}
	regionObj, err := validators.ValidateModel(ctx, userCred, CloudregionManager, &input.CloudregionId)
	if err != nil {
		return nil, err
	}
	region := regionObj.(*SCloudregion)
	if len(input.CloudproviderId) > 0 {
		providerObj, err := validators.ValidateModel(ctx, userCred, CloudproviderManager, &input.CloudproviderId)
		if err != nil {
			return nil, err
		}
		input.ManagerId = input.CloudproviderId
		provider := providerObj.(*SCloudprovider)
		if provider.Provider != region.Provider {
			return nil, httperrors.NewConflictError("conflict region %s and cloudprovider %s", region.Name, provider.Name)
		}
	}

	return input, nil
}

func (self *SLoadbalancerCertificate) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SSharableVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	self.StartCreateTask(ctx, userCred, "")
}

func (lbcert *SLoadbalancerCertificate) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerCertificateCreateTask", lbcert, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (self *SCloudregion) GetLoadbalancerCertificates(managerId string) ([]SLoadbalancerCertificate, error) {
	q := LoadbalancerCertificateManager.Query().Equals("cloudregion_id", self.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	ret := []SLoadbalancerCertificate{}
	err := db.FetchModelObjects(LoadbalancerCertificateManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SCloudregion) SyncLoadbalancerCertificates(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, exts []cloudprovider.ICloudLoadbalancerCertificate, xor bool) compare.SyncResult {
	lockman.LockRawObject(ctx, LoadbalancerCertificateManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, provider.Id))
	defer lockman.ReleaseRawObject(ctx, LoadbalancerCertificateManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, provider.Id))

	result := compare.SyncResult{}

	dbCerts, err := self.GetLoadbalancerCertificates(provider.Id)
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SLoadbalancerCertificate, 0)
	commondb := make([]SLoadbalancerCertificate, 0)
	commonext := make([]cloudprovider.ICloudLoadbalancerCertificate, 0)
	added := make([]cloudprovider.ICloudLoadbalancerCertificate, 0)

	err = compare.CompareSets(dbCerts, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].RealDelete(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	for i := 0; i < len(commondb); i += 1 {
		if !xor {
			err = commondb[i].SyncWithCloudCert(ctx, userCred, commonext[i], provider)
			if err != nil {
				result.UpdateError(err)
				continue
			}
		}
		result.Update()
	}

	for i := 0; i < len(added); i += 1 {
		err := self.newFromCloudCert(ctx, userCred, provider, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func (lbcert *SLoadbalancerCertificate) SyncWithCloudCert(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudLoadbalancerCertificate, provider *SCloudprovider) error {
	_, err := db.Update(lbcert, func() error {
		lbcert.Name = ext.GetName()
		lbcert.CommonName = ext.GetCommonName()
		lbcert.SubjectAlternativeNames = ext.GetSubjectAlternativeNames()
		lbcert.Fingerprint = ext.GetFingerprint()
		lbcert.NotAfter = ext.GetExpireTime()
		lbcert.Status = ext.GetStatus()
		if key := ext.GetPublickKey(); len(key) > 0 {
			lbcert.Certificate = key
		}
		if key := ext.GetPrivateKey(); len(key) > 0 {
			lbcert.PrivateKey = key
		}
		return nil
	})
	if err != nil {
		return err
	}

	syncVirtualResourceMetadata(ctx, userCred, lbcert, ext, false)
	SyncCloudProject(ctx, userCred, lbcert, provider.GetOwnerId(), ext, provider)

	return nil
}

func (self *SCloudregion) newFromCloudCert(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudLoadbalancerCertificate) error {
	ret := &SLoadbalancerCertificate{}
	ret.SetModelManager(LoadbalancerCertificateManager, ret)
	ret.ExternalId = ext.GetGlobalId()
	ret.CloudregionId = self.Id
	ret.ManagerId = provider.Id
	ret.Name = ext.GetName()
	ret.Status = ext.GetStatus()
	ret.CommonName = ext.GetCommonName()
	ret.SubjectAlternativeNames = ext.GetSubjectAlternativeNames()
	ret.Fingerprint = ext.GetFingerprint()
	ret.NotAfter = ext.GetExpireTime()
	ret.Certificate = ext.GetPublickKey()
	ret.PrivateKey = ext.GetPrivateKey()

	err := LoadbalancerCertificateManager.TableSpec().Insert(ctx, ret)
	if err != nil {
		return errors.Wrapf(err, "Insert")
	}

	syncVirtualResourceMetadata(ctx, userCred, ret, ext, false)
	SyncCloudProject(ctx, userCred, ret, provider.GetOwnerId(), ext, provider)

	return nil
}

func (man *SLoadbalancerCertificateManager) InitializeData() error {
	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"update %s set deleted = true where pending_deleted = true",
			man.TableSpec().Name(),
		),
	)
	return err
}
