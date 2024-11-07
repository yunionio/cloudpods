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
	"database/sql"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=scalinggroup
// +onecloud:swagger-gen-model-plural=scalinggroups
type SScalingGroupManager struct {
	db.SVirtualResourceBaseManager
	SCloudregionResourceBaseManager
	SVpcResourceBaseManager
	SLoadbalancerBackendgroupResourceBaseManager
	SGroupResourceBaseManager
	SGuestTemplateResourceBaseManager
	SGuestResourceBaseManager
	db.SEnabledResourceBaseManager
}

type SScalingGroup struct {
	db.SVirtualResourceBase
	SCloudregionResourceBase
	SVpcResourceBase
	SLoadbalancerBackendgroupResourceBase
	// GuestGroupId represent the guest gropu related to this scaling group.
	// Every scaling group will have only one guest group related to itself.
	SGroupResourceBase
	SGuestTemplateResourceBase
	db.SEnabledResourceBase

	Hypervisor        string `width:"16" charset:"ascii" default:"kvm" create:"required" list:"user" get:"user" update:"user"`
	MinInstanceNumber int    `nullable:"false" default:"0" create:"required" list:"user" get:"user" update:"user"`
	MaxInstanceNumber int    `nullable:"false" default:"10" create:"required" list:"user" get:"user" update:"user"`

	// DesireInstanceNumber represent the number of instances that should exist in the scaling group.
	// Scaling controller will monitor and ensure this in real time.
	// Scaling activities triggered by various policies will also modify this value.
	// This value should between MinInstanceNumber and MaxInstanceNumber
	DesireInstanceNumber int `nullable:"false" default:"0" create:"required" list:"user" get:"user" update:"user"`

	// ExpansionPrinciple represent the principle when creating new instance to join in.
	ExpansionPrinciple string `width:"32" charset:"ascii" default:"balanced" create:"optional" list:"user" update:"user" get:"user"`

	// ShrinkPrinciple represent the principle when removing instance from scaling group.
	ShrinkPrinciple string `width:"32" charset:"ascii" default:"earliest" create:"optional" list:"user" update:"user" get:"user"`

	HealthCheckMode  string `width:"32" charset:"ascii" default:"normal" create:"optional" list:"user" update:"user" get:"user"`
	HealthCheckCycle int    `nullable:"false" default:"300" create:"optional" list:"user" update:"user" get:"user"`
	HealthCheckGov   int    `nullable:"false" default:"180" create:"optional" list:"user" update:"user" get:"user"`

	LoadbalancerBackendPort   int `nullable:"false" default:"80" create:"optional" list:"user" get:"user"`
	LoadbalancerBackendWeight int `nillable:"false" default:"1" create:"optional" list:"user" get:"user"`

	// Time to allow scale
	AllowScaleTime time.Time
	// NextCheckTime descripe the next time to check instance's health
	NextCheckTime time.Time
}

var ScalingGroupManager *SScalingGroupManager

func init() {
	ScalingGroupManager = &SScalingGroupManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SScalingGroup{},
			"scalinggroups_tbl",
			"scalinggroup",
			"scalinggroups",
		),
	}
	ScalingGroupManager.SetVirtualObject(ScalingGroupManager)
}

