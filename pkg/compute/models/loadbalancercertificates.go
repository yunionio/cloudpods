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
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"encoding/pem"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SLoadbalancerCertificateManager struct {
	SLoadbalancerLogSkipper
	db.SSharableVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
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

// TODO
//
//  - notify users of cert expiration
//  - ca info: self-signed, public ca
type SLoadbalancerCertificate struct {
	db.SSharableVirtualResourceBase
	db.SExternalizedResourceBase

	// SManagedResourceBase
	// SCloudregionResourceBase

	db.SCertificateResourceBase
}

func (lbcert *SLoadbalancerCertificate) GetCachedCerts() ([]SCachedLoadbalancerCertificate, error) {
	ret := []SCachedLoadbalancerCertificate{}
	q := CachedLoadbalancerCertificateManager.Query().Equals("certificate_id", lbcert.Id).IsFalse("pending_deleted")
	err := db.FetchModelObjects(CachedLoadbalancerCertificateManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (lbcert *SLoadbalancerCertificate) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (lbcert *SLoadbalancerCertificate) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if data.Contains("certificate") || data.Contains("private_key") {
		return nil, httperrors.NewForbiddenError("not allowed update content of certificate")
	}

	updateData := jsonutils.NewDict()
	if name, err := data.GetString("name"); err == nil {
		updateData.Set("name", jsonutils.NewString(name))
	}

	if desc, err := data.GetString("description"); err == nil {
		updateData.Set("description", jsonutils.NewString(desc))
	}

	input := apis.SharableVirtualResourceBaseUpdateInput{}
	err := updateData.Unmarshal(&input)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	input, err = lbcert.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBase.ValidateUpdateData")
	}
	updateData.Update(jsonutils.Marshal(input))

	return updateData, nil
}

func (lbcert *SLoadbalancerCertificate) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbcert.SSharableVirtualResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)
	lbcert.SetStatus(userCred, api.LB_STATUS_ENABLED, "")
}

func (lbcert *SLoadbalancerCertificate) IsComplete() bool {
	return lbcert.PrivateKey != "" && lbcert.Certificate != ""
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

	for i := range rows {
		rows[i] = api.LoadbalancerCertificateDetails{
			SharableVirtualResourceDetails: virtRows[i],
			IsComplete:                     objs[i].(*SLoadbalancerCertificate).IsComplete(),
		}
	}

	for i := range objs {
		q := LoadbalancerListenerManager.Query().IsFalse("pending_deleted").Equals("certificate_id", objs[i].(*SLoadbalancerCertificate).GetId())
		ownerId, queryScope, err := db.FetchCheckQueryOwnerScope(ctx, userCred, query, LoadbalancerListenerManager, policy.PolicyActionList, true)
		if err != nil {
			log.Errorf("FetchCheckQueryOwnerScope error: %v", err)
			return rows
		}

		q = LoadbalancerListenerManager.FilterByOwner(q, ownerId, queryScope)
		count, err := q.CountWithError()
		if err != nil {
			log.Errorf("db.CountWithError error: %v", err)
		}

		rows[i].LbListenerCount = count
	}

	return rows
}

func (lbcert *SLoadbalancerCertificate) ValidateDeleteCondition(ctx context.Context) error {
	men := []db.IModelManager{
		LoadbalancerListenerManager,
	}
	lbcertId := lbcert.Id
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

	// cache delete condition
	caches, err := lbcert.GetCachedCerts()
	if err != nil {
		return errors.Wrap(err, "GetCachedCerts")
	}

	for i := range caches {
		err := caches[i].ValidateDeleteCondition(ctx)
		if err != nil {
			return errors.Wrap(err, "ValidateDeleteCondition")
		}
	}

	return nil
}

func (lbcert *SLoadbalancerCertificate) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, lbcert, "purge")
}

func (lbcert *SLoadbalancerCertificate) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, lbcert.CustomizeDelete(ctx, userCred, query, data)
}

