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
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/keystone/models"
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

func (self *SCASDriverClass) ValidateConfig(tconf api.TConfigs) error {

	conf := api.SCASIdpConfigOptions{}
	confJson := jsonutils.Marshal(tconf["cas"])
	log.Debugf("%s %s", tconf, confJson)
	err := confJson.Unmarshal(&conf)
	if err != nil {
		return errors.Wrap(err, "unmarshal config")
	}
	if len(conf.DefaultCasProjectId) > 0 {
		_, err := models.ProjectManager.FetchProjectById(conf.DefaultCasProjectId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return errors.Wrapf(httperrors.ErrResourceNotFound, "project %s", conf.DefaultCasProjectId)
			} else {
				return errors.Wrap(err, "FetchProjectById")
			}
		}
	}
	if len(conf.DefaultCasRoleId) > 0 {
		_, err := models.RoleManager.FetchRoleById(conf.DefaultCasRoleId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return errors.Wrapf(httperrors.ErrResourceNotFound, "role %s", conf.DefaultCasRoleId)
			} else {
				return errors.Wrap(err, "FetchRoleById")
			}
		}
	}
	return nil
}

func init() {
	driver.RegisterDriverClass(&SCASDriverClass{})
}
