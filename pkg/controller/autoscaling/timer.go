package autoscaling

import (
	"context"
	"fmt"
	"time"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"

	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
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

func (asc *)

func (asc *SASController) Timer(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {

	// 60 is for fault tolerance
	interval := asc.options.TimerInterval + 30
	timeScope := asc.timeScope(time.Now(), time.Duration(interval)*time.Second)
	spSubQ := models.ScalingPolicyManager.Query("id").Equals("status", compute.SP_STATUS_READY).SubQuery()
	q := models.ScalingTimerManager.Query().
		LT("next_time", timeScope.End).
		GT("next_time", timeScope.End).
		IsFalse("is_expired").In("scaling_policy_id", spSubQ)
	scalingTimers := make([]models.SScalingTimer, 0, 5)
	err := db.FetchModelObjects(models.ScalingTimerManager, q, &scalingTimers)
	if err != nil {
		log.Errorf("db.FetchModelObjects error: %s", err.Error())
		return
	}
	for _, scalingTimer := range scalingTimers {
		asc.timerQueue <- struct{}{}
		go func(ctx context.Context) {
			scalingActivity := &models.SScalingActivity{
				ScalingPolicyId: scalingTimer.ScalingPolicyId,
				TriggerDesc:     scalingTimer.Description(),
				ActionDesc:     "",
				StartTime:       time.Now(),
				EndTime:         time.Time{},
			}
			scalingActivity.Status = compute.SA_STATUS_EXEC
			err := models.ScalingActivityManager.TableSpec().Insert(&scalingActivity)
			if err != nil {
				log.Errorf("insert ScalingActivity whose ScalingPolicyId is %s error: %s",
					scalingActivity.ScalingPolicyId, err.Error())
			}

			// start exec
			lockman.LockObject()
			scalingGroup, err := scalingTimer.ScalingGroup()
			if err != nil {
				scalingActivity.SetResult("", false, fmt.Sprintf("fail to get ScalingTimer's ScalingGroup: %s", err.Error()))
			}

		}(ctx)

		scalingTimer.Update()
		err = models.ScalingTimerManager.TableSpec().InsertOrUpdate(&scalingTimer)
		if err != nil {
			log.Errorf("update ScalingTimer whose ScalingPolicyId is %s error: %s",
				scalingTimer.ScalingPolicyId, err.Error())
		}
	}
}
