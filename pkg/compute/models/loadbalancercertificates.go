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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
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

func (lbcert *SLoadbalancerCertificate) GetCachedCerts() ([]SCachedLoadbalancerCertificate, error) {
	ret := []SCachedLoadbalancerCertificate{}
	q := CachedLoadbalancerCertificateManager.Query().Equals("certificate_id", lbcert.Id)
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

	if _, err := lbcert.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, updateData); err != nil {
		return nil, err
	}

	return updateData, nil
}

func (lbcert *SLoadbalancerCertificate) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbcert.SVirtualResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)
	lbcert.SetStatus(userCred, api.LB_STATUS_ENABLED, "")
}

func (lbcert *SLoadbalancerCertificate) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := lbcert.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return extra
}

func (lbcert *SLoadbalancerCertificate) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra := lbcert.GetCustomizeColumns(ctx, userCred, query)
	return extra, nil
}

func (lbcert *SLoadbalancerCertificate) ValidateDeleteCondition(ctx context.Context) error {
	men := []db.IModelManager{
		CachedLoadbalancerCertificateManager,
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

func (lbcert *SLoadbalancerCertificate) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, lbcert, "purge")
}

func (lbcert *SLoadbalancerCertificate) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, lbcert.CustomizeDelete(ctx, userCred, query, data)
}

func (lbcert *SLoadbalancerCertificate) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	if !lbcert.PendingDeleted {
		return lbcert.DoPendingDelete(ctx, userCred)
	}

	return nil
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
	if jsonutils.QueryBoolean(query, "usable", false) {
		region, _ := data.GetString("cloudregion")
		manager, _ := data.GetString("manager")

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

	data.Remove("cloudregion_id")
	data.Remove("manager_id")
	return data, nil
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
	lbcerts = []SLoadbalancerCertificate{}
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
	}

	return nil
}

func (man *SLoadbalancerCertificateManager) CreateCertificate(userCred mcclient.TokenCredential, name string, publicKey string, privateKey, fingerprint string) (*SLoadbalancerCertificate, error) {
	if len(fingerprint) == 0 {
		return nil, fmt.Errorf("CreateCertificate fingerprint can not be empty")
	}

	data := jsonutils.NewDict()
	data.Set("certificate", jsonutils.NewString(publicKey))
	data.Set("private_key", jsonutils.NewString(privateKey))
	data.Set("name", jsonutils.NewString(name))
	data.Set("fingerprint", jsonutils.NewString(fingerprint))
	q := man.Query().Equals("fingerprint", fingerprint).Asc("created_at").IsFalse("pending_deleted")
	count, err := q.CountWithError()
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if count == 0 {
		cert := &SLoadbalancerCertificate{}
		err := data.Unmarshal(cert)
		if err != nil {
			return nil, err
		}

		// usercred
		cert.DomainId = userCred.GetProjectDomainId()
		cert.ProjectId = userCred.GetProjectId()
		cert.ProjectSrc = string(db.PROJECT_SOURCE_CLOUD)

		err = man.TableSpec().Insert(cert)
		if err != nil {
			return nil, err
		}
	}

	ret := &SLoadbalancerCertificate{}
	err = q.First(ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (manager *SLoadbalancerCertificateManager) GetResourceCount() ([]db.SProjectResourceCount, error) {
	virts := manager.Query().IsFalse("pending_deleted")
	return db.CalculateProjectResourceCount(virts)
}
