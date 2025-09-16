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
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/scheduler"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/nopanic"
)

type SASController struct {
	options         options.SASControllerOptions
	scalingQueue    chan struct{}
	timerQueue      chan struct{}
	scalingGroupSet *SLockedSet
	scalingSql      *sqlchemy.SQuery
	// record the consecutive failures of scaling group's scale
	failRecord map[string]int
}

type SScalingInfo struct {
	ScalingGroup *models.SScalingGroup
	Total        int
}

type SLockedSet struct {
	set  sets.String
	lock sync.Mutex
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
	asc.timerQueue = make(chan struct{}, 20)
	asc.scalingQueue = make(chan struct{}, options.ConcurrentUpper)
	asc.scalingGroupSet = &SLockedSet{set: sets.NewString()}
	asc.failRecord = make(map[string]int)

	// init scalingSql
	sggQ := models.ScalingGroupGuestManager.Query("scaling_group_id").GroupBy("scaling_group_id")
	sggQ = sggQ.AppendField(sqlchemy.COUNT("total", sggQ.Field("guest_id")))
	sggSubQ := sggQ.SubQuery()
	sgQ := models.ScalingGroupManager.Query("id", "desire_instance_number").IsTrue("enabled")
	sgQ = sgQ.LeftJoin(sggSubQ, sqlchemy.AND(sqlchemy.Equals(sggSubQ.Field("scaling_group_id"),
		sgQ.Field("id")), sqlchemy.NotEquals(sggSubQ.Field("total"), sgQ.Field("desire_instance_number"))))
	sgQ.AppendField(sggSubQ.Field("total"))
	asc.scalingSql = sgQ

	cronm.AddJobAtIntervalsWithStartRun("CheckTimer", time.Duration(options.TimerInterval)*time.Second, asc.Timer, true)
	cronm.AddJobAtIntervalsWithStartRun("CheckScale", time.Duration(options.CheckScaleInterval)*time.Second, asc.CheckScale, true)
	cronm.AddJobAtIntervalsWithStartRun("CheckInstanceHealth", time.Duration(options.CheckHealthInterval)*time.Minute, asc.CheckInstanceHealth, true)

	// check all scaling activity
	nopanic.Run(func() {
		log.Infof("check and update scaling activities...")
		sas := make([]models.SScalingActivity, 0, 10)
		q := models.ScalingActivityManager.Query().Equals("status", compute.SA_STATUS_EXEC)
		err := db.FetchModelObjects(models.ScalingActivityManager, q, &sas)
		if err != nil {
			log.Errorf("unable to check and update scaling activities")
			return
		}
		for i := range sas {
			sas[i].SetFailed("", "As the service restarts, the status becomes unknown")
		}
		log.Infof("check and update scalngactivities complete")
	})
}

func (asc *SASController) PreScale(group *models.SScalingGroup, userCred mcclient.TokenCredential) bool {
	maxFailures := 3
	disableReason := fmt.Sprintf("The number of consecutive failures of creating a machine exceeds %d times", maxFailures)
	times := asc.failRecord[group.GetId()]
	if times >= maxFailures {
		_, err := db.Update(group, func() error {
			group.SetEnabled(false)
			return nil
		})
		if err != nil {
			return false
		}
		logclient.AddSimpleActionLog(group, logclient.ACT_DISABLE, disableReason, userCred, true)
		return false
	}
	return true
}

func (asc *SASController) Finish(groupId string, success bool) {
	asc.scalingGroupSet.Delete(groupId)
	if success {
		asc.failRecord[groupId] = 0
		return
	}
	asc.failRecord[groupId]++
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
		go asc.Scale(ctx, userCred, short)
	}
	// log.Debugf("This cronJob about CheckScale finished")
}

