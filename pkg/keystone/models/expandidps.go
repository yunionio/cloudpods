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

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

func expandIdpAttributes(entType string, idList []string, fields stringutils2.SSortedStrings) []api.IdpResourceInfo {
	rows := make([]api.IdpResourceInfo, len(idList))
	if len(fields) == 0 || fields.Contains("idp_id") || fields.Contains("idp") || fields.Contains("idp_entity_id") || fields.Contains("idp_driver") {
		idps, err := fetchIdmappings(idList, entType)
		if err == nil && idps != nil {
			for i := range idList {
				if idp, ok := idps[idList[i]]; ok {
					if len(fields) == 0 || fields.Contains("idp_id") {
						rows[i].IdpId = idp[0].IdpId
					}
					if len(fields) == 0 || fields.Contains("idp") {
						rows[i].Idp = idp[0].Idp
					}
					if len(fields) == 0 || fields.Contains("idp_entity_id") {
						rows[i].IdpEntityId = idp[0].IdpEntityId
					}
					if len(fields) == 0 || fields.Contains("idp_driver") {
						rows[i].IdpDriver = idp[0].IdpDriver
					}
					if len(fields) == 0 || fields.Contains("template") {
						rows[i].Template = idp[0].Template
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
	api.IdpResourceInfo
	PublicId string
}

func fetchIdmappings(idList []string, resType string) (map[string][]sIdpInfo, error) {
	idmappings := IdmappingManager.Query().SubQuery()
	idps := IdentityProviderManager.Query().SubQuery()

	q := idmappings.Query(idmappings.Field("domain_id", "idp_id"),
		idmappings.Field("local_id", "idp_entity_id"),
		idps.Field("name", "idp"),
		idps.Field("driver", "idp_driver"),
		idps.Field("template", "template"),
		idps.Field("is_sso", "is_sso"),
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

	ret := make(map[string][]sIdpInfo)
	for i := range idpInfos {
		if idpList, ok := ret[idpInfos[i].PublicId]; ok {
			ret[idpInfos[i].PublicId] = append(idpList, idpInfos[i])
		} else {
			ret[idpInfos[i].PublicId] = []sIdpInfo{idpInfos[i]}
		}
	}
	return ret, nil
}
