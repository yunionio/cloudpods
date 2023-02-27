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

package tasks

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/notify/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type RepullSuncontactTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(RepullSuncontactTask{})
}

func (self *RepullSuncontactTask) taskFailed(ctx context.Context, config *models.SConfig, reason string) {
	if !config.Deleted {
		logclient.AddActionLogWithContext(ctx, config, logclient.ACT_PULL_SUBCONTACT, reason, self.UserCred, false)
	}
	self.SetStageFailed(ctx, jsonutils.NewString(reason))
}

type repullFailedReason struct {
	ReceiverId string
	Reason     string
}

func (s repullFailedReason) String() string {
	return fmt.Sprintf("receiver %q: %s", s.ReceiverId, s.Reason)
}

func (self *RepullSuncontactTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	config := obj.(*models.SConfig)
	if !utils.IsInStringArray(config.Type, PullContactType) {
		if del, _ := self.GetParams().Bool("deleted"); del {
			config.RealDelete(ctx, self.GetUserCred())
		}
		self.SetStageComplete(ctx, nil)
		return
	}
	subq := models.SubContactManager.Query("receiver_id").Equals("type", config.Type).SubQuery()
	q := models.ReceiverManager.Query()
	if config.Attribution == notify.CONFIG_ATTRIBUTION_DOMAIN {
		q = q.Equals("domain_id", config.DomainId)
	} else {
		// The system-level config update should not affect the receiver under the domain with config
		configq := models.ConfigManager.Query("domain_id").Equals("type", config.Type).Equals("attribution", notify.CONFIG_ATTRIBUTION_DOMAIN).SubQuery()
		q = q.NotIn("domain_id", configq)
	}
	q.Join(subq, sqlchemy.Equals(q.Field("id"), subq.Field("receiver_id")))
	rs := make([]models.SReceiver, 0)
	err := db.FetchModelObjects(models.ReceiverManager, q, &rs)
	if err != nil {
		self.taskFailed(ctx, config, fmt.Sprintf("unable to FetchModelObjects: %v", err))
		return
	}

	if del, _ := self.GetParams().Bool("deleted"); del {
		config.RealDelete(ctx, self.GetUserCred())
	}

	var reasons []string
	for i := range rs {
		r := &rs[i]
		func() {
			lockman.LockObject(ctx, r)
			defer lockman.ReleaseObject(ctx, r)
			// unverify
			cts, err := r.GetVerifiedContactTypes()
			if err != nil {
				reasons = append(reasons, repullFailedReason{
					ReceiverId: r.Id,
					Reason:     fmt.Sprintf("unable to GetVerifiedContactTypes: %v", err),
				}.String())
				return
			}
			ctSets := sets.NewString(cts...)
			if ctSets.Has(config.Type) {
				ctSets.Delete(config.Type)
				// err = r.SetVerifiedContactTypes(ctSets.UnsortedList())
				// if err != nil {
				// 	reasons = append(reasons, repullFailedReason{
				// 		ReceiverId: r.Id,
				// 		Reason:     fmt.Sprintf("unable to SetVerifiedContactTypes: %v", err),
				// 	}.String())
				// 	return
				// }
			}
			// pull
			params := jsonutils.NewDict()
			params.Set("contact_types", jsonutils.NewArray(jsonutils.NewString(config.Type)))
			// err = r.StartSubcontactPullTask(ctx, self.UserCred, params, self.Id)
			// if err != nil {
			// 	reasons = append(reasons, repullFailedReason{
			// 		ReceiverId: r.Id,
			// 		Reason:     fmt.Sprintf("unable to StartSubcontactPullTask: %v", err),
			// 	}.String())
			// }
		}()
	}
	if len(reasons) > 0 {
		self.taskFailed(ctx, config, strings.Join(reasons, "; "))
		return
	}
	self.SetStageComplete(ctx, nil)
}
