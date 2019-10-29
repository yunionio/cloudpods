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

package sql

import (
	"context"

	"github.com/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/keystone/models"
	o "yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SSQLDriver struct {
	driver.SBaseIdentityDriver
}

func NewSQLDriver(idpId, idpName, template, targetDomainId string, autoCreateProject bool, conf api.TIdentityProviderConfigs) (driver.IIdentityBackend, error) {
	base, err := driver.NewBaseIdentityDriver(idpId, idpName, template, targetDomainId, autoCreateProject, conf)
	if err != nil {
		return nil, err
	}
	drv := SSQLDriver{base}
	drv.SetVirtualObject(&drv)
	return &drv, nil
}

func (sql *SSQLDriver) Authenticate(ctx context.Context, ident mcclient.SAuthenticationIdentity) (*api.SUserExtended, error) {
	usrExt, err := models.UserManager.FetchUserExtended(
		ident.Password.User.Id,
		ident.Password.User.Name,
		ident.Password.User.Domain.Id,
		ident.Password.User.Domain.Name,
	)
	if err != nil {
		return nil, errors.Wrap(err, "UserManager.FetchUserExtended")
	}
	localUser, err := models.LocalUserManager.FetchLocalUser(usrExt)
	if err != nil {
		return nil, errors.Wrap(err, "LocalUserManager.FetchLocalUser")
	}
	err = models.VerifyPassword(usrExt, ident.Password.User.Password)
	if err != nil {
		localUser.SaveFailedAuth()
		if localUser.FailedAuthCount > o.Options.PasswordErrorLockCount {
			models.UserManager.LockUser(usrExt.Id)
		}
		return nil, errors.Wrap(err, "usrExt.VerifyPassword")
	}
	localUser.ClearFailedAuth()
	return usrExt, nil
}

func (sql *SSQLDriver) Sync(ctx context.Context) error {
	return nil
}

func (sql *SSQLDriver) Probe(ctx context.Context) error {
	return nil
}
