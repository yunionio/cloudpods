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
	"fmt"

	"golang.org/x/sync/errgroup"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var SubscriptionReceiverManager *SSubscriptionReceiverManager

func init() {
	SubscriptionReceiverManager = &SSubscriptionReceiverManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SSubscriptionReceiver{},
			"subscriptionreceiver_tbl",
			"subscriptionreceiver",
			"subscriptionreceivers",
		),
	}
	SubscriptionReceiverManager.SetVirtualObject(ReceiverNotificationManager)
}

type SSubscriptionReceiverManager struct {
	db.SStandaloneResourceBaseManager
}

type SSubscriptionReceiver struct {
	db.SStandaloneResourceBase

	SubscriptionID string `width:"128" charset:"ascii" nullable:"fase" index:"true"`
	// role id or receiver id or other and the value type is determined by the ReceiverType
	Receiver     string `width:"128" charset:"ascii" nullable:"false"`
	ReceiverType string `width:"16" charset:"ascii" nullable:"false" index:"true"`
	RoleScope    string `width:"8" charset:"ascii" nullable:"false"`
}

const (
	ReceiverRole          = "role"
	ReceiverNormal        = "normal"
	ReceiverFeishuRobot   = notify.FEISHU_ROBOT
	ReceiverDingtalkRobot = notify.DINGTALK_ROBOT
	ReceiverWorkwxRobot   = notify.WORKWX_ROBOT
	ReceiverWeebhook      = "webhook"

	ScopeSystem  = "system"
	ScopeDomain  = "domain"
	ScopeProject = "project"
)

func (srm *SSubscriptionReceiverManager) robot(ssid string) (string, error) {
	return srm.findSingleReceiver(ReceiverFeishuRobot, ReceiverDingtalkRobot, ReceiverWorkwxRobot)
}

func (srm *SSubscriptionReceiverManager) webhook(ssid string) (string, error) {
	return srm.findSingleReceiver(ssid, ReceiverWeebhook)
}

func (srm *SSubscriptionReceiverManager) findSingleReceiver(ssid string, receiverTypes ...string) (string, error) {
	srs, err := srm.findReceivers(ssid, receiverTypes...)
	if err != nil {
		return "", err
	}
	if len(srs) > 1 {
		return "", errors.Error("multi receiver")
	}
	if len(srs) == 0 {
		return "", errors.ErrNotFound
	}
	return srs[0].ReceiverType, nil
}

func (srm *SSubscriptionReceiverManager) findReceivers(ssid string, receiverTypes ...string) ([]SSubscriptionReceiver, error) {
	if len(receiverTypes) == 0 {
		return nil, nil
	}
	q := srm.Query().Equals("subscription_id", ssid)
	if len(receiverTypes) == 1 {
		q = q.Equals("receiver_type", receiverTypes[0])
	} else {
		q = q.In("receiver_type", receiverTypes)
	}
	srs := make([]SSubscriptionReceiver, 0)
	err := db.FetchModelObjects(srm, q, &srs)
	if err != nil {
		return nil, errors.Wrap(err, "unable to FetchModelObjects")
	}
	return srs, nil
}

// TODO: Use cache to increase speed
func (srm *SSubscriptionReceiverManager) getReceivers(ctx context.Context, ssid string, projectDomainId string, projectId string) ([]string, error) {
	srs, err := srm.findReceivers(ssid, ReceiverRole, ReceiverNormal)
	if err != nil {
		return nil, err
	}
	receivers := make([]string, 0, len(srs))
	roleMap := make(map[string][]string, 3)
	receivermap := make(map[string]*[]string, 3)
	for _, sr := range srs {
		if sr.ReceiverType == ReceiverNormal {
			receivers = append(receivers, sr.Receiver)
		} else if sr.ReceiverType == ReceiverRole {
			roleMap[sr.RoleScope] = append(roleMap[sr.RoleScope], sr.Receiver)
			receivermap[sr.RoleScope] = &[]string{}
		}
	}
	errgo, _ := errgroup.WithContext(ctx)
	for scope, roles := range roleMap {
		receivers := receivermap[scope]
		errgo.Go(func() error {
			query := jsonutils.NewDict()
			query.Set("roles", jsonutils.NewStringArray(roles))
			query.Set("effective", jsonutils.JSONTrue)
			switch scope {
			case ScopeSystem:
			case ScopeDomain:
				if len(projectDomainId) == 0 {
					return fmt.Errorf("need projectDomainId")
				}
				query.Set("project_domain_id", jsonutils.NewString(projectDomainId))
			case ScopeProject:
				if len(projectId) == 0 {
					return fmt.Errorf("need projectId")
				}
				query.Add(jsonutils.NewString(projectId), "scope", "project", "id")
			}
			s := auth.GetAdminSession(ctx, "", "")
			log.Debugf("query for role-assignments: %s", query.String())
			listRet, err := modules.RoleAssignments.List(s, query)
			if err != nil {
				return errors.Wrap(err, "unable to list RoleAssignments")
			}
			log.Debugf("return value for role-assignments: %s", jsonutils.Marshal(listRet))
			for i := range listRet.Data {
				ras := listRet.Data[i]
				user, err := ras.Get("user")
				if err == nil {
					id, err := user.GetString("id")
					if err != nil {
						return errors.Wrap(err, "unable to get user.id from result of RoleAssignments.List")
					}
					*receivers = append(*receivers, id)
				}
			}
			return nil
		})
	}
	err = errgo.Wait()
	if err != nil {
		return nil, err
	}
	for _, res := range receivermap {
		receivers = append(receivers, *res...)
	}
	return receivers, nil
}

func (srm *SSubscriptionReceiverManager) create(ctx context.Context, ssid, receiver, receiverType, roleScope string) (*SSubscriptionReceiver, error) {
	sr := &SSubscriptionReceiver{
		SubscriptionID: ssid,
		Receiver:       receiver,
		ReceiverType:   receiverType,
		RoleScope:      roleScope,
	}
	err := srm.TableSpec().Insert(ctx, sr)
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert")
	}
	return sr, nil
}

func (sr *SSubscriptionReceiver) receivingRole() notify.ReceivingRole {
	return notify.ReceivingRole{
		Role:  sr.Receiver,
		Scope: sr.RoleScope,
	}
}
