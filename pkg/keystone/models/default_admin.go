package models

import (
	"yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	defaultAdminCred mcclient.TokenCredential
)

func GetDefaultAdminCred() mcclient.TokenCredential {
	if defaultAdminCred == nil {
		defaultAdminCred = getDefaultAdminCred()
	}
	return defaultAdminCred
}

func getDefaultAdminCred() mcclient.TokenCredential {
	token := mcclient.SSimpleToken{}
	usr, _ := UserManager.FetchUserExtended("", options.Options.AdminUserName, options.Options.AdminUserDomainId, "")
	token.UserId = usr.Id
	token.User = usr.Name
	token.DomainId = usr.DomainId
	token.Domain = usr.DomainName
	prj, _ := ProjectManager.FetchProject("", options.Options.AdminProjectName, options.Options.AdminProjectDomainId, "")
	token.ProjectId = prj.Id
	token.Project = prj.Name
	token.ProjectDomainId = prj.DomainId
	token.ProjectDomain = prj.GetDomain().Name
	token.Roles = "admin"
	return &token
}
