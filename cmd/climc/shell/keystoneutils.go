package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func getUserId(s *mcclient.ClientSession, user string, domain string) (string, error) {
	query := jsonutils.NewDict()
	if len(domain) > 0 {
		domainId, err := modules.Domains.GetId(s, domain, nil)
		if err != nil {
			return "", err
		}
		query.Add(jsonutils.NewString(domainId), "domain_id")
	}
	return modules.UsersV3.GetId(s, user, query)
}

func getGroupId(s *mcclient.ClientSession, group string, domain string) (string, error) {
	query := jsonutils.NewDict()
	if len(domain) > 0 {
		domainId, err := modules.Domains.GetId(s, domain, nil)
		if err != nil {
			return "", err
		}
		query.Add(jsonutils.NewString(domainId), "domain_id")
	}
	return modules.Groups.GetId(s, group, query)
}

func getRoleId(s *mcclient.ClientSession, role string, domain string) (string, error) {
	query := jsonutils.NewDict()
	if len(domain) > 0 {
		domainId, err := modules.Domains.GetId(s, domain, nil)
		if err != nil {
			return "", err
		}
		query.Add(jsonutils.NewString(domainId), "domain_id")
	}
	return modules.RolesV3.GetId(s, role, query)
}

func getProjectId(s *mcclient.ClientSession, project string, domain string) (string, error) {
	query := jsonutils.NewDict()
	if len(domain) > 0 {
		domainId, err := modules.Domains.GetId(s, domain, nil)
		if err != nil {
			return "", err
		}
		query.Add(jsonutils.NewString(domainId), "domain_id")
	}
	return modules.Projects.GetId(s, project, query)
}

func getUserGroupId(s *mcclient.ClientSession, user, group, domain string) (string, string, error) {
	query := jsonutils.NewDict()
	if len(domain) > 0 {
		domainId, err := modules.Domains.GetId(s, domain, nil)
		if err != nil {
			return "", "", err
		}
		query.Add(jsonutils.NewString(domainId), "domain_id")
	}
	uid, err := modules.UsersV3.GetId(s, user, query)
	if err != nil {
		return "", "", err
	}
	gid, err := modules.Groups.GetId(s, group, query)
	if err != nil {
		return "", "", err
	}
	return uid, gid, nil
}
