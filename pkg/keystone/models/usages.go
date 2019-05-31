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
	api "yunion.io/x/onecloud/pkg/apis/identity"
)

func Usage() map[string]int {
	results := make(map[string]int)

	domCnt, _ := DomainManager.Query().IsTrue("is_domain").NotEquals("id", api.KeystoneDomainRoot).CountWithError()
	results["domains"] = domCnt

	projCnt, _ := ProjectManager.Query().IsFalse("is_domain").CountWithError()
	results["projects"] = projCnt

	roleCnt, _ := RoleManager.Query().CountWithError()
	results["roles"] = roleCnt

	usrCnt, _ := UserManager.Query().CountWithError()
	results["users"] = usrCnt

	grpCnt, _ := GroupManager.Query().CountWithError()
	results["groups"] = grpCnt

	policy, _ := PolicyManager.Query().CountWithError()
	results["policies"] = policy

	return results
}
