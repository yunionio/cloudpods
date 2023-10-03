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
	"fmt"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/sets"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	noapi "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/keystone/models"
	o "yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/notify"
)

type SSQLDriver struct {
	driver.SBaseIdentityDriver
}

func NewSQLDriver(idpId, idpName, template, targetDomainId string, conf api.TConfigs) (driver.IIdentityBackend, error) {
	base, err := driver.NewBaseIdentityDriver(idpId, idpName, template, targetDomainId, conf)
	if err != nil {
		return nil, err
	}
	drv := SSQLDriver{base}
	drv.SetVirtualObject(&drv)
	return &drv, nil
}

func (sql *SSQLDriver) GetSsoRedirectUri(ctx context.Context, callbackUrl, state string) (string, error) {
	return "", errors.Wrap(httperrors.ErrNotSupported, "not a SSO driver")
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
	localUser, err := models.LocalUserManager.FetchLocalUserById(usrExt.LocalId)
	if err != nil {
		return nil, errors.Wrap(err, "LocalUserManager.FetchLocalUser")
	}
	err = models.VerifyPassword(usrExt, ident.Password.User.Password)
	if err != nil {
		localUser.SaveFailedAuth()
		if o.Options.PasswordErrorLockCount > 0 && localUser.FailedAuthCount > o.Options.PasswordErrorLockCount && !usrExt.IsSystemAccount {
			// do not lock system account!!!
			models.UserManager.LockUser(usrExt.Id, "too many failed auth attempts")
			data := jsonutils.NewDict()
			data.Set("name", jsonutils.NewString(usrExt.Name))
			notifyclient.SystemEventNotify(ctx, noapi.ActionLock, noapi.TOPIC_RESOURCE_USER, data)
			return nil, errors.Wrap(httperrors.ErrTooManyAttempts, "user locked")
		}
		return nil, errors.Wrap(err, "usrExt.VerifyPassword")
	}
	localUser.ClearFailedAuth()

	usrExt.AuditIds = []string{fmt.Sprintf("%d", localUser.Id)}
	return usrExt, nil
}

func (sql *SSQLDriver) alertNotify(ctx context.Context, uext *api.SUserExtended, triggerTime time.Time) {
	// users
	data := jsonutils.NewDict()
	data.Set("user", jsonutils.NewString(uext.Name))
	data.Set("domain", jsonutils.NewString(uext.DomainName))
	metadata := map[string]interface{}{
		"trigger_time": triggerTime,
	}
	p := notifyclient.SNotifyParams{
		RecipientId:               []string{uext.Id},
		Priority:                  notify.NotifyPriorityCritical,
		Event:                     notifyclient.USER_LOGIN_EXCEPTION,
		Data:                      data,
		Tag:                       noapi.NOTIFICATION_TAG_ALERT,
		Metadata:                  metadata,
		IgnoreNonexistentReceiver: true,
	}
	notifyclient.NotifyWithTag(ctx, p)

	// admin user
	daUserIds, err := getDomainAdminUserIds(uext.DomainName)
	if err != nil {
		log.Errorf("unable to get user with role domainadmin in domain %s: %v", uext.DomainName, err)
	}
	aUserIds, err := getAdminUserIds()
	if err != nil {
		log.Errorf("unable to get user with role admin: %v", err)
	}
	userSet := sets.NewString(daUserIds...)
	userSet.Insert(aUserIds...)
	data.Set("admin", jsonutils.JSONTrue)
	// prevent duplicate messages
	userSet.Delete(uext.Id)
	p.RecipientId = userSet.UnsortedList()
	p.Data = data
	notifyclient.NotifyWithTag(ctx, p)
}

func fetchRoleId(name string) (string, error) {
	id := struct {
		Id string
	}{}
	q := models.RoleManager.Query().Equals("name", name)
	err := q.First(&id)
	if err != nil {
		return "", err
	}
	return id.Id, nil
}

func getAdminUserIds() ([]string, error) {
	return getUserIdsWithRole(o.Options.AdminRoleToNotify, "")
}

func getUserIdsWithRole(roleName string, domainId string) ([]string, error) {
	roleId, err := fetchRoleId(roleName)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to fetch roleid of %s", roleName)
	}
	ras, _, err := models.AssignmentManager.FetchAll("", "", roleId, "", "", "", []string{}, []string{}, []string{}, []string{}, []string{}, []string{domainId}, false, true, false, false, false, 0, 0)
	if err != nil {
		return nil, err
	}
	userIds := make([]string, 0, len(ras))
	for i := range ras {
		userIds = append(userIds, ras[i].User.Id)
	}
	log.Infof("%s User: %v", roleName, userIds)
	return userIds, nil
}

func getDomainAdminUserIds(domainId string) ([]string, error) {
	return getUserIdsWithRole(o.Options.DomainAdminRoleToNotify, domainId)
}

func (sql *SSQLDriver) Sync(ctx context.Context) error {
	return nil
}

func (sql *SSQLDriver) Probe(ctx context.Context) error {
	return nil
}
