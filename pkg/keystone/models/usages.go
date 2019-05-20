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