func (asc *SASController) Scale(ctx context.Context, userCred mcclient.TokenCredential, short SScalingGroupShort) {
	log.Debugf("scale for ScalingGroup '%s', desire: %d, total: %d", short.ID, short.DesireInstanceNumber, short.Total)
	var (
		err     error
		success = true
	)
	setFail := func(sa *models.SScalingActivity, reason string) {
		success = false
		err = sa.SetFailed("", reason)
	}
	defer func() {
		if err != nil {
			log.Errorf("Scaling for ScalingGroup '%s': %s", short.ID, err.Error())
		}
		asc.Finish(short.ID, success)
		<-asc.scalingQueue
		log.Debugf("Scale for ScalingGroup '%s' finished", short.ID)
	}()
	log.Debugf("fetch the latest data")
	// fetch the latest data
	model, err := models.ScalingGroupManager.FetchById(short.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			err = nil
		}
		return
	}
	sg := model.(*models.SScalingGroup)
	if !asc.PreScale(sg, userCred) {
		success = true
		return
	}
	total, err := sg.GuestNumber()
	if err != nil {
		return
	}
	log.Debugf("total: %d, desire: %d", total, sg.DesireInstanceNumber)

	// don't scale
	if sg.DesireInstanceNumber-total == 0 {
		return
	}

	log.Debugf("insert sa")

	scalingActivity, err := models.ScalingActivityManager.CreateScalingActivity(
		ctx,
		sg.Id,
		fmt.Sprintf(`The Desire Instance Number was changed, so change the Total Instance Number from "%d" to "%d"`,
			total, sg.DesireInstanceNumber,
		),
		compute.SA_STATUS_EXEC,
	)
	if err != nil {
		return
	}

	// userCred是管理员，ownerId是拥有者
	ownerId := sg.GetOwnerId()
	num := sg.DesireInstanceNumber - total
	switch {
	case num > 0:
		// check guest template
		gt := sg.GetGuestTemplate()
		if gt == nil {
			setFail(scalingActivity, fmt.Sprintf("fetch GuestTemplate of ScalingGroup '%s' error", sg.Id))
			return
		}
		nets, err := sg.NetworkIds()
		if err != nil {
			setFail(scalingActivity, fmt.Sprintf("fetch Networks of ScalingGroup '%s' error", sg.Id))
			return
		}
		valid, msg := gt.Validate(context.TODO(), auth.AdminCredential(), gt.GetOwnerId(),
			models.SGuestTemplateValidate{
				Hypervisor:    sg.Hypervisor,
				CloudregionId: sg.CloudregionId,
				VpcId:         sg.VpcId,
				NetworkIds:    nets,
			},
		)
		if !valid {
			err = scalingActivity.SetReject("", msg)
			return
		}
		if len(nets) == 0 {
			setFail(scalingActivity, fmt.Sprintf("empty networks '%s' error", sg.Id))
			return
		}
		succeedInstances, err := asc.CreateInstances(ctx, userCred, ownerId, sg, gt, nets[0], num)
		switch len(succeedInstances) {
		case 0:
			setFail(scalingActivity, fmt.Sprintf("All instances create failed: %s", err.Error()))
		case num:
			var action bytes.Buffer
			action.WriteString("Instances ")
			for _, instance := range succeedInstances {
				action.WriteString(fmt.Sprintf("'%s', ", instance.Name))
			}
			action.Truncate(action.Len() - 2)
			action.WriteString(" are created")
			err = scalingActivity.SetResult(action.String(), compute.SA_STATUS_SUCCEED, "", sg.DesireInstanceNumber)
		default:
			var action bytes.Buffer
			action.WriteString("Instances ")
			for _, instance := range succeedInstances {
				action.WriteString(fmt.Sprintf("'%s', ", instance.Name))
			}
			action.Truncate(action.Len() - 2)
			action.WriteString(" are created")
			err = scalingActivity.SetResult(action.String(), compute.SA_STATUS_PART_SUCCEED, fmt.Sprintf("Some instances create failed: %s", err.Error()), total+len(succeedInstances))
		}
		return
	case num < 0:
		num = -num
		succeedInstances, err := asc.DetachInstances(ctx, userCred, ownerId, sg, num)
		switch len(succeedInstances) {
		case 0:
			setFail(scalingActivity, fmt.Sprintf("All instance remove failed: %s", err.Error()))
		case num:
			var action bytes.Buffer
			action.WriteString("Instances ")
			for _, instance := range succeedInstances {
				action.WriteString(fmt.Sprintf("'%s', ", instance.Name))
			}
			action.Truncate(action.Len() - 2)
			action.WriteString(" are deleted")
			err = scalingActivity.SetResult(action.String(), compute.SA_STATUS_SUCCEED, "", sg.DesireInstanceNumber)
		default:
			var action bytes.Buffer
			action.WriteString("Instances ")
			for _, instance := range succeedInstances {
				action.WriteString(fmt.Sprintf("'%s', ", instance.Name))
			}
			action.Truncate(action.Len() - 2)
			action.WriteString(" are deleted")
			err = scalingActivity.SetResult(action.String(), compute.SA_STATUS_PART_SUCCEED, fmt.Sprintf("Some instance removed failed: %s", err.Error()), sg.DesireInstanceNumber)
		}
		return
	}
}

