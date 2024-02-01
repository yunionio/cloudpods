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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var containerManager *SContainerManager

func GetContainerManager() *SContainerManager {
	if containerManager == nil {
		containerManager = &SContainerManager{
			SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
				SContainer{},
				"containers_tbl",
				"container",
				"containers"),
		}
		containerManager.SetVirtualObject(containerManager)
	}
	return containerManager
}

func init() {
	GetContainerManager()
}

type SContainerManager struct {
	db.SVirtualResourceBaseManager
}

type SContainer struct {
	db.SVirtualResourceBase

	// GuestId is also the pod id
	GuestId string `width:"36" charset:"ascii" create:"required" list:"user" index:"true"`
	// Spec stores all container running options
	Spec *api.ContainerSpec `length:"long" create:"required" list:"user"`
}

func (m *SContainerManager) CreateOnPod(
	ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider,
	pod *SGuest, data *api.PodContainerCreateInput) (*SContainer, error) {
	input := &api.ContainerCreateInput{
		GuestId:  pod.GetId(),
		Spec:     data.ContainerSpec,
		SkipTask: true,
	}
	input.Name = data.Name
	obj, err := db.DoCreate(m, ctx, userCred, nil, jsonutils.Marshal(input), ownerId)
	if err != nil {
		return nil, errors.Wrap(err, "create container")
	}
	return obj.(*SContainer), nil
}

func (m *SContainerManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	guestId, _ := data.GetString("guest_id")
	return jsonutils.Marshal(map[string]string{"guest_id": guestId})
}

func (m *SContainerManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	guestId, _ := values.GetString("guest_id")
	if len(guestId) > 0 {
		q = q.Equals("guest_id", guestId)
	}
	return q
}