func (sgm *SScalingGroupManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.ScalingGroupCreateInput) (api.ScalingGroupCreateInput,
	error) {
	var err error
	input.VirtualResourceCreateInput, err = sgm.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred,
		ownerId, query, input.VirtualResourceCreateInput)
	// check InstanceNumber
	if input.MinInstanceNumber < 0 {
		return input, httperrors.NewInputParameterError("min_instance_number should not be smaller than 0")
	}
	if input.MinInstanceNumber > input.MaxInstanceNumber {
		return input, httperrors.NewInputParameterError(
			"min_instance_number should not be bigger than max_instance_number")
	}
	if input.DesireInstanceNumber < input.MinInstanceNumber || input.DesireInstanceNumber > input.MaxInstanceNumber {
		return input, httperrors.NewInputParameterError(
			"desire_instance_number should between min_instance_number and max_instance_number")
	}

	// check cloudregion
	idOrName := input.Cloudregion
	if len(input.CloudregionId) != 0 {
		idOrName = input.CloudregionId
	}
	cloudregion, err := CloudregionManager.FetchByIdOrName(ctx, userCred, idOrName)
	if errors.Cause(err) == sql.ErrNoRows {
		return input, httperrors.NewInputParameterError("no such cloud region %s", idOrName)
	}
	if err != nil {
		return input, errors.Wrap(err, "CloudregionManager.FetchByIdOrName")
	}
	input.CloudregionId = cloudregion.GetId()

	// check vpc
	_, err = validators.ValidateModel(ctx, userCred, VpcManager, &input.VpcId)
	if err != nil {
		return input, err
	}

	// check networks
	if len(input.Networks) == 0 {
		return input, httperrors.NewInputParameterError("ScalingGroup should have some networks")
	}
	networks := make([]SNetwork, 0, len(input.Networks))
	q := NetworkManager.Query()
	if len(input.Networks) == 1 {
		q = q.Filter(sqlchemy.OR(sqlchemy.Equals(q.Field("id"), input.Networks[0]), sqlchemy.Equals(q.Field("name"), input.Networks[0])))
	} else {
		q = q.Filter(sqlchemy.OR(sqlchemy.In(q.Field("id"), input.Networks), sqlchemy.In(q.Field("name"), input.Networks)))
	}
	err = db.FetchModelObjects(NetworkManager, q, &networks)
	if err != nil {
		return input, errors.Wrap(err, "db.FetchModelObjects")
	}
	if len(networks) != len(input.Networks) {
		return input, httperrors.NewInputParameterError("some networks not exist")
	}

	// check networks in vpc
	for i := range networks {
		vpc, _ := networks[i].GetVpc()
		if vpc == nil {
			return input, fmt.Errorf("Get vpc of network '%s' failed", networks[i].Id)
		}
		if vpc.Id != input.VpcId {
			return input, httperrors.NewInputParameterError("network '%s' not in vpc '%s'", networks[i].Id, input.VpcId)
		}
		input.Networks[i] = networks[i].Id
	}

	// check Guest Template
	idOrName = input.GuestTemplate
	if len(input.GuestTemplateId) != 0 {
		idOrName = input.GuestTemplateId
	}
	guestTemplate, err := GuestTemplateManager.FetchByIdOrName(ctx, userCred, idOrName)
	if errors.Cause(err) == sql.ErrNoRows {
		return input, httperrors.NewInputParameterError("no such guest template %s", idOrName)
	}
	if err != nil {
		return input, errors.Wrap(err, "GuestTempalteManager.FetchByIdOrName")
	}
	if ok, reason := guestTemplate.(*SGuestTemplate).Validate(ctx, userCred, ownerId,
		SGuestTemplateValidate{input.Hypervisor, input.CloudregionId, input.VpcId, input.Networks}); !ok {
		return input, httperrors.NewInputParameterError("the guest template %s is not valid in cloudregion %s, "+
			"reason: %s", idOrName, input.CloudregionId, reason)
	}
	input.GuestTemplateId = guestTemplate.GetId()

	// check Expansion Principle
	if !utils.IsInStringArray(input.ExpansionPrinciple, []string{api.EXPANSION_BALANCED, ""}) {
		return input, httperrors.NewInputParameterError("unkown expansion principle %s", input.ExpansionPrinciple)
	}

	// check Shrink Principle
	if !utils.IsInStringArray(input.ShrinkPrinciple, []string{api.SHRINK_EARLIEST_CREATION_FIRST, api.SHRINK_LATEST_CREATION_FIRST,
		api.SHRINK_CONFIG_EARLIEST_CREATION_FIRST, api.SHRINK_CONFIG_LATEST_CREATION_FIRST, ""}) {
		return input, httperrors.NewInputParameterError("unkown shrink principle %s", input.ShrinkPrinciple)
	}

	// check health check mod
	if !utils.IsInStringArray(input.HealthCheckMode, []string{api.HEALTH_CHECK_MODE_LOADBALANCER, api.HEALTH_CHECK_MODE_NORMAL, ""}) {
		return input, httperrors.NewInputParameterError("unkown health check mode %s", input.HealthCheckMode)
	}

	// check lb
	if len(input.LbBackendGroup) != 0 {
		idOrName = input.LbBackendGroup
		lb, err := LoadbalancerBackendGroupManager.FetchByIdOrName(ctx, userCred, idOrName)
		if errors.Cause(err) == sql.ErrNoRows {
			return input, httperrors.NewInputParameterError("no such loadbalancer backend group '%s'", idOrName)
		}
		if err != nil {
			return input, errors.Wrap(err, "LoadbalancerBackendGroupManager.FetchByIdOrName")
		}
		input.BackendGroupId = lb.GetId()

		// check lbeg port
		if input.LoadbalancerBackendPort < 1 || input.LoadbalancerBackendPort > 65535 {
			return input, httperrors.NewInputParameterError("invalid loadbalancer backend port '%d'", input.LoadbalancerBackendPort)
		}

		// check lbeg weight
		if input.LoadbalancerBackendWeight < 1 {
			return input, httperrors.NewInputParameterError("invalid loadbalancer backend weight '%d'", input.LoadbalancerBackendWeight)
		}
	}

	return input, nil
}