func (lbcert *SLoadbalancerCertificate) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	if !lbcert.PendingDeleted {
		caches, err := lbcert.GetCachedCerts()
		if err != nil {
			return errors.Wrap(err, "GetCachedCerts")
		}

		for i := range caches {
			err := caches[i].MarkPendingDelete(userCred)
			if err != nil {
				return errors.Wrap(err, "MarkPendingDelete")
			}
		}

		return lbcert.DoPendingDelete(ctx, userCred)
	}

	return nil
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

	if query.Usable != nil && *query.Usable {
		region := query.CloudregionId
		manager := query.CloudproviderId

		// 证书可用包含两类：1.本地证书内容不为空 2.公有云中已经存在，但是证书内容不完整的证书
		if len(region) > 0 || len(manager) > 0 {
			q2 := CachedLoadbalancerCertificateManager.Query("certificate_id").IsFalse("pending_deleted")
			if len(region) > 0 {
				q2 = q2.Equals("cloudregion_id", region)
			}

			if len(manager) > 0 {
				q2 = q2.Equals("manager_id", manager)
			}

			count, err := q2.CountWithError()
			if err != nil && err != sql.ErrNoRows {
				return nil, err
			}

			if count > 0 {
				conditionA := sqlchemy.AND(sqlchemy.IsNotEmpty(q.Field("certificate")), sqlchemy.IsNotEmpty(q.Field("private_key")))
				conditionB := sqlchemy.In(q.Field("id"), q2.SubQuery())
				q = q.Filter(sqlchemy.OR(conditionA, conditionB))
			} else {
				q = q.IsNotEmpty("certificate").IsNotEmpty("private_key")
			}
		} else {
			q = q.IsNotEmpty("certificate").IsNotEmpty("private_key")
		}
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

	return q, nil
}

func (man *SLoadbalancerCertificateManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SSharableVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (man *SLoadbalancerCertificateManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	v := validators.NewCertKeyValidator("certificate", "private_key")
	if err := v.Validate(data); err != nil {
		return nil, err
	}
	data = v.UpdateCertKeyInfo(ctx, data)

	input := apis.SharableVirtualResourceCreateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal VirtualResourceCreateInput fail %s", err)
	}
	input, err = man.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))

	data.Remove("cloudregion_id")
	data.Remove("manager_id")
	return data, nil
}

func (lbcert *SLoadbalancerCertificate) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if !data.Contains("public_scope") {
		if db.IsAdminAllowPerform(userCred, lbcert, "public") && ownerId.GetProjectDomainId() == userCred.GetProjectDomainId() {
			lbcert.SetShare(rbacutils.ScopeSystem)
		} else if db.IsDomainAllowPerform(userCred, lbcert, "public") && ownerId.GetProjectId() == userCred.GetProjectId() && consts.GetNonDefaultDomainProjects() {
			// only if non_default_domain_projects turned on, share to domain
			lbcert.SetShare(rbacutils.ScopeDomain)
		} else {
			lbcert.SetShare(rbacutils.ScopeNone)
		}

		data.(*jsonutils.JSONDict).Set("public_scope", jsonutils.NewString(lbcert.PublicScope))
	}

	return lbcert.SSharableVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (man *SLoadbalancerCertificateManager) InitializeData() error {
	// initialize newly added null certificate fingerprint column
	q := man.Query().IsNull("fingerprint")
	lbcerts := []SLoadbalancerCertificate{}
	if err := q.All(&lbcerts); err != nil {
		return err
	}
	for i := range lbcerts {
		lbcert := &lbcerts[i]
		fp := lbcert.Fingerprint
		if fp != "" {
			continue
		}
		if lbcert.Certificate == "" {
			continue
		}
		{
			p, _ := pem.Decode([]byte(lbcert.Certificate))
			c, err := x509.ParseCertificate(p.Bytes)
			if err != nil {
				log.Errorf("parsing certificate %s(%s): %s", lbcert.Name, lbcert.Id, err)
				continue
			}
			d := sha256.Sum256(c.Raw)
			fp = api.LB_TLS_CERT_FINGERPRINT_ALGO_SHA256 + ":" + hex.EncodeToString(d[:])
		}
		_, err := db.Update(lbcert, func() error {
			lbcert.Fingerprint = fp
			return nil
		})
		if err != nil {
			return err
		}
	}

	// sync certificate to  certificate cache
	/*lbcerts = []SLoadbalancerCertificate{}
	cachedCerts := CachedLoadbalancerCertificateManager.Query("certificate_id").SubQuery()
	q2 := man.Query().IsNotEmpty("external_id").IsNotEmpty("cloudregion_id").NotIn("id", cachedCerts)
	if err := q2.All(&lbcerts); err != nil {
		return err
	}

	for i := range lbcerts {
		cert := lbcerts[i]
		certObj := jsonutils.Marshal(cert)
		cachedCert := &SCachedLoadbalancerCertificate{}
		err := certObj.Unmarshal(cachedCert)
		if err != nil {
			return err
		}
		cachedCert.Id = ""
		cachedCert.CertificateId = cert.Id
		err = CachedLoadbalancerCertificateManager.TableSpec().Insert(cachedCert)
		if err != nil {
			return err
		}
	}*/

	return nil
}

