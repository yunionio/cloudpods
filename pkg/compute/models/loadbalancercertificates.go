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
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
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

type SLoadbalancerCertificateManager struct {
	SLoadbalancerLogSkipper
	db.SVirtualResourceBaseManager
}

var LoadbalancerCertificateManager *SLoadbalancerCertificateManager

func init() {
	LoadbalancerCertificateManager = &SLoadbalancerCertificateManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
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
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase

	Certificate string `create:"required" list:"user" update:"user"`
	PrivateKey  string `create:"required" list:"admin" update:"user"`

	// derived attributes
	PublicKeyAlgorithm      string    `create:"optional" list:"user" update:"user"`
	PublicKeyBitLen         int       `create:"optional" list:"user" update:"user"`
	SignatureAlgorithm      string    `create:"optional" list:"user" update:"user"`
	Fingerprint             string    `create:"optional" list:"user" update:"user"`
	NotBefore               time.Time `create:"optional" list:"user" update:"user"`
	NotAfter                time.Time `create:"optional" list:"user" update:"user"`
	CommonName              string    `create:"optional" list:"user" update:"user"`
	SubjectAlternativeNames string    `create:"optional" list:"user" update:"user"`
}

func (man *SLoadbalancerCertificateManager) pendingDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential, q *sqlchemy.SQuery) {
	subs := []SLoadbalancerCertificate{}
	db.FetchModelObjects(man, q, &subs)
	for _, sub := range subs {
		sub.DoPendingDelete(ctx, userCred)
	}
}

func (man *SLoadbalancerCertificateManager) validateCertKey(ctx context.Context, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	certV := validators.NewCertificateValidator("certificate")
	pkeyV := validators.NewPrivateKeyValidator("private_key")
	keyV := map[string]validators.IValidator{
		"certificate": certV,
		"private_key": pkeyV,
	}
	for _, v := range keyV {
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}
	cert := certV.Certificates[0]
	var certPubKeyAlgo string
	{
		// x509.PublicKeyAlgorithm.String() is only available since go1.10
		switch cert.PublicKeyAlgorithm {
		case x509.RSA:
			certPubKeyAlgo = api.LB_TLS_CERT_PUBKEY_ALGO_RSA
		case x509.ECDSA:
			certPubKeyAlgo = api.LB_TLS_CERT_PUBKEY_ALGO_ECDSA
		default:
			certPubKeyAlgo = fmt.Sprintf("algo %#v", cert.PublicKeyAlgorithm)
		}
		if !api.LB_TLS_CERT_PUBKEY_ALGOS.Has(certPubKeyAlgo) {
			return nil, httperrors.NewInputParameterError("invalid cert pubkey algorithm: %s, want %s",
				certPubKeyAlgo, api.LB_TLS_CERT_PUBKEY_ALGOS.String())
		}
	}
	err := pkeyV.MatchCertificate(cert)
	if err != nil {
		return nil, err
	}
	// NOTE subject alternative names also includes email, url, ip addresses,
	// but we ignore them here.
	//
	// NOTE we use white space to separate names
	data.Set("common_name", jsonutils.NewString(cert.Subject.CommonName))
	data.Set("subject_alternative_names", jsonutils.NewString(strings.Join(cert.DNSNames, " ")))

	data.Set("not_before", jsonutils.NewTimeString(cert.NotBefore))
	data.Set("not_after", jsonutils.NewTimeString(cert.NotAfter))
	data.Set("public_key_algorithm", jsonutils.NewString(certPubKeyAlgo))
	data.Set("public_key_bit_len", jsonutils.NewInt(int64(certV.PublicKeyBitLen())))
	data.Set("signature_algorithm", jsonutils.NewString(cert.SignatureAlgorithm.String()))
	data.Set("fingerprint", jsonutils.NewString(api.LB_TLS_CERT_FINGERPRINT_ALGO_SHA256+":"+certV.FingerprintSha256String()))
	return data, nil
}

func (man *SLoadbalancerCertificateManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	data := query.(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "cloudregion", ModelKeyword: "cloudregion", OwnerId: userCred},
		{Key: "manager", ModelKeyword: "cloudprovider", OwnerId: userCred},
	})
	if err != nil {
		return nil, err
	}

	if jsonutils.QueryBoolean(query, "usable", false) {
		q = q.IsNotEmpty("certificate").IsNotEmpty("private_key")
	}

	return q, nil
}

func (man *SLoadbalancerCertificateManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := man.validateCertKey(ctx, data)
	if err != nil {
		return nil, err
	}
	if _, err := man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data); err != nil {
		return nil, err
	}

	managerIdV := validators.NewModelIdOrNameValidator("manager", "cloudprovider", ownerId)
	managerIdV.Optional(true)
	if err := managerIdV.Validate(data); err != nil {
		return nil, err
	}

	regionV := validators.NewModelIdOrNameValidator("cloudregion", "cloudregion", ownerId)
	regionV.Default("default")
	if err := regionV.Validate(data); err != nil {
		return nil, err
	}
	region := regionV.Model.(*SCloudregion)
	return region.GetDriver().ValidateCreateLoadbalancerCertificateData(ctx, userCred, data)
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
	return nil
}

func (lbcert *SLoadbalancerCertificate) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (lbcert *SLoadbalancerCertificate) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if !data.Contains("certificate") {
		data.Set("certificate", jsonutils.NewString(lbcert.Certificate))
	}
	if !data.Contains("private_key") {
		data.Set("private_key", jsonutils.NewString(lbcert.PrivateKey))
	}
	data, err := LoadbalancerCertificateManager.validateCertKey(ctx, data)
	if err != nil {
		return nil, err
	}
	if _, err := lbcert.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, data); err != nil {
		return nil, err
	}
	region := lbcert.GetRegion()
	if region == nil {
		return nil, httperrors.NewResourceNotFoundError("failed to find region for loadbalancer certificate %s", lbcert.Name)
	}
	return region.GetDriver().ValidateUpdateLoadbalancerCertificateData(ctx, userCred, data)
}

func (lbcert *SLoadbalancerCertificate) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbcert.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	lbcert.SetStatus(userCred, api.LB_CREATING, "")
	if err := lbcert.StartLoadBalancerCertificateCreateTask(ctx, userCred, ""); err != nil {
		log.Errorf("Failed to create loadbalancercertificate error: %v", err)
	}
}

func (lbcert *SLoadbalancerCertificate) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := lbcert.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	providerInfo := lbcert.SManagedResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if providerInfo != nil {
		extra.Update(providerInfo)
	}
	regionInfo := lbcert.SCloudregionResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if regionInfo != nil {
		extra.Update(regionInfo)
	}
	return extra
}

func (lbcert *SLoadbalancerCertificate) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra := lbcert.GetCustomizeColumns(ctx, userCred, query)
	return extra, nil
}

func (lbcert *SLoadbalancerCertificate) StartLoadBalancerCertificateCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerCertificateCreateTask", lbcert, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
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
	return nil
}

func (lbcert *SLoadbalancerCertificate) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (lbcert *SLoadbalancerCertificate) GetRegion() *SCloudregion {
	region, err := CloudregionManager.FetchById(lbcert.CloudregionId)
	if err != nil {
		log.Errorf("failed to find region for loadbalancer certificate %s", lbcert.Name)
		return nil
	}
	return region.(*SCloudregion)
}

func (lbcert *SLoadbalancerCertificate) GetIRegion() (cloudprovider.ICloudRegion, error) {
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

func (lbcert *SLoadbalancerCertificate) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, lbcert, "purge")
}

func (lbcert *SLoadbalancerCertificate) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	parasm := jsonutils.NewDict()
	parasm.Add(jsonutils.JSONTrue, "purge")
	return nil, lbcert.StartLoadBalancerCertificateDeleteTask(ctx, userCred, parasm, "")
}