func (sg *SScalingGroup) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	// check enabled
	if sg.Enabled.IsTrue() {
		return httperrors.NewForbiddenError("Please disable this ScalingGroup firstly")
	}
	count, err := sg.GuestNumber()
	if err != nil {
		return errors.Wrap(err, "ScalingGroup.GuestNumber error")
	}
	if count != 0 {
		return httperrors.NewForbiddenError("There are some guests in this ScalingGroup, please delete them firstly")
	}
	return nil
}

func (sg *SScalingGroup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("SScaling Group delete do nothing")
	return nil
}

func (sg *SScalingGroup) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := db.DeleteModel(ctx, userCred, sg)
	if err != nil {
		return errors.Wrap(err, "db.DeleteModel")
	}
	sg.SetStatus(ctx, userCred, api.SG_STATUS_DELETED, "")
	return nil
}

func (sg *SScalingGroup) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return sg.StartScalingGroupDeleteTask(ctx, userCred)
}

func (sg *SScalingGroup) StartScalingGroupDeleteTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ScalingGroupDeleteTask", sg, userCred, nil, "", "", nil)
	if err != nil {
		return errors.Wrap(err, "Start ScalingGroupDeleteTask failed")
	}
	task.ScheduleRun(nil)
	return nil
}

func (sg *SScalingGroup) StartScalingGroupCreateTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ScalingGroupCreateTask", sg, userCred, nil, "", "", nil)
	if err != nil {
		return errors.Wrap(err, "Start ScalingGroupCreateTask failed")
	}
	task.ScheduleRun(nil)
	return nil
}

func (sg *SScalingGroup) ScalingPolicies() ([]SScalingPolicy, error) {
	ret := make([]SScalingPolicy, 0)
	q := ScalingPolicyManager.Query().Equals("scaling_group_id", sg.Id)
	err := db.FetchModelObjects(ScalingPolicyManager, q, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return ret, nil
}

func (sgm *SScalingGroupManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential, input api.ScalingGroupListInput) (*sqlchemy.SQuery, error) {

	// hack
	// vpc and backendgroup may be empty, and these subresoruce shouldn't be fiter by brand
	brand := input.Brand
	input.Brand = ""

	q, err := sgm.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, input.RegionalFilterListInput)
	if err != nil {
		return q, err
	}
	q, err = sgm.SVpcResourceBaseManager.ListItemFilter(ctx, q, userCred, input.VpcFilterListInput)
	if err != nil {
		return q, err
	}
	q, err = sgm.SLoadbalancerBackendgroupResourceBaseManager.ListItemFilter(ctx, q, userCred, input.LoadbalancerBackendGroupFilterListInput)
	if err != nil {
		return q, err
	}
	q, err = sgm.SGroupResourceBaseManager.ListItemFilter(ctx, q, userCred, input.GroupFilterListInput)
	if err != nil {
		return q, err
	}
	q, err = sgm.SGuestTemplateResourceBaseManager.ListItemFilter(ctx, q, userCred, input.GuestTemplateFilterListInput)
	if err != nil {
		return q, err
	}
	q, err = sgm.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, input.EnabledResourceBaseListInput)
	if err != nil {
		return q, err
	}
	if len(input.Hypervisor) > 0 {
		q = q.Equals("hypervisor", input.Hypervisor)
	}
	if len(brand) > 0 {
		q = q.Equals("hypervisor", Brand2Hypervisor(input.Brand))
	}
	input.Brand = brand
	return q, nil
}