func (asc *SASController) DetachInstances(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, sg *models.SScalingGroup, num int) ([]SInstance, error) {
	instances, err := asc.findSuitableInstance(sg, num)
	if err != nil {
		return nil, errors.Wrap(err, "find suitable instances failed")
	}
	removeParams := jsonutils.NewDict()
	removeParams.Set("scaling_group", jsonutils.NewString(sg.Id))
	removeParams.Set("delete_server", jsonutils.JSONTrue)
	removeParams.Set("auto", jsonutils.JSONTrue)
	session := auth.GetSession(ctx, userCred, "")
	failedList := make([]string, 0)
	waitList := make([]string, 0, len(instances))
	instanceMap := make(map[string]SInstance, len(instances))
	// request to detach instances with scaling group
	for i := range instances {
		instanceMap[instances[i].Id] = SInstance{instances[i].Id, instances[i].Name}
		_, err := modules.Servers.PerformAction(session, instances[i].GetId(), "detach-scaling-group", removeParams)
		if err != nil {
			failedList = append(failedList, fmt.Sprintf("remove instance '%s' failed: %s", instances[i].GetId(), err.Error()))
			continue
		}
		waitList = append(waitList, instances[i].GetId())
	}
	// wait for all requests finished
	succeedList := sets.NewString(waitList...)
	ticker := time.NewTicker(3 * time.Second)
	timer := time.NewTimer(5 * time.Minute)
Loop:
	for {
		select {
		default:
			sggs, err := sg.ScalingGroupGuests(waitList)
			if err != nil {
				log.Errorf("ScalingGroup.ScalingGroupGuests error: %s", err.Error())
				<-ticker.C
				continue Loop
			}
			waitList = make([]string, 0, 1)
			for i := range sggs {
				switch sggs[i].GuestStatus {
				case compute.SG_GUEST_STATUS_REMOVE_FAILED:
					succeedList.Delete(sggs[i].GetId())
					failedList = append(failedList, fmt.Sprintf("remove instance '%s' failed", sggs[i].GetId()))
				case compute.SG_GUEST_STATUS_READY, compute.SG_GUEST_STATUS_REMOVING, compute.SG_GUEST_STATUS_PENDING_REMOVE:
					waitList = append(waitList, sggs[i].GetId())
				default:
					log.Errorf("unkown guest status for ScalingGroupGuest '%s'", sggs[i].GetId())
				}
			}
			if len(waitList) == 0 {
				break Loop
			}
			<-ticker.C
		case <-timer.C:
			log.Errorf("come check jobs for removing servers timeout")
			for _, id := range waitList {
				failedList = append(failedList, fmt.Sprintf("remove instance '%s' timeout", id))
				succeedList.Delete(id)
			}
		}
	}
	ticker.Stop()
	timer.Stop()
	log.Debugf("finish all check jobs when removing servers")
	err = nil
	if len(failedList) != 0 {
		err = fmt.Errorf(strings.Join(failedList, "; "))
	}
	instanceRet := make([]SInstance, 0, succeedList.Len())
	for _, id := range succeedList.UnsortedList() {
		instanceRet = append(instanceRet, instanceMap[id])
	}
	return instanceRet, err
}

func (asc *SASController) findSuitableInstance(sg *models.SScalingGroup, num int) ([]models.SGuest, error) {
	ggSubQ := models.ScalingGroupGuestManager.Query("guest_id").Equals("scaling_group_id", sg.Id).SubQuery()
	guestQ := models.GuestManager.Query().In("id", ggSubQ)
	switch sg.ShrinkPrinciple {
	case compute.SHRINK_EARLIEST_CREATION_FIRST:
		guestQ = guestQ.Asc("created_at").Limit(num)
	case compute.SHRINK_LATEST_CREATION_FIRST:
		guestQ = guestQ.Desc("created_at").Limit(num)
	}
	guests := make([]models.SGuest, 0, num)
	err := db.FetchModelObjects(models.GuestManager, guestQ, &guests)
	if err != nil {
		return nil, err
	}
	return guests, nil
}

