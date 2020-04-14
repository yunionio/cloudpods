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

package cas

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

type SCASDriverClass struct{}

func (self *SCASDriverClass) SingletonInstance() bool {
	return true
}

func (self *SCASDriverClass) SyncMethod() string {
	return api.IdentityProviderSyncOnAuth
}

func (self *SCASDriverClass) NewDriver(idpId, idpName, template, targetDomainId string, autoCreateProject bool, conf api.TConfigs) (driver.IIdentityBackend, error) {
	return NewCASDriver(idpId, idpName, template, targetDomainId, autoCreateProject, conf)
}

func (self *SCASDriverClass) Name() string {
	return api.IdentityDriverCAS
}

func (self *SCASDriverClass) ValidateConfig(ctx context.Context, userCred mcclient.TokenCredential, tconf api.TConfigs) (api.TConfigs, error) {

	conf := api.SCASIdpConfigOptions{}
	confJson := jsonutils.Marshal(tconf["cas"])
	err := confJson.Unmarshal(&conf)
	if err != nil {
		return tconf, errors.Wrap(err, "unmarshal config")
	}
	if len(conf.DefaultCasProjectId) > 0 {
		obj, err := models.ProjectManager.FetchByIdOrName(userCred, conf.DefaultCasProjectId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return tconf, errors.Wrapf(httperrors.ErrResourceNotFound, "project %s", conf.DefaultCasProjectId)
			} else {
				return tconf, errors.Wrap(err, "FetchProjectById")
			}
		}
		tconf["cas"]["default_cas_project_id"] = jsonutils.NewString(obj.GetId())
	}
	if len(conf.DefaultCasRoleId) > 0 {
		obj, err := models.RoleManager.FetchByIdOrName(userCred, conf.DefaultCasRoleId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return tconf, errors.Wrapf(httperrors.ErrResourceNotFound, "role %s", conf.DefaultCasRoleId)
			} else {
				return tconf, errors.Wrap(err, "FetchRoleById")
			}
		}
		tconf["cas"]["default_cas_role_id"] = jsonutils.NewString(obj.GetId())
	}

	return tconf, nil
}

func init() {
	driver.RegisterDriverClass(&SCASDriverClass{})
}