func (m *SContainerManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.ContainerListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (m *SContainerManager) GetContainersByPod(guestId string) ([]SContainer, error) {
	q := m.Query().Equals("guest_id", guestId)
	ctrs := make([]SContainer, 0)
	if err := db.FetchModelObjects(m, q, &ctrs); err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return ctrs, nil
}

func (m *SContainerManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, _ jsonutils.JSONObject, input *api.ContainerCreateInput) (*api.ContainerCreateInput, error) {
	if input.GuestId == "" {
		return nil, httperrors.NewNotEmptyError("guest_id is required")
	}
	obj, err := GuestManager.FetchByIdOrName(userCred, input.GuestId)
	if err != nil {
		return nil, errors.Wrapf(err, "fetch guest by %s", input.GuestId)
	}
	pod := obj.(*SGuest)
	input.GuestId = pod.GetId()
	if err := m.ValidateSpec(ctx, userCred, &input.Spec, pod); err != nil {
		return nil, errors.Wrap(err, "validate spec")
	}
	return input, nil
}

func (m *SContainerManager) ValidateSpec(ctx context.Context, userCred mcclient.TokenCredential, spec *api.ContainerSpec, pod *SGuest) error {
	if spec.ImagePullPolicy == "" {
		spec.ImagePullPolicy = apis.ImagePullPolicyIfNotPresent
	}
	if !sets.NewString(apis.ImagePullPolicyAlways, apis.ImagePullPolicyIfNotPresent).Has(string(spec.ImagePullPolicy)) {
		return httperrors.NewInputParameterError("invalid image_pull_policy %s", spec.ImagePullPolicy)
	}

	if pod != nil {
		if err := m.ValidateSpecVolumeMounts(ctx, userCred, pod, spec); err != nil {
			return errors.Wrap(err, "ValidateSpecVolumeMounts")
		}
		for idx, dev := range spec.Devices {
			newDev, err := m.ValidateSpecDevice(ctx, userCred, pod, dev)
			if err != nil {
				return errors.Wrapf(err, "validate device %s", jsonutils.Marshal(dev))
			}
			spec.Devices[idx] = newDev
		}
	}

	return nil
}

func (m *SContainerManager) ValidateSpecDevice(ctx context.Context, userCred mcclient.TokenCredential, pod *SGuest, dev *api.ContainerDevice) (*api.ContainerDevice, error) {
	drv, err := GetContainerDeviceDriverWithError(dev.Type)
	if err != nil {
		return nil, httperrors.NewInputParameterError("get device drvice: %v", err)
	}
	return drv.ValidateCreateData(ctx, userCred, pod, dev)
}

func (m *SContainerManager) ValidateSpecVolumeMounts(ctx context.Context, userCred mcclient.TokenCredential, pod *SGuest, spec *api.ContainerSpec) error {
	relation, err := m.GetVolumeMountRelations(ctx, userCred, pod, spec)
	if err != nil {
		return errors.Wrap(err, "GetVolumeMountRelations")
	}
	for idx, vm := range spec.VolumeMounts {
		newVm, err := m.ValidateSpecVolumeMount(ctx, userCred, pod, vm)
		if err != nil {
			return errors.Wrapf(err, "validate volume mount %s", jsonutils.Marshal(vm))
		}
		spec.VolumeMounts[idx] = newVm
	}
	if _, err := m.ConvertVolumeMountRelationToSpec(relation); err != nil {
		return errors.Wrap(err, "ConvertVolumeMountRelationToSpec")
	}
	return nil
}

func (m *SContainerManager) ValidateSpecVolumeMount(ctx context.Context, userCred mcclient.TokenCredential, pod *SGuest, vm *apis.ContainerVolumeMount) (*apis.ContainerVolumeMount, error) {
	if vm.Type == "" {
		return nil, httperrors.NewNotEmptyError("type is required")
	}
	if vm.MountPath == "" {
		return nil, httperrors.NewNotEmptyError("mount_path is required")
	}
	drv, err := GetContainerVolumeMountDriverWithError(vm.Type)
	if err != nil {
		return nil, errors.Wrapf(err, "get container volume mount driver %s", vm.Type)
	}
	vm, err = drv.ValidateCreateData(ctx, userCred, pod, vm)
	if err != nil {
		return nil, errors.Wrapf(err, "validate %s create data", vm.Type)
	}
	return vm, nil
}

/*func (m *SContainerManager) GetContainerIndex(guestId string) (int, error) {
	cnt, err := m.Query("guest_id").Equals("guest_id", guestId).CountWithError()
	if err != nil {
		return -1, errors.Wrapf(err, "get container numbers of pod %s", guestId)
	}
	return cnt, nil
}

func (c *SContainer) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	input := new(api.ContainerCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return errors.Wrap(err, "unmarshal to ContainerCreateInput")
	}
	if input.Spec.ImagePullPolicy == "" {
		c.Spec.ImagePullPolicy = apis.ImagePullPolicyIfNotPresent
	}
	return nil
}*/

func (c *SContainer) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	c.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	if !jsonutils.QueryBoolean(data, "skip_task", false) {
		if err := c.StartCreateTask(ctx, userCred, ""); err != nil {
			log.Errorf("StartCreateTask error: %v", err)
		}
	}
}

func (c *SContainer) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ContainerCreateTask", c, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (c *SContainer) GetPod() *SGuest {
	return GuestManager.FetchGuestById(c.GuestId)
}

func (c *SContainer) GetVolumeMounts() []*apis.ContainerVolumeMount {
	return c.Spec.VolumeMounts
}

type ContainerVolumeMountRelation struct {
	VolumeMount *apis.ContainerVolumeMount

	pod *SGuest
}

func (vm *ContainerVolumeMountRelation) ToHostMount() (*apis.ContainerVolumeMount, error) {
	return vm.VolumeMount, nil
}

func (m *SContainerManager) GetVolumeMountRelations(ctx context.Context, userCred mcclient.TokenCredential, pod *SGuest, spec *api.ContainerSpec) ([]*ContainerVolumeMountRelation, error) {
	relation := make([]*ContainerVolumeMountRelation, len(spec.VolumeMounts))
	for idx, vm := range spec.VolumeMounts {
		tmpVm := vm
		relation[idx] = &ContainerVolumeMountRelation{
			VolumeMount: tmpVm,
			pod:         pod,
		}
	}
	return relation, nil
}

func (c *SContainer) GetVolumeMountRelations(ctx context.Context, userCred mcclient.TokenCredential) ([]*ContainerVolumeMountRelation, error) {
	return GetContainerManager().GetVolumeMountRelations(ctx, userCred, c.GetPod(), c.Spec)
}

