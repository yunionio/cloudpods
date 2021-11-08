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

package identity

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
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
