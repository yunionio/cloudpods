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
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SScalingGroupManager struct {
	db.SVirtualResourceBaseManager
}

type SScalingGroup struct {
	db.SVirtualResourceBase
	SCloudregionResourceBase

	Hypervisor        string `width:"16" charset:"ascii" default:"kvm" create:"required" list:"user" get:"user"`
	MinInstanceNumber int    `nullable:"false" default:"0" create:"required" list:"user" get:"user"`
	MaxInstanceNumber int    `nullable:"false" default:"10" create:"required" list:"user" get:"user"`

	// DesireInstanceNumber represent the number of instances that should exist in the scaling group.
	// Scaling controller will monitor and ensure this in real time.
	// Scaling activities triggered by various policies will also modify this value.
	// This value should between MinInstanceNumber and MaxInstanceNumber
	DesireInstanceNumber int    `nullable:"false" default:"0" create:"required" list:"user" get:"user"`
	GuestTemplateId      string `width:"128" charset:"ascii" create:"required" list:"user" get:"user"`

	// ExpansionPrinciple represent the principle when creating new instance to join in.
	ExpansionPrinciple string `width:"32" charset:"ascii" default:"balanced" create:"optional"`

	// ShrinkPrinciple represent the principle when removing instance from scaling group.
	ShrinkPrinciple string `width:"32" charset:"ascii" default:"earliest" create:"optional"`

	// GuestGroupId represent the guest gropu related to this scaling group.
	// Every scaling group will have only one guest group related to itself.
	GuestGroupId string `width:"128" charset:"ascii"`

	Enabled bool `nullable:"false" default:"true" list:"user" create:"optional"`
}

var ScalingGroupManager *SScalingGroupManager

func init() {
	ScalingGroupManager = &SScalingGroupManager{
		db.NewVirtualResourceBaseManager(
			SScalingGroup{},
			"scalinggroups_tbl",
			"scalinggroups",
			"scalinggropu",
		),
	}
	ScalingGroupManager.SetVirtualObject(ScalingGroupManager)
}

func (sgm *SScalingGroupManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.ScalingGroupCreateInput) (api.ScalingGroupCreateInput,
	error) {

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
	cloudregion, err := CloudregionManager.FetchByIdOrName(userCred, idOrName)
	if errors.Cause(err) == sql.ErrNoRows {
		return input, httperrors.NewInputParameterError("no such cloud region %s", idOrName)
	}
	input.CloudregionId = cloudregion.GetId()

	// check Guest Template
	idOrName = input.GuestTemplate
	if len(input.GuestTemplateId) != 0 {
		idOrName = input.GuestTemplateId
	}
	guestTemplate, err := GuestTemplateManager.FetchByIdOrName(userCred, idOrName)
	if errors.Cause(err) == sql.ErrNoRows {
		return input, httperrors.NewInputParameterError("no such guest template %s", idOrName)
	}
	if err != nil {
		return input, errors.Wrap(err, "GuestTempalteManager.FetchByIdOrName")
	}
	if !guestTemplate.(*SGuestTemplate).Validate(ctx, input.Hypervisor, input.CloudregionId) {
		return input, httperrors.NewInputParameterError("the guest template %s is not valid in cloudregion %s", idOrName,
			input.CloudregionId)
	}
	input.GuestTemplateId = guestTemplate.GetId()

	// check Expansion Principle
	if !utils.IsInStringArray(input.ExpansionPrinciple, []string{api.EXPANSION_BALANCED, ""}) {
		return input, httperrors.NewInputParameterError("unkown expansion principle %s", input.ExpansionPrinciple)
	}

	// check Shrink Principle
	if !utils.IsInStringArray(input.ShrinkPrinciple, []string{api.SHRINK_EARLIEST_CREATION_FIRST, api.SHRINK_LATEST_CREATION_FIRST,
		api.SHRINK_CONFIG_EARLIEST_CREATION_FIRST, api.SHRINK_CONFIG_EARLIEST_CREATION_FIRST, ""}) {
		return input, httperrors.NewInputParameterError("unkown shrink principle %s", input.ShrinkPrinciple)
	}

	return input, nil
}

func (sg *SScalingGroup) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {

	// todo: if group.Granularity is zero, how to do
	// create guest group related to this scaling group
	gp := SGroup{
		SVirtualResourceBase: sg.SVirtualResourceBase,
		ServiceType:          "",
		ParentId:             "",
		SchedStrategy:        "",
		Granularity:          0,
		ForceDispersion:      tristate.False,
		Enabled:              tristate.False,
	}
	// special description
	gp.Description = "Don't operate this group related to scaling group"
	// set this special group is a system resource
	gp.IsSystem = true
	err := GroupManager.TableSpec().Insert(&gp)
	if err != nil {
		return errors.Wrap(err, "sqlchemy.STableSpec.Insert")
	}
	sg.GuestGroupId = gp.GetId()
	sg.Status = api.SG_STATUS_READY
	return nil
}

// Delete group 的时候，把状态标识为deleting，然后让controller去删除
// 或者把policy也设置为deleting，然后自己通过task删除，set status要加锁
// 查询的就不用加锁，controller 碰到policy的状态为deleting, 会拒绝执行相关的
// 伸缩活动 过滤掉就好了; 对于正在进行的伸缩活动，需要等待其停止？
// 伸缩组不允许手动添加机器进来

func (sg *SScalingGroup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("SScaling Group delete do nothing")
	return nil
}

func (sg *SScalingGroup) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := db.DeleteModel(ctx, userCred, sg)
	if err == nil {
		return errors.Wrap(err, "db.DeleteModel")
	}
	sg.SetStatus(userCred, api.SG_STATUS_DELETED, "")
	return nil
}

func (sg *SScalingGroup) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return sg.StartScalingGroupDeleteTask(ctx, userCred)
}

func (sg *SScalingGroup) StartScalingGroupDeleteTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ScalingGroupDeleteTask", sg, userCred, nil, "", "", nil)
	if err == nil {
		return errors.Wrap(err, "Start ScalingGroupDeleteTask failed")
	}
	task.ScheduleRun(nil)
	return nil
}

func (sg *SScalingGroup) ScalingPolicies() ([]SScalingPolicy, error) {
	ret := make([]SScalingPolicy, 0)
	q := ScalingPolicyManager.Query().Equals("scaling_group_id", sg.Id)
	err := db.FetchModelObjects(ScalingPolicyManager, q, &ret)
	if err == nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return ret, nil
}

func (sg *SScalingGroupManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential, input api.ScalingGroupListInput) (*sqlchemy.SQuery, error) {

	if len(input.Cloudregion) != 0 {
		model, err := CloudregionManager.FetchByIdOrName(userCred, input.Cloudregion)
		if errors.Cause(err) == sql.ErrNoRows {
			return q, httperrors.NewInputParameterError("no such cloudregion %s", input.Cloudregion)
		}
		if err == nil {
			return q, errors.Wrap(err, "CloudregionManager.FetchByIdOrName")
		}
		q = q.Equals("cloudregion_id", model.GetId())
	}
	if len(input.Hypervisor) != 0 {

	}
	return nil, nil
}

func (sg *SScalingGroup) Guests() ([]SGuest, error) {
	q := GuestManager.Query().In("id", GroupguestManager.Query("guest_id").Equals("id", sg.GuestGroupId).SubQuery())
	guests := make([]SGuest, 0, 1)
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err == nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return guests, nil
}
