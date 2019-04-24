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

package modules

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type DomainManager struct {
	ResourceManager
}

func (this *DomainManager) GetConfig(s *mcclient.ClientSession, domain string) (jsonutils.JSONObject, error) {
	url := fmt.Sprintf("/domains/%s/config", domain)
	return this._get(s, url, "config")
}

func (this *DomainManager) UpdateConfig(s *mcclient.ClientSession, domain string, config jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	result, e := this._updateConfig(s, domain, config)
	if e != nil {
		return result, e
	}
	driver, e := config.Get("config", "identity", "driver")
	if e == nil {
		body := jsonutils.NewDict()
		body.Add(driver, "driver")
		this.Patch(s, domain, body)
	}
	return result, e
}

func (this *DomainManager) _updateConfig(s *mcclient.ClientSession, domain string, config jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	driver, e := config.GetString("config", "identity", "driver")
	if e != nil {
		return nil, httperrors.NewInputParameterError("Malformed domain configuration %s", driver)
	}
	if driver != "ldap" {
		return nil, httperrors.NewInputParameterError("Invalid driver: %s, ONLY ldap is supported", driver)
	}
	url := fmt.Sprintf("/domains/%s/config", domain)
	ret, e := this._patch(s, url, config, "config")
	if e != nil {
		je, ok := e.(*httputils.JSONClientError)
		if ok && je.Code == 404 {
			return this._put(s, url, config, "config")
		} else {
			return nil, e
		}
	} else {
		return ret, nil
	}
}

func (this *DomainManager) DeleteConfig(s *mcclient.ClientSession, domain string) (jsonutils.JSONObject, error) {
	if domain == "default" {
		return nil, httperrors.NewForbiddenError("domain %s did not allowed deleted", domain)
	}

	result, e := this._deleteConfig(s, domain)
	if e != nil {
		return result, e
	}
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString(""), "driver")
	this.Patch(s, domain, body)
	return result, e
}

func (this *DomainManager) _deleteConfig(s *mcclient.ClientSession, domain string) (jsonutils.JSONObject, error) {
	url := fmt.Sprintf("/domains/%s/config", domain)
	return this._delete(s, url, nil, "config")
}

func (this *DomainManager) GetDomainConfig(s *mcclient.ClientSession, domain string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	ret := jsonutils.NewDict()

	domain_detail, err := this.Get(s, domain, nil)
	if err != nil {
		return ret, err
	}

	config, err := this.GetConfig(s, domain)
	if err != nil {
		// for empty domain config
		log.Infof("err fetch domain config for %s with error: %s", domain, err)
		config = jsonutils.NewDict()
	}

	ret.Add(domain_detail, "domain")
	ret.Add(config, "config")
	return ret, nil
}

func (this *DomainManager) DoDomainConfigUpdate(s *mcclient.ClientSession, domain string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// params example:
	// {
	//     "config": {
	//         "identity": {
	//             "driver": "ldap"
	//         },
	//         "ldap": {
	//             "group_id_attribute": "cn",
	//             "group_member_attribute": "member",
	//             "group_name_attribute": "cn",
	//             "group_objectclass": "ipausergroup",
	//             "group_tree_dn": "CN=groups,CN=accounts,DC=ipa,DC=yunionyun,DC=com",
	//             "page_size": 20,
	//             "query_scope": "sub",
	//             "suffix": "DC=ipa,DC=yunionyun,DC=com",
	//             "url": "ldap://192.168.0.222",
	//             "user": "UID=dcadmin,CN=users,CN=accounts,DC=ipa,DC=yunionyun,DC=com",
	//             "user_additional_attribute_mapping": [
	//                 "displayName:displayname",
	//                 "telephoneNumber:mobile"
	//             ],
	//             "user_enabled_attribute": "nsAccountLock",
	//             "user_enabled_default": "FALSE",
	//             "user_enabled_invert": true,
	//             "user_enabled_mask": 0,
	//             "user_id_attribute": "uid",
	//             "user_name_attribute": "uid",
	//             "user_objectclass": "person",
	//             "user_tree_dn": "CN=users,CN=accounts,DC=ipa,DC=yunionyun,DC=com"
	//         }
	//     },
	//     "domain": {
	//         "description": "SqnkThciWBq7",
	//         "enabled": true,
	//         "name": "os8vFdmqlgji-delete-free"
	//     }
	// }

	ret := jsonutils.NewDict()

	_domain, err := params.Get("domain")
	if err != nil {
		return ret, httperrors.NewMissingParameterError("domain")
	}
	name, _ := _domain.GetString("name")
	if domain == "default" && name != "Default" {
		return nil, httperrors.NewUnsupportOperationError("domain %s did not allowed update Name", domain)
	}

	_domain, err = this.Patch(s, domain, _domain)
	if err != nil {
		return ret, err
	}

	config := jsonutils.NewDict()
	_config, _ := params.Get("config")
	if _config == nil {
		_config = jsonutils.NewDict()
	}
	_driver, _ := _config.GetString("identity", "driver")

	if _driver == "ldap" {
		config.Add(_config, "config")
		log.Infof("to update config: %s", config)
		_config, err = this.UpdateConfig(s, domain, config)
		if err != nil {
			return ret, err
		}
		ret.Add(_config, "config")
	}

	ret.Add(_domain, "domain")

	return ret, nil
}

func (this *DomainManager) DoDomainConfigCreate(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	_domain, err := params.Get("domain")

	if err != nil {
		return ret, httperrors.NewMissingParameterError("domain")
	}

	_domain, err = this.Create(s, _domain)
	if err != nil {
		return ret, err
	}

	objId, err := _domain.GetString("id")
	if err != nil {
		return ret, err
	}

	config := jsonutils.NewDict()
	_config, _ := params.Get("config")
	_driver, _ := _config.Get("identity")

	if _driver != nil {
		config.Add(_config, "config")
		_config, err = this.UpdateConfig(s, objId, config)
		if err != nil {
			return ret, err
		}
		ret.Add(_config, "config")
	}

	ret.Add(_domain, "domain")
	return ret, nil
}

func (this *DomainManager) DoDomainConfigDelete(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()

	ids, _ := params.GetArray("ids")
	domains := jsonutils.JSONArray2StringArray(ids)

	for _, domain := range domains {
		objId, err := this.GetId(s, domain, nil)
		if err != nil {
			return ret, err
		}

		defer func() {
			if err := recover(); err != nil {
				this.DeleteConfig(s, objId)
				this.Delete(s, objId, nil)
			}
		}()

		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(objId), "domain_id")

		detail, err := this.GetById(s, domain, nil)
		if err != nil {
			log.Errorf("got domain detail error: %v", err)
			return ret, httperrors.NewResourceNotFoundError("%s %s not find", "Domain", domain)
		}

		driver, _ := detail.GetString("driver")
		if driver != "ldap" {
			if result, err := UsersV3.List(s, params); err != nil {
				log.Errorf("user list got error: %v", err)
				return ret, httperrors.NewInternalServerError("fetching user list failed: %s", err)
			} else if len(result.Data) > 0 {
				return ret, httperrors.NewForbiddenError("cannot delete: there still exists %d user related with domain %s.", len(result.Data), objId)
			}
		}

		this.DeleteConfig(s, objId)
		this.Delete(s, objId, nil)
	}
	return ret, nil
}

var (
	Domains DomainManager
)

func init() {
	Domains = DomainManager{NewIdentityV3Manager("domain", "domains",
		[]string{"ID", "Name", "Enabled", "Description", "Driver"},
		[]string{})}

	register(&Domains)
}
