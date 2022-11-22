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
	"database/sql"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
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
//   - notify users of cert expiration
//   - ca info: self-signed, public ca
type SLoadbalancerCertificate struct {
	db.SSharableVirtualResourceBase
	db.SExternalizedResourceBase

	db.SCertificateResourceBase
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

func (lbcert *SLoadbalancerCertificate) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.LoadbalancerCertificateUpdateInput) (*api.LoadbalancerCertificateUpdateInput, error) {
	var err error
	input.SharableVirtualResourceBaseUpdateInput, err = lbcert.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.SharableVirtualResourceBaseUpdateInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBase.ValidateUpdateData")
	}
	return input, nil
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
		q := LoadbalancerListenerManager.Query().Equals("certificate_id", objs[i].(*SLoadbalancerCertificate).GetId())
		ownerId, queryScope, err, _ := db.FetchCheckQueryOwnerScope(ctx, userCred, query, LoadbalancerListenerManager, policy.PolicyActionList, true)
		if err != nil {
			log.Errorf("FetchCheckQueryOwnerScope error: %v", err)
			return rows
		}

		q = LoadbalancerListenerManager.FilterByOwner(q, ownerId, queryScope)
		rows[i].LbListenerCount, _ = q.CountWithError()
	}

	return rows
}

func (lbcert *SLoadbalancerCertificate) ValidateDeleteCondition(ctx context.Context, info *api.LoadbalancerCertificateDetails) error {
	if info != nil && info.LbListenerCount > 0 {
		return httperrors.NewNotEmptyError("cert %s with %d listeners", lbcert.Name, info.LbListenerCount)
	}
	return lbcert.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx, jsonutils.Marshal(info))
}

func (lbcert *SLoadbalancerCertificate) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	caches, err := lbcert.GetCachedCerts()
	if err != nil {
		return errors.Wrap(err, "GetCachedCerts")
	}

	for i := range caches {
		err := caches[i].Delete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "Delete cache %s", caches[i].Id)
		}
	}
	return lbcert.SSharableVirtualResourceBase.Delete(ctx, userCred)
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
			q2 := CachedLoadbalancerCertificateManager.Query("certificate_id")
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

func (man *SLoadbalancerCertificateManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.LoadbalancerCertificateCreateInput) (*api.LoadbalancerCertificateCreateInput, error) {
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
	input.Status = api.LB_STATUS_ENABLED
	return input, nil
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
