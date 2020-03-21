package autoscaling

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SASController struct {
	options         options.SASControllerOptions
	scalingQueue    chan struct{}
	timerQueue      chan struct{}
	scalingGroupSet SLockedSet
	scalingSql      string
}

type SScalingInfo struct {
	ScalingGroup *models.SScalingGroup
	Total        int
}

type SLockedSet struct {
	set  sets.String
	lock *sync.Mutex
}

func (set *SLockedSet) Has(s string) bool {
	return set.set.Has(s)
}

func (set *SLockedSet) CheckAndInsert(s string) bool {
	set.lock.Lock()
	defer set.lock.Unlock()
	if set.set.Has(s) {
		return false
	}
	set.set.Insert(s)
	return true
}

func (set *SLockedSet) Delete(s string) {
	set.lock.Lock()
	defer set.lock.Unlock()
	set.set.Delete(s)
}

var ASController = new(SASController)

func (asc *SASController) Init(options options.SASControllerOptions, cronm *cronman.SCronJobManager) {
	asc.options = options
	cronm.AddJobAtIntervals("CheckTimer", time.Duration(options.TimerInterval)*time.Second, asc.Timer)
	cronm.AddJobAtIntervals("CheckScale", time.Duration(options.CheckScaleInterval)*time.Second, asc.CheckScale)
	asc.timerQueue = make(chan struct{}, 20)
	asc.scalingQueue = make(chan struct{}, options.ConcurrentUpper)

	// init scaling sql
	var (
		scalingGroupAlias      = "sg"
		scalingGroupGuestAlias = "sgg"
	)
	asc.scalingSql = fmt.Sprintf("select %s.id, %s.desire_instance_number, %s.total from %s as %s left join "+
		"(select scaling_group_id, count(*) as total from %s where deleted='0' group by scaling_group_id) as %s "+
		"on %s.id = %s.scaling_group_id and %s.desire_instance_number != %s.total where deleted='0'",
		scalingGroupAlias, scalingGroupAlias, scalingGroupGuestAlias, models.ScalingGroupManager.TableSpec().Name(),
		scalingGroupAlias, models.ScalingGroupGuestManager.TableSpec().Name(), scalingGroupGuestAlias, scalingGroupAlias,
		scalingGroupGuestAlias, scalingGroupAlias, scalingGroupGuestAlias)
}

// SScalingGroupShort wrap the ScalingGroup's ID and DesireInstanceNumber with field 'total' which means the total
// guests number in this ScalingGroup
type SScalingGroupShort struct {
	ID                   string
	DesireInstanceNumber int `default:"0"`
	Total                int `default:"0"`
}

func (asc *SASController) CheckScale(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	sgShorts, err := asc.ScalingGroupsNeedScale()
	if err != nil {
		log.Errorf("asc.ScalingGroupNeedScale: %s", err.Error())
		return
	}
	for _, short := range sgShorts {
		if short.DesireInstanceNumber == short.Total {
			continue
		}
		insert := asc.scalingGroupSet.CheckAndInsert(short.ID)
		if !insert {
			log.Infof("A scaling activity of ScalingGroup %s is in progress, so current scaling activity was rejected.", short.ID)
			continue
		}
		asc.scalingQueue <- struct{}{}
	}
}

func (asc *SASController) Scale(short SScalingGroupShort) {
	var err error
	defer func() {
		if err != nil {
			log.Errorf("scale for ScalingGroup error: %s", err.Error())
		}
		asc.scalingGroupSet.Delete(short.ID)
		<-asc.scalingQueue
	}()
	// generate activity
	scalingActivity, err := models.ScalingActivityManager.CreateScalingActivity(short.ID, "", compute.SA_STATUS_WAIT)
	if err != nil {
		return
	}
	// fetch the latest data
	model, err := models.ScalingGroupManager.FetchById(short.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			scalingActivity.SimpleDelete()
			err = nil
		}
		return
	}
	sg := model.(*models.SScalingGroup)
	q := models.ScalingGroupGuestManager.Query().Equals("scaling_group_id", sg.Id)
	total, err := q.CountWithError()
	if err != nil {
		return
	}
	scalingActivity, err = scalingActivity.StartToScale(
		fmt.Sprintf(`The Desire Instance Number was changed, so change the Total Instance Number from "%d" to "%d"`,
			total, sg.DesireInstanceNumber,
		),
	)
	if err != nil {
		return
	}
	// check guest template
	gt := sg.GetGuestTemplate()
	if gt == nil {
		scalingActivity.SetResult("", false, fmt.Sprintf("fetch GuestTemplate of ScalingGroup '%s' error", sg.Id))
		return
	}
	valid, msg := gt.Validate(context.TODO(), auth.AdminCredential(), gt.GetOwnerId(), sg.Hypervisor, sg.CloudregionId)
	if !valid {
		scalingActivity.SetResult("", false, msg)
	}

	// userCred是管理员，ownerId是拥有者
}

// ScalingGroupNeedScale will fetch all ScalingGroup need to scale
func (asc *SASController) ScalingGroupsNeedScale() ([]SScalingGroupShort, error) {
	q := sqlchemy.NewRawQuery(asc.scalingSql)
	rows, err := q.Rows()
	if err != nil {
		return nil, errors.Wrap(err, "execute scaling sql error")
	}
	defer rows.Close()
	sgShorts := make([]SScalingGroupShort, 0, 10)
	for rows.Next() {
		sgPro := SScalingGroupShort{}
		err := q.Row2Struct(rows, &sgPro)
		if err != nil {
			return nil, errors.Wrap(err, "sqlchemy.SQuery.Row2Struct error")
		}
		sgShorts = append(sgShorts, sgPro)
	}
	return sgShorts, nil
}