func (sgm *SScalingGroupManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ScalingGroupDetails {
	virtRows := sgm.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	crRows := sgm.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	vpcRows := sgm.SVpcResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	lbbgRows := sgm.SLoadbalancerBackendgroupResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	groupRows := sgm.SGroupResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	gtRows := sgm.SGuestTemplateResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	rows := make([]api.ScalingGroupDetails, len(objs))
	for i := range rows {
		rows[i] = api.ScalingGroupDetails{
			VirtualResourceDetails:               virtRows[i],
			CloudregionResourceInfo:              crRows[i],
			LoadbalancerBackendGroupResourceInfo: lbbgRows[i],
			VpcResourceInfo:                      vpcRows[i],
			GroupResourceInfo:                    groupRows[i],
			GuestTemplateResourceInfo:            gtRows[i],
		}
		sg := objs[i].(*SScalingGroup)
		n, _ := sg.GuestNumber()
		rows[i].InstanceNumber = n
		n, _ = sg.ScalingPolicyNumber()
		rows[i].ScalingPolicyNumber = n
		rows[i].Brand = Hypervisor2Brand(sg.Hypervisor)
		nets, err := sg.Networks()
		if err != nil {
			log.Errorf("sg.Networks error: %s", err)
			continue
		}
		sgNets := make([]api.ScalingGroupNetwork, 0, len(nets))
		for i := range nets {
			sgNets = append(sgNets, api.ScalingGroupNetwork{
				Id:           nets[i].GetId(),
				Name:         nets[i].GetName(),
				GuestIpStart: nets[i].GuestIpStart,
				GuestIpEnd:   nets[i].GuestIpEnd,
			})
		}
		rows[i].Networks = sgNets
	}
	return rows
}

func (sg *SScalingGroup) GuestNumber() (int, error) {
	q := GuestManager.Query().In("id", ScalingGroupGuestManager.Query("guest_id").Equals("scaling_group_id",
		sg.Id).SubQuery()).IsFalse("pending_deleted")
	return q.CountWithError()
}

func (sg *SScalingGroup) ScalingPolicyNumber() (int, error) {
	q := ScalingPolicyManager.Query().Equals("scaling_group_id", sg.Id)
	return q.CountWithError()
}

/*func (sg *SScalingGroup) RemoveAllGuests(ctx context.Context, userCred mcclient.TokenCredential) error {
	q := ScalingGroupGuestManager.Query().Equals("scaling_group_id", sg.Id)
	scalingGroupGuests := make([]SScalingGroupGuest, 0)
	err := db.FetchModelObjects(ScalingGroupGuestManager, q, &scalingGroupGuests)
	if err != nil {
		return errors.Wrap(err, "db.FetchModelObjects")
	}
	for i := range scalingGroupGuests {
		err := scalingGroupGuests[i].Delete(ctx, userCred)
		return errors.Wrapf(err, "delete ScalingGroupGuests(ScalingGroupId: '%s', GuestId: '%s') failed",
			scalingGroupGuests[i].ScalingGroupId, scalingGroupGuests[i].GuestId)
	}
	return nil
}*/

func (sg *SScalingGroup) ScalingGroupGuests(guestIds []string) ([]SScalingGroupGuest, error) {
	q := ScalingGroupGuestManager.Query().Equals("scaling_group_id", sg.GetId())
	if len(guestIds) == 0 {
		return nil, nil
	}
	if len(guestIds) == 1 {
		q = q.Equals("guest_id", guestIds[0])
	}
	if len(guestIds) > 1 {
		q = q.In("guest_id", guestIds)
	}
	sggs := make([]SScalingGroupGuest, 0, len(guestIds))
	err := db.FetchModelObjects(ScalingGroupGuestManager, q, &sggs)
	return sggs, err
}

func (sg *SScalingGroup) Guests() ([]SGuest, error) {
	q := GuestManager.Query().In("id", ScalingGroupGuestManager.Query("guest_id").Equals("scaling_group_id", sg.Id).SubQuery())
	guests := make([]SGuest, 0, 1)
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return guests, nil
}

type sExecResult struct {
	// 0: success; 1: part success; 2: reject; 3: fail
	code       uint8
	actionStr  string
	reason     string
	intanceNum int
}

func (sg *SScalingGroup) exec(ctx context.Context, action IScalingAction) (ret sExecResult) {
	ret.code = 3
	ret.intanceNum = -1
	// query again to fetch the latest desire instance number of sg
	model, err := ScalingGroupManager.FetchById(sg.Id)
	if err != nil {
		ret.reason = fmt.Sprintf("fail to get ScalingGroup: %s", err.Error())
		return
	}
	sg = model.(*SScalingGroup)
	targetNum := action.Exec(sg.DesireInstanceNumber)
	// targetNum must between sg.MinInstanceNumber and sg.MaxInstanceNumber
	ret.code = 0
	if targetNum > sg.MaxInstanceNumber {
		if sg.DesireInstanceNumber == sg.MaxInstanceNumber {
			ret.code = 2
			ret.reason = fmt.Sprintf(
				`Want to change the Desired Instance Number from "%d" to "%d", but the Desired Instance Number has reached the Max Instance Number`,
				sg.DesireInstanceNumber, targetNum,
			)
			return
		}
		ret.code = 1
		ret.reason = fmt.Sprintf(
			`Want to change the Desired Instance Number from "%d" to "%d", but "%d" is greater than the Max Instance Number "%d"`,
			sg.DesireInstanceNumber, targetNum, targetNum, sg.MaxInstanceNumber,
		)
		targetNum = sg.MaxInstanceNumber
	} else if targetNum < sg.MinInstanceNumber {
		if sg.DesireInstanceNumber == sg.MinInstanceNumber {
			ret.code = 2
			ret.reason = fmt.Sprintf(
				`Want to change the Desired Instance Number from "%d" to "%d", but the Desired Instance Number has reached the Min Instance Number`,
				sg.DesireInstanceNumber, targetNum,
			)
			return
		}
		ret.code = 1
		ret.reason = fmt.Sprintf(
			`Want to change the Desired Instance Number from "%d" to "%d", but "%d" is less than the Min Instance Number "%d"`,
			sg.DesireInstanceNumber, targetNum, targetNum, sg.MinInstanceNumber,
		)
		targetNum = sg.MinInstanceNumber
	}
	ret.actionStr = fmt.Sprintf(`Change the Desired Instance Number from "%d" to "%d"`, sg.DesireInstanceNumber, targetNum)
	_, err = db.Update(sg, func() error {
		sg.DesireInstanceNumber = targetNum
		return nil
	})
	if err != nil {
		ret.code = 3
		ret.reason = fmt.Sprintf("Update ScalingGroup's DesireInstanceNumber failed: %s", err.Error())
	}
	return
}

var CoolingTimeLocation, _ = time.LoadLocation("Asia/Shanghai")

// Scale will modify SScalingGroup.DesireInstanceNumber and generate SScalingActivity based on the trigger and its
// corresponding SScalingPolicy.
func (sg *SScalingGroup) Scale(ctx context.Context, triggerDesc IScalingTriggerDesc, action IScalingAction,
	coolingTime int) error {
	lockman.LockObject(ctx, sg)
	defer lockman.ReleaseObject(ctx, sg)
	isExec := false
	defer func() {
		if isExec && coolingTime > 0 {
			sg.SetAllowScaleTime(time.Now().Add(time.Duration(coolingTime) * time.Second))
		}
	}()
	if sg.Enabled.IsFalse() {
		return nil
	}
	scalingActivity, err := ScalingActivityManager.CreateScalingActivity(ctx, sg.Id, triggerDesc.TriggerDescription(), api.SA_STATUS_EXEC)
	if err != nil {
		return errors.Wrapf(err, "create ScalingActivity whose ScalingGroup is %s error", sg.Id)
	}
	if action.CheckCoolTime() && !sg.AllowScale() {
		err = scalingActivity.SetReject("",
			fmt.Sprintf("The Cooling Time limit the execution time of the policy to at least: %s",
				sg.AllowScaleTime.In(CoolingTimeLocation).Format("2006-01-02 15:04:05 -0700")))
		return nil
	}

	ret := sg.exec(ctx, action)
	switch ret.code {
	case 0:
		err = scalingActivity.SetResult(ret.actionStr, api.SA_STATUS_SUCCEED, "", ret.intanceNum)
		isExec = true
	case 1:
		err = scalingActivity.SetResult(ret.actionStr, api.SA_STATUS_PART_SUCCEED, ret.reason, ret.intanceNum)
		isExec = true
	case 2:
		err = scalingActivity.SetReject("", ret.reason)
	case 3:
		err = scalingActivity.SetFailed(ret.actionStr, ret.reason)
	}

	if err != nil {
		log.Errorf("ScalingActivity set result failed: %s", err.Error())
	}
	return nil
}

func (sgm *SScalingGroupManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := sgm.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = sgm.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = sgm.SVpcResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = sgm.SLoadbalancerBackendgroupResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = sgm.SGroupResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = sgm.SGuestTemplateResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	scalinggroupGuests := ScalingGroupGuestManager.Query("scaling_group_id", "guest_id").SubQuery()
	q = q.LeftJoin(scalinggroupGuests, sqlchemy.Equals(q.Field("id"), scalinggroupGuests.Field("scaling_group_id")))
	q, err = sgm.SGuestResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SScalingGroupManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential, query api.ScalingGroupListInput) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, err
	}
	q, err = manager.SVpcResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VpcFilterListInput)
	if err != nil {
		return nil, err
	}
	q, err = manager.SLoadbalancerBackendgroupResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.LoadbalancerBackendGroupFilterListInput)
	if err != nil {
		return nil, err
	}
	q, err = manager.SGroupResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.GroupFilterListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (sg *SScalingGroup) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	// attach with networks
	networks, _ := data.Get("networks")
	networkIds := networks.(*jsonutils.JSONArray).GetStringArray()
	for _, netId := range networkIds {
		err := ScalingGroupNetworkManager.Attach(ctx, sg.Id, netId)
		if err != nil {
			reason := fmt.Sprintf("Attach ScalingGroup '%s' with Network '%s' failed: %s", sg.Id, netId, err.Error())
			sg.SetStatus(ctx, userCred, api.SG_STATUS_CREATE_FAILED, reason)
			logclient.AddActionLogWithContext(ctx, sg, logclient.ACT_CREATE, reason, userCred, false)
			return
		}
	}
	now := time.Now()
	db.Update(sg, func() error {
		sg.Status = api.SG_STATUS_READY
		sg.AllowScaleTime = now
		sg.NextCheckTime = now.Add(time.Duration(sg.HealthCheckCycle) * time.Second)
		sg.SetEnabled(true)
		return nil
	})
	logclient.AddActionLogWithContext(ctx, sg, logclient.ACT_CREATE, "", userCred, true)
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    sg,
		Action: notifyclient.ActionCreate,
	})
}