func (man *SLoadbalancerCertificateManager) CreateCertificate(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, name string, extCert cloudprovider.ICloudLoadbalancerCertificate) (*SLoadbalancerCertificate, error) {
	fingerprint := extCert.GetFingerprint()
	if len(fingerprint) == 0 {
		return nil, fmt.Errorf("CreateCertificate fingerprint can not be empty")
	}

	data := jsonutils.NewDict()
	data.Set("certificate", jsonutils.NewString(extCert.GetPublickKey()))
	data.Set("private_key", jsonutils.NewString(extCert.GetPrivateKey()))
	data.Set("name", jsonutils.NewString(name))
	data.Set("fingerprint", jsonutils.NewString(fingerprint))
	q := man.Query().Equals("fingerprint", fingerprint).Asc("created_at").IsFalse("pending_deleted")
	count, err := q.CountWithError()
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if count == 0 {
		cert := &SLoadbalancerCertificate{}
		cert.SetModelManager(man, cert)
		err := data.Unmarshal(cert)
		if err != nil {
			return nil, err
		}

		// other information's
		cert.CommonName = extCert.GetCommonName()
		cert.SubjectAlternativeNames = extCert.GetSubjectAlternativeNames()
		cert.NotAfter = extCert.GetExpireTime()
		cert.PublicScope = string(rbacutils.ScopeDomain)
		cert.IsPublic = true

		err = man.TableSpec().Insert(ctx, cert)
		if err != nil {
			return nil, err
		}

		SyncCloudProject(userCred, cert, provider.GetOwnerId(), extCert, provider.GetId())
	}

	ret := &SLoadbalancerCertificate{}
	err = q.First(ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (manager *SLoadbalancerCertificateManager) GetResourceCount() ([]db.SScopeResourceCount, error) {
	virts := manager.Query().IsFalse("pending_deleted")
	return db.CalculateResourceCount(virts, "tenant_id")
}

func (manager *SLoadbalancerCertificateManager) GetLbCertByFingerprint(tenantId, fingerprint string) (*SLoadbalancerCertificate, error) {
	c := SLoadbalancerCertificate{}
	c.SetModelManager(manager, &c)
	q1 := LoadbalancerCertificateManager.Query().IsFalse("pending_deleted")
	q1 = q1.Equals("fingerprint", fingerprint)
	q1 = q1.Equals("tenant_id", tenantId)
	if err := q1.First(&c); err != nil {
		return nil, err
	}

	return &c, nil
}
