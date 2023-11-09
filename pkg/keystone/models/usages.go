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
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/tagutils"
)

func Usage(result rbacutils.SPolicyResult) map[string]int {
	results := make(map[string]int)

	dq := DomainManager.Query()
	dq = db.ObjectIdQueryWithPolicyResult(dq, DomainManager, result)
	domCnt, _ := dq.IsTrue("is_domain").NotEquals("id", api.KeystoneDomainRoot).CountWithError()
	results["domains"] = domCnt

	pq := ProjectManager.Query()
	pq = db.ObjectIdQueryWithPolicyResult(pq, ProjectManager, result)

	// 根据项目标签过滤
	if result.ProjectTags.Len() > 0 {
		tagFilters := tagutils.STagFilters{}
		tagFilters.AddFilters(result.ProjectTags)
		orConditions := []sqlchemy.ICondition{}
		metadataResSQ := db.Metadata.Query().SubQuery()
		for _, filter := range tagFilters.Filters {
			for key, val := range filter {
				orConditions = append(orConditions,
					sqlchemy.AND(
						sqlchemy.Equals(metadataResSQ.Field("key"), key),
						sqlchemy.In(metadataResSQ.Field("value"), val),
					),
				)
			}
		}
		metadataResSQ = metadataResSQ.Query().Filter(sqlchemy.OR(orConditions...)).SubQuery()
		pq = pq.Join(metadataResSQ, sqlchemy.Equals(pq.Field("id"), metadataResSQ.Field("obj_id")))
	}

	projCnt, _ := pq.IsFalse("is_domain").CountWithError()
	results["projects"] = projCnt

	rq := RoleManager.Query()
	rq = db.ObjectIdQueryWithPolicyResult(rq, RoleManager, result)
	roleCnt, _ := rq.CountWithError()
	results["roles"] = roleCnt

	uq := UserManager.Query()
	uq = db.ObjectIdQueryWithPolicyResult(uq, UserManager, result)
	usrCnt, _ := uq.CountWithError()
	results["users"] = usrCnt

	gq := GroupManager.Query()
	gq = db.ObjectIdQueryWithPolicyResult(gq, GroupManager, result)
	grpCnt, _ := gq.CountWithError()
	results["groups"] = grpCnt

	pcq := PolicyManager.Query()
	pcq = db.ObjectIdQueryWithPolicyResult(pcq, PolicyManager, result)
	policy, _ := pcq.CountWithError()
	results["policies"] = policy

	return results
}