func (sg *SScalingGroup) AllowScale() bool {
	return sg.AllowScaleTime.Before(time.Now())
}

func (sg *SScalingGroup) SetAllowScaleTime(t time.Time) {
	if sg.AllowScaleTime.After(t) {
		return
	}
	_, err := db.Update(sg, func() error {
		sg.AllowScaleTime = t
		return nil
	})
	if err != nil {
		log.Errorf("Set AllowScaleTime error: %s", err)
	}
	return
}

func (sg *SScalingGroup) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(sg, ctx, userCred, true)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (sg *SScalingGroup) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(sg, ctx, userCred, false)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (s *SGuest) PerformDetachScalingGroup(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, input api.SGPerformDetachScalingGroupInput) (jsonutils.JSONObject, error) {
	// check ScalingGroup
	model, err := ScalingGroupManager.FetchByIdOrName(ctx, userCred, input.ScalingGroup)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewInputParameterError("no such ScalingGroup '%s'", input.ScalingGroup)
		}
		return nil, errors.Wrap(err, "ScalingGroupManager.FetchByIdOrName")
	}
	// check ScalingGroupGuest
	sggs, err := ScalingGroupGuestManager.Fetch(model.GetId(), s.Id)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewInputParameterError("Guest '%s' don't belong to ScalingGroup '%s'", s.Id, model.GetId())
		}
		return nil, errors.Wrap(err, "ScalingGroupGuestManager.Fetch")
	}
	if len(sggs) == 0 {
		return nil, httperrors.NewInputParameterError("Guest '%s' don't belong to ScalingGroup '%s'", s.Id, model.GetId())
	}
	sg := model.(*SScalingGroup)
	input.ScalingGroup = sggs[0].ScalingGroupId
	sggs[0].SetGuestStatus(api.SG_GUEST_STATUS_REMOVING)
	taskData := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	taskData.Set("guest", jsonutils.NewString(s.GetId()))
	task, err := taskman.TaskManager.NewTask(ctx, "GuestDetachScalingGroupTask", sg, userCred, taskData, "", "")
	if err != nil {
		return nil, errors.Wrap(err, "Start GuestDetachScalingGroupTask failed")
	}
	task.ScheduleRun(nil)
	return nil, nil
}

