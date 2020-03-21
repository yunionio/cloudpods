package autoscaling

import (
	"context"
	"time"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
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
			defer func() {
				<- asc.timerQueue
			}()
			sp, err := scalingTimer.ScalingPolicy()
			if err != nil {
				log.Errorf("fail to get ScalingPolicy of ScalingTimer '%s': %s", scalingTimer.Id,  err)
				return
			}
			sg, err := sp.ScalingGroup()
			if err != nil {
				log.Errorf("fail to get ScalingGroup of ScalingPolicy '%s': %s", sp.Id,  err)
				return
			}
			err = sg.Scale(ctx, &scalingTimer, sp)
			if err != nil {
				log.Errorf("ScalingGroup '%s' scale error", sg.Id)
			}
			scalingTimer.Update()
			err = models.ScalingTimerManager.TableSpec().InsertOrUpdate(&scalingTimer)
			if err != nil {
				log.Errorf("update ScalingTimer whose ScalingPolicyId is %s error: %s",
					scalingTimer.ScalingPolicyId, err.Error())
			}
		}(ctx)
	}
}
