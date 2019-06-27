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
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

func expandIdpAttributes(rows []*jsonutils.JSONDict, objs []db.IModel, fields stringutils2.SSortedStrings, entType string) []*jsonutils.JSONDict {
	if len(fields) == 0 || fields.Contains("idp_id") || fields.Contains("idp") || fields.Contains("idp_entity_id") || fields.Contains("idp_driver") {
		log.Debugf("objs %d", len(objs))
		idList := make([]string, len(objs))
		for i := range objs {
			idList[i] = objs[i].GetId()
		}
		idps, err := fetchIdmappings(idList, entType)
		if err == nil && idps != nil {
			for i := range rows {
				if idp, ok := idps[objs[i].GetId()]; ok {
					if len(fields) == 0 || fields.Contains("idp_id") {
						rows[i].Set("idp_id", jsonutils.NewString(idp.IdpId))
					}
					if len(fields) == 0 || fields.Contains("idp") {
						rows[i].Set("idp", jsonutils.NewString(idp.IdpName))
					}
					if len(fields) == 0 || fields.Contains("idp_entity_id") {
						rows[i].Set("idp_entity_id", jsonutils.NewString(idp.EntityId))
					}
					if len(fields) == 0 || fields.Contains("idp_driver") {
						rows[i].Set("idp_driver", jsonutils.NewString(idp.Driver))
					}
				}
			}
		} else if err != nil {
			log.Warningf("fetchIdmappings error %s", err)
		}
	}
	return rows
}

type sIdpInfo struct {
	IdpId    string
	IdpName  string
	EntityId string
	Driver   string
	PublicId string
}

func fetchIdmappings(idList []string, resType string) (map[string]sIdpInfo, error) {
	idmappings := IdmappingManager.Query().SubQuery()
	idps := IdentityProviderManager.Query().SubQuery()

	q := idmappings.Query(idmappings.Field("domain_id", "idp_id"),
		idmappings.Field("local_id", "entity_id"),
		idps.Field("name", "idp_name"),
		idps.Field("driver"),
		idmappings.Field("public_id"),
	)
	q = q.Join(idps, sqlchemy.Equals(idps.Field("id"), idmappings.Field("domain_id")))
	q = q.Filter(sqlchemy.In(idmappings.Field("public_id"), idList))
	q = q.Filter(sqlchemy.Equals(idmappings.Field("entity_type"), resType))

	idpInfos := make([]sIdpInfo, 0)
	err := q.All(&idpInfos)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "query")
	}

	ret := make(map[string]sIdpInfo)
	for i := range idpInfos {
		ret[idpInfos[i].PublicId] = idpInfos[i]
	}
	return ret, nil
}
