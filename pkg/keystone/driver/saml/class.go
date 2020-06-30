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

package saml

import (
	"context"
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SSAMLDriverClass struct{}

func (self *SSAMLDriverClass) SingletonInstance() bool {
	return false
}

func (self *SSAMLDriverClass) SyncMethod() string {
	return api.IdentityProviderSyncOnAuth
}

func (self *SSAMLDriverClass) NewDriver(idpId, idpName, template, targetDomainId string, conf api.TConfigs) (driver.IIdentityBackend, error) {
	return NewSAMLDriver(idpId, idpName, template, targetDomainId, conf)
}

func (self *SSAMLDriverClass) Name() string {
	return api.IdentityDriverSAML
}

func (self *SSAMLDriverClass) ValidateConfig(ctx context.Context, userCred mcclient.TokenCredential, tconf api.TConfigs) (api.TConfigs, error) {
	conf := api.SSAMLIdpConfigOptions{}
	confJson := jsonutils.Marshal(tconf["saml"])
	err := confJson.Unmarshal(&conf)
	if err != nil {
		return tconf, errors.Wrap(err, "unmarshal config")
	}
	if len(conf.DefaultProjectId) > 0 {
		obj, err := models.ProjectManager.FetchByIdOrName(userCred, conf.DefaultProjectId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return tconf, errors.Wrapf(httperrors.ErrResourceNotFound, "project %s", conf.DefaultProjectId)
			} else {
				return tconf, errors.Wrap(err, "FetchProjectById")
			}
		}
		tconf["cas"]["default_project_id"] = jsonutils.NewString(obj.GetId())
	}
	if len(conf.DefaultRoleId) > 0 {
		obj, err := models.RoleManager.FetchByIdOrName(userCred, conf.DefaultRoleId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return tconf, errors.Wrapf(httperrors.ErrResourceNotFound, "role %s", conf.DefaultRoleId)
			} else {
				return tconf, errors.Wrap(err, "FetchRoleById")
			}
		}
		tconf["cas"]["default_role_id"] = jsonutils.NewString(obj.GetId())
	}

	return tconf, nil
}

func init() {
	driver.RegisterDriverClass(&SSAMLDriverClass{})
}