func (lbcert *SLoadbalancerCertificate) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	lbcert.SetStatus(userCred, api.LB_STATUS_DELETING, "")
	return lbcert.StartLoadBalancerCertificateDeleteTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (lbcert *SLoadbalancerCertificate) StartLoadBalancerCertificateDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerCertificateDeleteTask", lbcert, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (man *SLoadbalancerCertificateManager) getLoadbalancerCertificatesByRegion(region *SCloudregion, provider *SCloudprovider) ([]SLoadbalancerCertificate, error) {
	certificates := []SLoadbalancerCertificate{}
	q := man.Query().Equals("cloudregion_id", region.Id).Equals("manager_id", provider.Id).IsFalse("pending_deleted")
	if err := db.FetchModelObjects(man, q, &certificates); err != nil {
		log.Errorf("failed to get lb certificates for region: %v provider: %v error: %v", region, provider, err)
		return nil, err
	}
	return certificates, nil
}

func (man *SLoadbalancerCertificateManager) SyncLoadbalancerCertificates(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, certificates []cloudprovider.ICloudLoadbalancerCertificate, syncRange *SSyncRange) compare.SyncResult {
	syncOwnerId := provider.GetOwnerId()

	lockman.LockClass(ctx, man, db.GetLockClassKey(man, syncOwnerId))
	defer lockman.ReleaseClass(ctx, man, db.GetLockClassKey(man, syncOwnerId))

	syncResult := compare.SyncResult{}

	dbCertificates, err := man.getLoadbalancerCertificatesByRegion(region, provider)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := []SLoadbalancerCertificate{}
	commondb := []SLoadbalancerCertificate{}
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
		err = commondb[i].SyncWithCloudLoadbalancerCertificate(ctx, userCred, commonext[i], syncOwnerId)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		local, err := man.newFromCloudLoadbalancerCertificate(ctx, userCred, provider, added[i], region, syncOwnerId)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, local, added[i])
			syncResult.Add()
		}
	}
	return syncResult
}

func (man *SLoadbalancerCertificateManager) newFromCloudLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extCertificate cloudprovider.ICloudLoadbalancerCertificate, region *SCloudregion, syncOwnerId mcclient.IIdentityProvider) (*SLoadbalancerCertificate, error) {
	lbcert := SLoadbalancerCertificate{}
	lbcert.SetModelManager(man, &lbcert)

	newName, err := db.GenerateName(man, syncOwnerId, extCertificate.GetName())
	if err != nil {
		return nil, err
	}
	lbcert.Name = newName
	lbcert.ExternalId = extCertificate.GetGlobalId()
	lbcert.ManagerId = provider.Id
	lbcert.CloudregionId = region.Id

	lbcert.CommonName = extCertificate.GetCommonName()
	lbcert.SubjectAlternativeNames = extCertificate.GetSubjectAlternativeNames()
	lbcert.Fingerprint = extCertificate.GetFingerprint()
	lbcert.NotAfter = extCertificate.GetExpireTime()

	err = man.TableSpec().Insert(&lbcert)
	if err != nil {
		log.Errorf("newFromCloudLoadbalancerCertificate fail %s", err)
		return nil, err
	}

	SyncCloudProject(userCred, &lbcert, syncOwnerId, extCertificate, lbcert.ManagerId)

	db.OpsLog.LogEvent(&lbcert, db.ACT_CREATE, lbcert.GetShortDesc(ctx), userCred)

	return &lbcert, nil
}

func (lbcert *SLoadbalancerCertificate) syncRemoveCloudLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential) error {
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

func (lbcert *SLoadbalancerCertificate) SyncWithCloudLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, extCertificate cloudprovider.ICloudLoadbalancerCertificate, syncOwnerId mcclient.IIdentityProvider) error {
	diff, err := db.UpdateWithLock(ctx, lbcert, func() error {
		lbcert.Name = extCertificate.GetName()
		lbcert.CommonName = extCertificate.GetCommonName()
		lbcert.SubjectAlternativeNames = extCertificate.GetSubjectAlternativeNames()
		lbcert.Fingerprint = extCertificate.GetFingerprint()
		lbcert.NotAfter = extCertificate.GetExpireTime()
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(lbcert, diff, userCred)

	SyncCloudProject(userCred, lbcert, syncOwnerId, extCertificate, lbcert.ManagerId)

	return nil
}

func (manager *SLoadbalancerCertificateManager) GetResourceCount() ([]db.SProjectResourceCount, error) {
	virts := manager.Query().IsFalse("pending_deleted")
	return db.CalculateProjectResourceCount(virts)
}