func (asc *SASController) createInstances(session *mcclient.ClientSession, params jsonutils.JSONObject, count int,
	failedList []string, succeedList []SInstance) ([]string, []SInstance) {
	// forcast first
	dict := params.(*jsonutils.JSONDict)
	dict.Set("count", jsonutils.NewInt(int64(count)))
	_, err := scheduler.SchedManager.DoForecast(session, dict)
	if err != nil {
		clientErr := err.(*httputils.JSONClientError)
		failedList = append(failedList, clientErr.Details)
	}
	dict.Remove("domain_id")
	dict.Remove("count")

	if count == 1 {
		ret, err := modules.Servers.Create(session, params)
		if err != nil {
			clientErr := err.(*httputils.JSONClientError)
			failedList = append(failedList, clientErr.Details)
			return failedList, succeedList
		}
		id, _ := ret.GetString("id")
		name, _ := ret.GetString("name")
		succeedList = append(succeedList, SInstance{id, name})
		return failedList, succeedList
	}
	rets := modules.Servers.BatchCreate(session, params, count)
	for _, ret := range rets {
		if ret.Status >= 400 {
			failedList = append(failedList, ret.Data.String())
		} else {
			id, _ := ret.Data.GetString("id")
			name, _ := ret.Data.GetString("name")
			succeedList = append(succeedList, SInstance{id, name})
		}
	}
	return failedList, succeedList
}

type SInstance struct {
	ID   string
	Name string
}

func (asc *SASController) CreateInstances(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	sg *models.SScalingGroup,
	gt *models.SGuestTemplate,
	defaultNet string,
	num int,
) ([]SInstance, error) {
	// build the create request data
	content := gt.Content.(*jsonutils.JSONDict)
	// no network, add a default
	if len(gt.VpcId) == 0 {
		net := jsonutils.NewDict()
		net.Set("network", jsonutils.NewString(defaultNet))
		content.Set("nets", jsonutils.NewArray(net))
	}

	content.Set("auto_start", jsonutils.JSONTrue)
	// set description
	content.Set("description", jsonutils.NewString(fmt.Sprintf("Belong to scaling group '%s'", sg.Name)))
	// For compatibility
	content.Remove("__count__")
	// set onwer project and id
	content.Set("project_id", jsonutils.NewString(ownerId.GetProjectId()))
	content.Set("user_id", jsonutils.NewString(ownerId.GetUserId()))

	countPR, requests := asc.countPRAndRequests(num)
	log.Debugf("countPR: %d, requests: %d", countPR, requests)
	rand.Seed(time.Now().UnixNano())
	session := auth.GetSession(ctx, userCred, "")

	failedList := make([]string, 0)
	succeedList := make([]SInstance, 0, num/2)

	// fisrt stage: create request
	for rn := 0; rn < requests; rn++ {
		generateName := fmt.Sprintf("sg-%s-%s", sg.Name, asc.randStringRunes(5))
		content.Set("generate_name", jsonutils.NewString(generateName))
		failedList, succeedList = asc.createInstances(session, content, countPR, failedList, succeedList)
	}

	if remain := num - requests*countPR; remain > 0 {
		generateName := fmt.Sprintf("sg-%s-%s", sg.Name, asc.randStringRunes(5))
		content.Set("generate_name", jsonutils.NewString(generateName))
		failedList, succeedList = asc.createInstances(session, content, remain, failedList, succeedList)
	}

	// second stage: joining scaling group
	for _, instance := range succeedList {
		err := models.ScalingGroupGuestManager.Attach(ctx, sg.Id, instance.ID, false)
		if err != nil {
			log.Errorf("Attach ScalingGroup '%s' with Guest '%s' failed", sg.Id, instance.ID)
		}
	}
	// third stage: wait for create complete

	retChan := make(chan SCreateRet, requests)

	guestIds := make([]string, len(succeedList))
	instanceMap := make(map[string]SInstance, len(succeedList))
	for i, instane := range succeedList {
		guestIds[i] = instane.ID
		instanceMap[instane.ID] = instane
	}
	// check all server's status
	var waitLimit, waitinterval time.Duration
	if sg.Hypervisor == compute.HYPERVISOR_KVM {
		waitLimit = 5 * time.Minute
		waitinterval = 3 * time.Second
	} else {
		waitLimit = 10 * time.Minute
		waitinterval = 10 * time.Second
	}
	go asc.checkAllServer(session, guestIds, retChan, waitLimit, waitinterval)

	// fourth stage: bind lb and db
	failRecord := &SFailRecord{
		recordList: failedList,
	}
	succeedInstances := make([]string, 0, len(succeedList))
	workerLimit := make(chan struct{}, requests)
	for {
		ret, ok := <-retChan
		if !ok {
			break
		}
		workerLimit <- struct{}{}
		// bind ld and db
		go func() {
			succeed := asc.actionAfterCreate(ctx, userCred, session, sg, ret, failRecord)
			log.Debugf("action after create instance '%s', succeed: %t", ret.Id, succeed)
			if succeed {
				succeedInstances = append(succeedInstances, ret.Id)
			}
			<-workerLimit
		}()
	}
	log.Debugf("wait fo all worker finish")
	// wait for all worker finish
	log.Debugf("workerlimit cap: %d", cap(workerLimit))
	for i := 0; i < cap(workerLimit); i++ {
		log.Debugf("no.%d insert worker limit", i)
		workerLimit <- struct{}{}
	}

	instances := make([]SInstance, 0, len(succeedInstances))
	for _, id := range succeedInstances {
		instances = append(instances, instanceMap[id])
	}
	return instances, fmt.Errorf(failRecord.String())
}

