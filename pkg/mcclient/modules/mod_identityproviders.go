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

type IdentityProviderManager struct {
	ResourceManager
}

var (
	IdentityProviders IdentityProviderManager
)

/*
func (this *IdentityProviderManager) GetConfig(s *mcclient.ClientSession, idpId string) (jsonutils.JSONObject, error) {
	return this.GetSpecific(s, idpId, "config", nil)
}

func (this *IdentityProviderManager) UpdateConfig(s *mcclient.ClientSession, idpId string, config jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this.PerformAction(s, idpId, "config", config)
}

func (this *IdentityProviderManager) GetIdpConfig(s *mcclient.ClientSession, idpId string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()

	idpDetail, err := this.Get(s, idpId, nil)
	if err != nil {
		return ret, err
	}

	config, err := this.GetConfig(s, idpId)
	if err != nil {
		// for empty domain config
		log.Infof("err fetch domain config for %s with error: %s", idpId, err)
		config = jsonutils.NewDict()
	}

	ret.Add(idpDetail, "domain")
	ret.Add(config, "config")
	return ret, nil
}

func (this *IdentityProviderManager) DoIdpConfigUpdate(s *mcclient.ClientSession, idpId string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
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

	idp, err := params.Get("identity_provider")
	if err != nil {
		return ret, httperrors.NewMissingParameterError("domain")
	}
	name, _ := idp.GetString("name")
	if domain == "default" && name != "Default" {
		return nil, httperrors.NewUnsupportOperationError("domain %s did not allowed update Name", domain)
	}

	domain, err = this.Update(s, idpId, idp)
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
*/

func init() {
	IdentityProviders = IdentityProviderManager{
		NewIdentityV3Manager("identity_provider",
			"identity_providers",
			[]string{},
			[]string{"ID", "Name", "Driver", "Template", "Enabled", "Status", "Sync_Status", "Error_count", "Sync_Interval_Seconds"}),
	}

	register(&IdentityProviders)
}
