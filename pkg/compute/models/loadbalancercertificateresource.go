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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SLoadbalancerCertificateResourceBase struct {
	// 本地负载均衡证书ID
	CertificateId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
}

type SLoadbalancerCertificateResourceBaseManager struct{}

func (self *SLoadbalancerCertificateResourceBase) GetCertificate() (*SLoadbalancerCertificate, error) {
	cert, err := LoadbalancerCertificateManager.FetchById(self.CertificateId)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchById(%s)", self.CertificateId)
	}
	return cert.(*SLoadbalancerCertificate), nil
}

func (manager *SLoadbalancerCertificateResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LoadbalancerCertificateResourceInfo {
	rows := make([]api.LoadbalancerCertificateResourceInfo, len(objs))
	certIds := make([]string, len(objs))
	for i := range objs {
		var base *SLoadbalancerCertificateResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SCloudregionResourceBase in object %s", objs[i])
			continue
		}
		certIds[i] = base.CertificateId
	}
	certs := make(map[string]SLoadbalancerCertificate)
	err := db.FetchStandaloneObjectsByIds(LoadbalancerCertificateManager, certIds, &certs)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}
	for i := range rows {
		rows[i] = api.LoadbalancerCertificateResourceInfo{}
		if cert, ok := certs[certIds[i]]; ok {
			rows[i].Certificate = cert.Name
		}
	}
	return rows
}

func (manager *SLoadbalancerCertificateResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerCertificateFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.CertificateId) > 0 {
		_, err := validators.ValidateModel(ctx, userCred, LoadbalancerCertificateManager, &query.CertificateId)
		if err != nil {
			return q, err
		}
		q = q.Equals("certificate_id", query.CertificateId)
	}
	return q, nil
}

func (manager *SLoadbalancerCertificateResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerCertificateFilterListInput,
) (*sqlchemy.SQuery, error) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, nil
	}
	orderQ := LoadbalancerCertificateManager.Query("id")
	orderSubQ := orderQ.SubQuery()
	orderQ, orders, fields := manager.GetOrderBySubQuery(orderQ, orderSubQ, orderQ.Field("id"), userCred, query, nil, nil)
	q = q.LeftJoin(orderSubQ, sqlchemy.Equals(q.Field("certificate_id"), orderSubQ.Field("id")))
	q = db.OrderByFields(q, orders, fields)
	return q, nil
}

func (manager *SLoadbalancerCertificateResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	if field == "certificate" {
		certQuery := LoadbalancerCertificateManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(certQuery.Field("name", field))
		q = q.Join(certQuery, sqlchemy.Equals(q.Field("certificate_id"), certQuery.Field("id")))
		q.GroupBy(certQuery.Field("name"))
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SLoadbalancerCertificateResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	subq *sqlchemy.SSubQuery,
	joinField sqlchemy.IQueryField,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerCertificateFilterListInput,
	orders []string,
	fields []sqlchemy.IQueryField,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, orders, fields
	}
	certQ := LoadbalancerCertificateManager.Query().SubQuery()
	q = q.LeftJoin(certQ, sqlchemy.Equals(joinField, certQ.Field("id")))
	q = q.AppendField(certQ.Field("name").Label("certificate"))
	orders = append(orders, query.OrderByCertificate)
	fields = append(fields, subq.Field("certificate"))
	return q, orders, fields
}

func (manager *SLoadbalancerCertificateResourceBaseManager) GetOrderByFields(query api.LoadbalancerCertificateFilterListInput) []string {
	return []string{query.OrderByCertificate}
}

func (manager *SLoadbalancerCertificateResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		subq := LoadbalancerCertificateManager.Query("id", "name").SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("certificate_id"), subq.Field("id")))
		if keys.Contains("certificate") {
			q = q.AppendField(subq.Field("name", "certificate"))
		}
	}
	return q, nil
}

func (manager *SLoadbalancerCertificateResourceBaseManager) GetExportKeys() []string {
	return []string{"certificate"}
}