type SCreateRet struct {
	Id     string
	Status string
}

func (asc *SASController) checkAllServer(session *mcclient.ClientSession, guestIds []string, retChan chan SCreateRet,
	waitLimit, waitInterval time.Duration) {
	guestIDSet := sets.NewString(guestIds...)
	timer := time.NewTimer(waitLimit)
	ticker := time.NewTicker(waitInterval)
	defer func() {
		close(retChan)
		ticker.Stop()
		timer.Stop()
		log.Debugf("finish all check jobs when creating servers")
	}()
	log.Debugf("guestIds: %s", guestIds)
	for {
		select {
		default:
			for _, id := range guestIDSet.UnsortedList() {
				ret, e := modules.Servers.GetSpecific(session, id, "status", nil)
				if e != nil {
					log.Errorf("Servers.GetSpecific failed: %s", e)
					<-ticker.C
					continue
				}
				log.Debugf("ret from GetSpecific: %s", ret.String())
				status, _ := ret.GetString("status")
				if status == compute.VM_RUNNING || strings.HasSuffix(status, "fail") || strings.HasSuffix(status, "failed") {
					guestIDSet.Delete(id)
					retChan <- SCreateRet{
						Id:     id,
						Status: status,
					}
				}
			}
			if guestIDSet.Len() == 0 {
				return
			}
			<-ticker.C
		case <-timer.C:
			log.Errorf("some check jobs for server timeout")
			for _, id := range guestIDSet.UnsortedList() {
				retChan <- SCreateRet{
					Id:     id,
					Status: "timeout",
				}
			}
			return
		}
	}
}

type SFailRecord struct {
	lock       sync.Mutex
	recordList []string
}

func (fr *SFailRecord) Append(record string) {
	fr.lock.Lock()
	defer fr.lock.Unlock()
	fr.recordList = append(fr.recordList, record)
}

func (fr *SFailRecord) String() string {
	return strings.Join(fr.recordList, "; ")
}