func (sg *SScalingGroup) Networks() ([]SNetwork, error) {
	nets := make([]SNetwork, 0, 1)
	sgnQuery := ScalingGroupNetworkManager.Query("network_id").Equals("scaling_group_id", sg.Id).SubQuery()
	netQuery := NetworkManager.Query().In("id", sgnQuery)

	err := db.FetchModelObjects(NetworkManager, netQuery, &nets)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return nets, nil
}

func (sg *SScalingGroup) NetworkIds() ([]string, error) {
	sgnQuery := ScalingGroupNetworkManager.Query("network_id").Equals("scaling_group_id", sg.Id)
	rows, err := sgnQuery.Rows()
	if err != nil {
		return nil, errors.Wrap(err, "SQuery.Rows")
	}
	defer rows.Close()
	nets := make([]string, 0, 1)
	for rows.Next() {
		var net string
		rows.Scan(&net)
		nets = append(nets, net)
	}
	return nets, nil
}

func (sg *SScalingGroup) Activities() ([]SScalingActivity, error) {
	q := ScalingActivityManager.Query().Equals("scaling_group_id", sg.Id)
	activities := make([]SScalingActivity, 0, 1)
	err := db.FetchModelObjects(ScalingActivityManager, q, &activities)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return activities, nil
}

func (manager *SScalingGroupManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SVpcResourceBaseManager.GetExportKeys()...) {
		// SCloudregionResourceBaseManager
		q, err = manager.SVpcResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SLoadbalancerBackendgroupResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SLoadbalancerBackendgroupResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SLoadbalancerBackendgroupResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SGroupResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SGroupResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SGroupResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SGuestTemplateResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SGuestTemplateResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SGuestTemplateResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}
