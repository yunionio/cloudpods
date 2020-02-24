package autoscaling

import (
	"sync"
	"time"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/pkg/util/sets"
)

type SASController struct {
	options         options.SASControllerOptions
	scalingQueue    chan SScalingInfo
	timerQueue      chan struct{}
	scalingGroupSet SLockedSet
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
	set.lock.Lock()
	defer set.lock.Unlock()
	return set.set.Has(s)
}

func (set *SLockedSet) Insert(s string) {
	set.lock.Lock()
	defer set.lock.Unlock()
	set.set.Insert(s)
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
	asc.scalingQueue = make(chan SScalingInfo, options.QueueSize)
	asc.timerQueue = make(chan struct{}, 20)
	// wait for activity
}

func (asc *SASController) StartScale(scaling SScalingInfo) {
	go func() {
		asc.scalingQueue <- scaling
	}()
}
