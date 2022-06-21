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

package models

import (
	"context"

	"golang.org/x/text/language"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/notify"
	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SRobotManager struct {
	db.SSharableVirtualResourceBaseManager
	db.SEnabledResourceBaseManager
}

var RobotManager *SRobotManager

func init() {
	RobotManager = &SRobotManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SRobot{},
			"robots_tbl",
			"robot",
			"robots",
		),
	}
	RobotManager.SetVirtualObject(RobotManager)
}

type SRobot struct {
	db.SSharableVirtualResourceBase
	db.SEnabledResourceBase

	Type    string `width:"16" nullable:"false" create:"required" get:"user" list:"user" index:"true"`
	Address string `nullable:"false" create:"required" update:"user" get:"user" list:"user"`
	Lang    string `width:"16" nullable:"false" create:"required" update:"user" get:"user" list:"user"`
}

func (rm *SRobotManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.RobotCreateInput) (api.RobotCreateInput, error) {
	var err error
	input.SharableVirtualResourceCreateInput, err = rm.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ValidateCreateData")
	}
	// check type
	if !utils.IsInStringArray(input.Type, GetRobotTypes()) {
		return input, httperrors.NewInputParameterError("unkown type %s support: %s", input.Type, GetRobotTypes())
	}
	// check lang
	if input.Lang == "" {
		input.Lang = "zh_CN"
	}
	_, err = language.Parse(input.Lang)
	if err != nil {
		return input, httperrors.NewInputParameterError("invalid lang %s: %s", input.Lang, err.Error())
	}
	input.SetEnabled()
	input.Status = api.ROBOT_STATUS_READY
	driver := GetDriver(input.Type)
	err = driver.Send(ctx, api.SendParams{
		Receivers: []api.SNotifyReceiver{
			{
				Contact:  input.Address,
				DomainId: input.ProjectDomainId,
			},
		},
		Title:   "Validate",
		Message: "This is a verification message, please ignore.",
	})
	if err != nil {
		return input, err
	}
	return input, nil
}

func (rm *SRobotManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []api.RobotDetails {
	sRows := rm.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	rows := make([]api.RobotDetails, len(objs))
	for i := range rows {
		rows[i].SharableVirtualResourceDetails = sRows[i]
	}
	return rows
}

func (rm *SRobotManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.RobotListInput) (*sqlchemy.SQuery, error) {
	q, err := rm.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, err
	}
	q, err = rm.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, input.EnabledResourceBaseListInput)
	if err != nil {
		return nil, err
	}
	if len(input.Type) > 0 {
		q = q.Equals("type", input.Type)
	}
	if len(input.Lang) > 0 {
		q = q.Equals("lang", input.Lang)
	}
	return q, nil
}

func (r *SRobot) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.RobotUpdateInput) (api.RobotUpdateInput, error) {
	var err error
	input.SharableVirtualResourceBaseUpdateInput, err = r.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.SharableVirtualResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SSharableVirtualResourceBase.ValidateUpdateData")
	}
	// check lang
	if len(input.Lang) > 0 {
		_, err = language.Parse(input.Lang)
		if err != nil {
			return input, httperrors.NewInputParameterError("invalid lang %q: %s", input.Lang, err.Error())
		}
	}
	if len(input.Address) > 0 {
		// check Address
		driver := GetDriver(r.Type)
		err = driver.Send(ctx, api.SendParams{
			Receivers: []api.SNotifyReceiver{
				{
					Contact:  input.Address,
					DomainId: r.DomainId,
				},
			},
			Title:   "Validate",
			Message: "This is a verification message, please ignore.",
		})
		if err != nil {
			return input, err
		}
	}
	return input, nil
}

func (rm *SRobotManager) FetchByIdOrNames(ctx context.Context, idOrNames ...string) ([]SRobot, error) {
	if len(idOrNames) == 0 {
		return nil, nil
	}
	var err error
	q := idOrNameFilter(rm.Query(), idOrNames...)
	robots := make([]SRobot, 0, len(idOrNames))
	err = db.FetchModelObjects(rm, q, &robots)
	if err != nil {
		return nil, err
	}
	return robots, nil
}

func (r *SRobot) IsEnabled() bool {
	return r.Enabled.Bool()
}

func (r *SRobot) IsEnabledContactType(ctype string) (bool, error) {
	return ctype == api.ROBOT || ctype == api.WEBHOOK, nil
}

func (r *SRobot) IsVerifiedContactType(ctype string) (bool, error) {
	return ctype == api.ROBOT || ctype == api.WEBHOOK, nil
}

func (r *SRobot) GetContact(ctype string) (string, error) {
	return r.Address, nil
}

func (r *SRobot) GetTemplateLang(ctx context.Context) (string, error) {
	lang, err := language.Parse(r.Lang)
	if err != nil {
		return "", errors.Wrapf(err, "unable to prase language %q", r.Lang)
	}
	tLang := notifyclientI18nTable.LookupByLang(lang, tempalteLang)
	return tLang, nil
}

func (r *SRobot) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(r, ctx, userCred, true)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (r *SRobot) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(r, ctx, userCred, false)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (r *SRobot) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	q := SubscriberManager.Query().Equals("type", notify.SUBSCRIBER_TYPE_ROBOT).Equals("identification", r.GetId())
	subscribers := make([]SSubscriber, 0, 2)
	err := db.FetchModelObjects(SubscriberManager, q, &subscribers)
	if err != nil {
		log.Errorf("unable to fetch subscribers with robot %q", r.GetId())
		return
	}
	for i := range subscribers {
		subscriber := &subscribers[i]
		_, err := db.Update(r, func() error {
			return subscriber.MarkDelete()
		})
		if err != nil {
			log.Errorf("unable to delete subscriber %q because of deleting of robot %q", subscribers[i].GetId(), r.GetId())
		}
	}
}

func (rm *SRobotManager) InitializeData() error {
	return nil
}