func (asc *SASController) actionAfterCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	session *mcclient.ClientSession,
	sg *models.SScalingGroup,
	ret SCreateRet,
	failRecord *SFailRecord,
) (succeed bool) {
	log.Debugf("start to action After create")
	deleteParams := jsonutils.NewDict()
	deleteParams.Set("override_pending_delete", jsonutils.JSONTrue)
	updateParams := jsonutils.NewDict()
	updateParams.Set("disable_delete", jsonutils.JSONFalse)
	rollback := func(failedReason string) {
		failRecord.Append(failedReason)
		// get scalingguest
		sggs, err := models.ScalingGroupGuestManager.Fetch(sg.GetId(), ret.Id)
		if err != nil || len(sggs) == 0 {
			log.Errorf("ScalingGroupGuestManager.Fetch failed: %s", err.Error())
			return
		}
		// cancel delete project
		_, e := modules.Servers.Update(session, ret.Id, updateParams)
		if err != nil {
			sggs[0].SetGuestStatus(compute.SG_GUEST_STATUS_READY)
			log.Errorf("cancel delete project of instance '%s' failed: %s", ret.Id, e.Error())
			return
		}
		// delete corresponding instance
		_, e = modules.Servers.Delete(session, ret.Id, deleteParams)
		if e != nil {
			// delete failed
			sggs[0].SetGuestStatus(compute.SG_GUEST_STATUS_READY)
			log.Errorf("delete instance '%s' failed: %s", ret.Id, e.Error())
			return
		}
		sggs[0].Detach(ctx, userCred)
		return
	}
	if ret.Status != compute.VM_RUNNING {
		if ret.Status == "timeout" {
			rollback(fmt.Sprintf("the creation process for instance '%s' has timed out", ret.Id))
		} else {
			// fetch the reason
			var reason string
			params := jsonutils.NewDict()
			params.Add(jsonutils.NewString(ret.Id), "obj_id")
			params.Add(jsonutils.NewStringArray([]string{db.ACT_ALLOCATE_FAIL}), "action")
			events, err := modules.Logs.List(session, params)
			if err != nil {
				log.Errorf("Logs.List failed: %s", err.Error())
				reason = fmt.Sprintf("instance '%s' which status is '%s' create failed", ret.Id, ret.Status)
			} else {
				switch events.Total {
				case 1:
					reason, _ = events.Data[0].GetString("notes")
				case 0:
					log.Errorf("These is no opslog about action '%s' for instance '%s", db.ACT_ALLOCATE_FAIL, ret.Id)
					reason = fmt.Sprintf("instance '%s' which status is '%s' create failed", ret.Id, ret.Status)
				default:
					log.Debugf("These are more than one optlogs about action '%s' for instance '%s'", db.ACT_ALLOCATE_FAIL, ret.Id)
					reason, _ = events.Data[0].GetString("notes")
				}
			}
			rollback(reason)
		}
		return
	}
	// bind lb
	if len(sg.BackendGroupId) != 0 {
		params := jsonutils.NewDict()
		params.Set("backend", jsonutils.NewString(ret.Id))
		params.Set("backend_type", jsonutils.NewString("guest"))
		params.Set("port", jsonutils.NewInt(int64(sg.LoadbalancerBackendPort)))
		params.Set("weight", jsonutils.NewInt(int64(sg.LoadbalancerBackendWeight)))
		params.Set("backend_group", jsonutils.NewString(sg.BackendGroupId))
		_, err := modules.LoadbalancerBackends.Create(session, params)
		if err != nil {
			rollback(fmt.Sprintf("bind instance '%s' to loadbalancer backend gropu '%s' failed: %s", ret.Id, sg.BackendGroupId, err.Error()))
		}
	}
	// todo bind bd

	// fifth stage: join scaling group finished
	sggs, err := models.ScalingGroupGuestManager.Fetch(sg.GetId(), ret.Id)
	if err != nil || sggs == nil || len(sggs) == 0 {
		log.Errorf("ScalingGroupGuestManager.Fetch failed; ScalingGroup '%s', Guest '%s'", sg.Id, ret.Id)
		return
	}
	sggs[0].SetGuestStatus(compute.SG_GUEST_STATUS_READY)
	return true
}

func (asc *SASController) randStringRunes(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz1234567890")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func (asc *SASController) countPRAndRequests(num int) (int, int) {
	key, tmp := 5, 1
	countPR, requests := 1, num
	if requests >= key {
		countPR := tmp * key
		requests = num / countPR
		tmp += 1
	}
	return countPR, requests
}

// ScalingGroupNeedScale will fetch all ScalingGroup need to scale
func (asc *SASController) ScalingGroupsNeedScale() ([]SScalingGroupShort, error) {
	rows, err := asc.scalingSql.Rows()
	if err != nil {
		return nil, errors.Wrap(err, "execute scaling sql error")
	}
	defer rows.Close()
	sgShorts := make([]SScalingGroupShort, 0, 10)
	for rows.Next() {
		sgPro := SScalingGroupShort{}
		err := asc.scalingSql.Row2Struct(rows, &sgPro)
		if err != nil {
			return nil, errors.Wrap(err, "sqlchemy.SQuery.Row2Struct error")
		}
		sgShorts = append(sgShorts, sgPro)
	}
	return sgShorts, nil
}
