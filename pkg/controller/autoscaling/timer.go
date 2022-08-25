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

package autoscaling

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type STimeScope struct {
	Start  time.Time
	End    time.Time
	Median time.Time
}

func (asc *SASController) timeScope(median time.Time, interval time.Duration) STimeScope {
	ri := interval / 2
	return STimeScope{
		Start:  median.Add(-ri),
		End:    median.Add(ri),
		Median: median,
	}
}

func (asc *SASController) Timer(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	// 60 is for fault tolerance
	interval := asc.options.TimerInterval + 30
	timeScope := asc.timeScope(time.Now(), time.Duration(interval)*time.Second)
	spSubQ := models.ScalingPolicyManager.Query("id").Equals("status", compute.SP_STATUS_READY).SubQuery()
	q := models.ScalingTimerManager.Query().
		LT("next_time", timeScope.End).
		IsFalse("is_expired").In("scaling_policy_id", spSubQ)
	scalingTimers := make([]models.SScalingTimer, 0, 5)
	err := db.FetchModelObjects(models.ScalingTimerManager, q, &scalingTimers)
	if err != nil {
		log.Errorf("db.FetchModelObjects error: %s", err.Error())
		return
	}
	log.Debugf("total %d need to exec, %v", len(scalingTimers), scalingTimers)
	log.Debugf("timeScope: start: %s, end: %s", timeScope.Start, timeScope.End)
	session := auth.GetSession(ctx, userCred, "")
	triggerParams := jsonutils.NewDict()
	for i := range scalingTimers {
		scalingTimer := scalingTimers[i]
		asc.timerQueue <- struct{}{}
		go func(ctx context.Context) {
			defer func() {
				<-asc.timerQueue
			}()
			if scalingTimer.NextTime.Before(timeScope.Start) {
				// For unknown reasons, the scalingTimer did not execute at the specified time
				scalingTimer.Update(timeScope.Start)
				// scalingTimer should not exec for now.
				if scalingTimer.NextTime.After(timeScope.End) || scalingTimer.IsExpired {
					err = models.ScalingTimerManager.TableSpec().InsertOrUpdate(ctx, &scalingTimer)
					if err != nil {
						log.Errorf("update ScalingTimer whose ScalingPolicyId is %s error: %s",
							scalingTimer.ScalingPolicyId, err.Error())
					}
					return
				}
			}
			_, err = modules.ScalingPolicy.PerformAction(session, scalingTimer.ScalingPolicyId, "trigger",
				triggerParams)
			if err != nil {
				log.Errorf("unable to request to trigger ScalingPolicy '%s'", scalingTimer.ScalingPolicyId)
			}
			scalingTimer.Update(timeScope.End)
			err = models.ScalingTimerManager.TableSpec().InsertOrUpdate(ctx, &scalingTimer)
			if err != nil {
				log.Errorf("update ScalingTimer whose ScalingPolicyId is %s error: %s",
					scalingTimer.ScalingPolicyId, err.Error())
			}
		}(ctx)
	}
}