func (c *SContainer) PerformStart(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !sets.NewString(api.CONTAINER_STATUS_EXITED, api.CONTAINER_STATUS_START_FAILED).Has(c.Status) {
		return nil, httperrors.NewInvalidStatusError("Can't start container in status %s", c.Status)
	}
	return nil, c.StartStartTask(ctx, userCred, "")
}

func (c *SContainer) StartStartTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	c.SetStatus(userCred, api.CONTAINER_STATUS_STARTING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ContainerStartTask", c, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (c *SContainer) PerformStop(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *api.ContainerStopInput) (jsonutils.JSONObject, error) {
	if !sets.NewString(api.CONTAINER_STATUS_RUNNING, api.CONTAINER_STATUS_STOP_FAILED).Has(c.Status) {
		return nil, httperrors.NewInvalidStatusError("Can't stop container in status %s", c.Status)
	}
	return nil, c.StartStopTask(ctx, userCred, data, "")
}

func (c *SContainer) StartStopTask(ctx context.Context, userCred mcclient.TokenCredential, data *api.ContainerStopInput, parentTaskId string) error {
	c.SetStatus(userCred, api.CONTAINER_STATUS_STOPPING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ContainerStopTask", c, userCred, jsonutils.Marshal(data).(*jsonutils.JSONDict), parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (c *SContainer) StartSyncStatusTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	c.SetStatus(userCred, api.CONTAINER_STATUS_SYNC_STATUS, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ContainerSyncStatusTask", c, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (c *SContainer) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) error {
	return c.StartDeleteTask(ctx, userCred, "")
}

func (c *SContainer) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	c.SetStatus(userCred, api.CONTAINER_STATUS_DELETING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ContainerDeleteTask", c, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (c *SContainer) StartPullImageTask(ctx context.Context, userCred mcclient.TokenCredential, input *hostapi.ContainerPullImageInput, parentTaskId string) error {
	c.SetStatus(userCred, api.CONTAINER_STATUS_PULLING_IMAGE, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ContainerPullImageTask", c, userCred, jsonutils.Marshal(input).(*jsonutils.JSONDict), parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (c *SContainer) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return c.SVirtualResourceBase.Delete(ctx, userCred)
}

func (m *SContainerManager) ConvertVolumeMountRelationToSpec(relation []*ContainerVolumeMountRelation) ([]*apis.ContainerVolumeMount, error) {
	mounts := make([]*apis.ContainerVolumeMount, 0)
	for _, r := range relation {
		mount, err := r.ToHostMount()
		if err != nil {
			return nil, errors.Wrapf(err, "ToMountOrDevice: %#v", r)
		}
		if mount != nil {
			mounts = append(mounts, mount)
		}
	}
	return mounts, nil
}

func (c *SContainer) ToHostContainerSpec(ctx context.Context, userCred mcclient.TokenCredential) (*hostapi.ContainerSpec, error) {
	vmRelation, err := c.GetVolumeMountRelations(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "GetVolumeMountRelations")
	}
	mounts, err := GetContainerManager().ConvertVolumeMountRelationToSpec(vmRelation)
	if err != nil {
		return nil, errors.Wrap(err, "ConvertVolumeRelationToSpec")
	}

	spec := c.Spec.ContainerSpec
	spec.VolumeMounts = mounts
	hSpec := &hostapi.ContainerSpec{
		ContainerSpec: spec,
	}
	ctrDevs := make([]*hostapi.ContainerDevice, 0)
	for _, dev := range c.Spec.Devices {
		ctrDev, err := GetContainerDeviceDriver(dev.Type).ToHostDevice(dev)
		if err != nil {
			return nil, errors.Wrapf(err, "ToHostDevice %s", jsonutils.Marshal(dev))
		}
		ctrDevs = append(ctrDevs, ctrDev)
	}
	hSpec.Devices = ctrDevs
	return hSpec, nil
}

func (c *SContainer) GetJsonDescAtHost() *api.ContainerDesc {
	return &api.ContainerDesc{
		Id:   c.GetId(),
		Name: c.GetName(),
		Spec: c.Spec,
	}
}
