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

package utils

import (
	"context"
	"database/sql"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func ValidateConfig(ctx context.Context, conf api.SIdpAttributeOptions, userCred mcclient.TokenCredential) (api.SIdpAttributeOptions, error) {
	if len(conf.DefaultProjectId) > 0 {
		obj, err := models.ProjectManager.FetchByIdOrName(ctx, userCred, conf.DefaultProjectId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return conf, errors.Wrapf(httperrors.ErrResourceNotFound, "project %s", conf.DefaultProjectId)
			} else {
				return conf, errors.Wrap(err, "FetchProjectById")
			}
		}
		conf.DefaultProjectId = obj.GetId()
	}
	if len(conf.DefaultRoleId) > 0 {
		obj, err := models.RoleManager.FetchByIdOrName(ctx, userCred, conf.DefaultRoleId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return conf, errors.Wrapf(httperrors.ErrResourceNotFound, "role %s", conf.DefaultRoleId)
			} else {
				return conf, errors.Wrap(err, "FetchRoleById")
			}
		}
		conf.DefaultRoleId = obj.GetId()
	}
	if len(conf.DefaultProjectId) > 0 && len(conf.DefaultRoleId) > 0 {
		// validate policy
		err := models.ValidateJoinProjectRoles(userCred, conf.DefaultProjectId, []string{conf.DefaultRoleId})
		if err != nil {
			return conf, errors.Wrap(err, "ValidateJoinProjectRoles")
		}
	}
	return conf, nil
}
