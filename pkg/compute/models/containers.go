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
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	identitymod "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	kubemod "yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
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
	Spec *api.ContainerSpec `length:"long" create:"required" list:"user" update:"user"`

	// 启动时间
	StartedAt time.Time `nullable:"true" created_at:"false" index:"true" get:"user" list:"user" json:"started_at"`
	// 上次退出时间
	LastFinishedAt time.Time `nullable:"true" created_at:"false" index:"true" get:"user" list:"user" json:"last_finished_at"`

	// 重启次数
	RestartCount int `nullable:"true" list:"user"`
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
	if query.GuestId != "" {
		gst, err := GuestManager.FetchByIdOrName(ctx, userCred, query.GuestId)
		if err != nil {
			return nil, errors.Wrapf(err, "fetch guest by %s", query.GuestId)
		}
		q = q.Equals("guest_id", gst.GetId())
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
	obj, err := GuestManager.FetchByIdOrName(ctx, userCred, input.GuestId)
	if err != nil {
		return nil, errors.Wrapf(err, "fetch guest by %s", input.GuestId)
	}
	pod := obj.(*SGuest)
	input.GuestId = pod.GetId()
	if err := m.ValidateSpec(ctx, userCred, &input.Spec, pod, nil); err != nil {
		return nil, errors.Wrap(err, "validate spec")
	}
	return input, nil
}

func (m *SContainerManager) ValidateSpec(ctx context.Context, userCred mcclient.TokenCredential, spec *api.ContainerSpec, pod *SGuest, ctr *SContainer) error {
	if spec.ImagePullPolicy == "" {
		spec.ImagePullPolicy = apis.ImagePullPolicyIfNotPresent
	}
	if !sets.NewString(apis.ImagePullPolicyAlways, apis.ImagePullPolicyIfNotPresent).Has(string(spec.ImagePullPolicy)) {
		return httperrors.NewInputParameterError("invalid image_pull_policy %s", spec.ImagePullPolicy)
	}
	if spec.ImageCredentialId != "" {
		if _, err := m.GetImageCredential(ctx, userCred, spec.ImageCredentialId); err != nil {
			return errors.Wrapf(err, "get image credential by id: %s", spec.ImageCredentialId)
		}
	}

	if pod != nil {
		if err := m.ValidateSpecVolumeMounts(ctx, userCred, pod, spec, ctr); err != nil {
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

	if err := m.ValidateSpecLifecycle(ctx, userCred, spec); err != nil {
		return errors.Wrap(err, "validate lifecycle")
	}

	if spec.ShmSizeMB != 0 && spec.ShmSizeMB < 64 {
		return httperrors.NewInputParameterError("/dev/shm size is small than 64MB")
	}

	if err := m.ValidateSpecProbe(ctx, userCred, spec); err != nil {
		return errors.Wrap(err, "validate probe configuration")
	}

	return nil
}

func (m *SContainerManager) ValidateSpecLifecycle(ctx context.Context, cred mcclient.TokenCredential, spec *api.ContainerSpec) error {
	if spec.Lifecyle == nil {
		return nil
	}
	if err := m.ValidateSpecLifecyclePostStart(ctx, cred, spec.Lifecyle.PostStart); err != nil {
		return errors.Wrap(err, "validate post start")
	}
	return nil
}

func (m *SContainerManager) ValidateSpecLifecyclePostStart(ctx context.Context, userCred mcclient.TokenCredential, input *apis.ContainerLifecyleHandler) error {
	drv, err := GetContainerLifecyleDriverWithError(input.Type)
	if err != nil {
		return httperrors.NewInputParameterError("get lifecycle driver: %v", err)
	}
	return drv.ValidateCreateData(ctx, userCred, input)
}

func (m *SContainerManager) ValidateSpecDevice(ctx context.Context, userCred mcclient.TokenCredential, pod *SGuest, dev *api.ContainerDevice) (*api.ContainerDevice, error) {
	drv, err := GetContainerDeviceDriverWithError(dev.Type)
	if err != nil {
		return nil, httperrors.NewInputParameterError("get device driver: %v", err)
	}
	return drv.ValidateCreateData(ctx, userCred, pod, dev)
}

func (m *SContainerManager) ValidateSpecVolumeMounts(ctx context.Context, userCred mcclient.TokenCredential, pod *SGuest, spec *api.ContainerSpec, ctr *SContainer) error {
	relation, err := m.GetVolumeMountRelations(pod, spec)
	if err != nil {
		return errors.Wrap(err, "GetVolumeMountRelations")
	}

	curCtrs, _ := m.GetContainersByPod(pod.GetId())
	volUniqNames := sets.NewString()
	for idx := range curCtrs {
		if ctr != nil && ctr.GetId() == curCtrs[idx].GetId() {
			continue
		}
		for _, vol := range curCtrs[idx].Spec.VolumeMounts {
			if vol.UniqueName != "" {
				volUniqNames.Insert(vol.UniqueName)
			}
		}
	}
	for idx, vm := range spec.VolumeMounts {
		if vm.UniqueName != "" {
			if volUniqNames.Has(vm.UniqueName) {
				return httperrors.NewDuplicateNameError("volume_mount unique_name %s", fmt.Sprintf("%s: %s, index: %d", vm.UniqueName, jsonutils.Marshal(vm), idx))
			} else {
				volUniqNames.Insert(vm.UniqueName)
			}
		}
		newVm, err := m.ValidateSpecVolumeMount(ctx, userCred, pod, vm)
		if err != nil {
			return errors.Wrapf(err, "validate volume mount %s", jsonutils.Marshal(vm))
		}
		spec.VolumeMounts[idx] = newVm
	}
	if _, err := m.ConvertVolumeMountRelationToSpec(ctx, userCred, relation); err != nil {
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
		return nil, errors.Wrapf(err, "validate %s create data", drv.GetType())
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

func (m *SContainerManager) ValidateSpecProbe(ctx context.Context, userCred mcclient.TokenCredential, spec *api.ContainerSpec) error {
	//if err := m.validateSpecProbe(ctx, userCred, spec.LivenessProbe); err != nil {
	//	return errors.Wrap(err, "validate liveness probe")
	//}
	if err := m.validateSpecProbe(ctx, userCred, spec.StartupProbe); err != nil {
		return errors.Wrap(err, "validate startup probe")
	}
	return nil
}

func (m *SContainerManager) validateSpecProbe(ctx context.Context, userCred mcclient.TokenCredential, probe *apis.ContainerProbe) error {
	if probe == nil {
		return nil
	}
	if err := m.validateSpecProbeHandler(probe.ContainerProbeHandler); err != nil {
		return errors.Wrap(err, "validate container probe handler")
	}
	for key, val := range map[string]int32{
		//"initial_delay_seconds": probe.InitialDelaySeconds,
		"timeout_seconds":   probe.TimeoutSeconds,
		"period_seconds":    probe.PeriodSeconds,
		"success_threshold": probe.SuccessThreshold,
		"failure_threshold": probe.FailureThreshold,
	} {
		if val < 0 {
			return httperrors.NewInputParameterError(key + " is negative")
		}
	}

	//if probe.InitialDelaySeconds == 0 {
	//	probe.InitialDelaySeconds = 5
	//}
	if probe.TimeoutSeconds == 0 {
		probe.TimeoutSeconds = 3
	}
	if probe.PeriodSeconds == 0 {
		probe.PeriodSeconds = 10
	}
	if probe.SuccessThreshold == 0 {
		probe.SuccessThreshold = 1
	}
	if probe.FailureThreshold == 0 {
		probe.FailureThreshold = 3
	}
	return nil
}

func (m *SContainerManager) validateSpecProbeHandler(probe apis.ContainerProbeHandler) error {
	isAllNil := true
	if probe.Exec != nil {
		isAllNil = false
		if len(probe.Exec.Command) == 0 {
			return httperrors.NewInputParameterError("exec command is required")
		}
	}
	if probe.TCPSocket != nil {
		isAllNil = false
		port := probe.TCPSocket.Port
		if port < 1 || port > 65535 {
			return httperrors.NewInputParameterError("invalid tcp socket port: %d, must between [1,65535]", port)
		}
	}
	if probe.HTTPGet != nil {
		isAllNil = false
		port := probe.HTTPGet.Port
		if port < 1 || port > 65535 {
			return httperrors.NewInputParameterError("invalid http port: %d, must between [1,65535]", port)
		}
	}
	if isAllNil {
		return httperrors.NewInputParameterError("one of [exec, http_get, tcp_socket] is required")
	}
	return nil
}

func (m *SContainerManager) startBatchTask(ctx context.Context, userCred mcclient.TokenCredential, taskName string, ctrs []SContainer, taskData *jsonutils.JSONDict, parentTaskId string) error {
	ctrPtrs := make([]db.IStandaloneModel, len(ctrs))
	for i := range ctrs {
		ctrPtrs[i] = &ctrs[i]
	}
	task, err := taskman.TaskManager.NewParallelTask(ctx, taskName, ctrPtrs, userCred, taskData, parentTaskId, "")
	if err != nil {
		return errors.Wrapf(err, "NewParallelTask %s", taskName)
	}
	return task.ScheduleRun(nil)
}

func (m *SContainerManager) StartBatchStartTask(ctx context.Context, userCred mcclient.TokenCredential, ctrs []SContainer, parentTaskId string) error {
	return m.startBatchTask(ctx, userCred, "ContainerBatchStartTask", ctrs, nil, parentTaskId)
}

func (m *SContainerManager) StartBatchStopTask(ctx context.Context, userCred mcclient.TokenCredential, ctrs []SContainer, timeout int, force bool, parentTaskId string) error {
	params := make([]api.ContainerStopInput, len(ctrs))
	for i := range ctrs {
		params[i] = api.ContainerStopInput{
			Timeout: timeout,
			Force:   force,
		}
	}
	taskParams := jsonutils.NewDict()
	taskParams.Add(jsonutils.Marshal(params), "params")

	return m.startBatchTask(ctx, userCred, "ContainerBatchStopTask", ctrs, taskParams, parentTaskId)
}

func (c *SContainer) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	c.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	if !jsonutils.QueryBoolean(data, "skip_task", false) {
		if err := c.StartCreateTask(ctx, userCred, "", nil); err != nil {
			log.Errorf("StartCreateTask error: %v", err)
		}
	}
}

func (c *SContainer) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, params *jsonutils.JSONDict) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ContainerCreateTask", c, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (c *SContainer) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.ContainerUpdateInput) (*api.ContainerUpdateInput, error) {
	if !api.ContainerExitedStatus.Has(c.GetStatus()) {
		return nil, httperrors.NewInvalidStatusError("current status %s is not in %v", c.GetStatus(), api.ContainerExitedStatus.List())
	}

	baseInput, err := c.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.VirtualResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SVirtualResourceBase.ValidateUpdateData")
	}
	input.VirtualResourceBaseUpdateInput = baseInput

	if err := GetContainerManager().ValidateSpec(ctx, userCred, &input.Spec, c.GetPod(), c); err != nil {
		return nil, errors.Wrap(err, "validate spec")
	}

	return input, nil
}

func (c *SContainer) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	c.SVirtualResourceBase.PostUpdate(ctx, userCred, query, data)
	if err := c.GetPod().StartSyncTaskWithoutSyncstatus(ctx, userCred, false, ""); err != nil {
		log.Errorf("container %s StartSyncTaskWithoutSyncstatus error: %v", c.GetName(), err)
	}
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

func (vm *ContainerVolumeMountRelation) toHostDiskMount(disk *apis.ContainerVolumeMountDisk) (*hostapi.ContainerVolumeMountDisk, error) {
	diskObj := DiskManager.FetchDiskById(disk.Id)
	if diskObj == nil {
		return nil, errors.Errorf("fetch disk by id %s", disk.Id)
	}
	ret := &hostapi.ContainerVolumeMountDisk{
		Index:                disk.Index,
		Id:                   disk.Id,
		TemplateId:           diskObj.TemplateId,
		SubDirectory:         disk.SubDirectory,
		StorageSizeFile:      disk.StorageSizeFile,
		Overlay:              disk.Overlay,
		CaseInsensitivePaths: disk.CaseInsensitivePaths,
		PostOverlay:          disk.PostOverlay,
	}
	return ret, nil
}

func (vm *ContainerVolumeMountRelation) toCephFSMount(fs *apis.ContainerVolumeMountCephFS) (*hostapi.ContainerVolumeMountCephFS, error) {
	fsObj, err := FileSystemManager.FetchById(fs.Id)
	if err != nil {
		return nil, errors.Errorf("fetch cephfs by id %s", fs.Id)
	}

	filesystem := fsObj.(*SFileSystem)
	ret := &hostapi.ContainerVolumeMountCephFS{
		Id:   filesystem.Id,
		Path: filesystem.ExternalId,
	}

	account := filesystem.GetCloudaccount()
	if gotypes.IsNil(account) {
		return nil, fmt.Errorf("invalid cephfs %s", filesystem.Name)
	}
	ret.Secret, err = account.GetOptionPassword()
	if err != nil {
		return nil, err
	}
	ret.Name, _ = account.Options.GetString("name")
	if len(ret.Name) == 0 {
		ret.Name = "admin"
	}
	ret.MonHost, _ = account.Options.GetString("mon_host")
	return ret, nil
}

func (vm *ContainerVolumeMountRelation) ToHostMount(ctx context.Context, userCred mcclient.TokenCredential) (*hostapi.ContainerVolumeMount, error) {
	ret := &hostapi.ContainerVolumeMount{
		UniqueName:     vm.VolumeMount.UniqueName,
		Type:           vm.VolumeMount.Type,
		Disk:           nil,
		CephFS:         nil,
		Text:           vm.VolumeMount.Text,
		HostPath:       vm.VolumeMount.HostPath,
		ReadOnly:       vm.VolumeMount.ReadOnly,
		MountPath:      vm.VolumeMount.MountPath,
		SelinuxRelabel: vm.VolumeMount.SelinuxRelabel,
		Propagation:    vm.VolumeMount.Propagation,
		FsUser:         vm.VolumeMount.FsUser,
		FsGroup:        vm.VolumeMount.FsGroup,
	}
	if vm.VolumeMount.Disk != nil {
		disk, err := vm.toHostDiskMount(vm.VolumeMount.Disk)
		if err != nil {
			return nil, errors.Wrap(err, "toHostDiskMount")
		}
		ret.Disk = disk
	}
	if vm.VolumeMount.CephFS != nil {
		fs, err := vm.toCephFSMount(vm.VolumeMount.CephFS)
		if err != nil {
			return nil, errors.Wrapf(err, "getCephFSSecret")
		}
		ret.CephFS = fs
	}
	return ret, nil
}

func (m *SContainerManager) GetVolumeMountRelations(pod *SGuest, spec *api.ContainerSpec) ([]*ContainerVolumeMountRelation, error) {
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

func (c *SContainer) GetVolumeMountRelations() ([]*ContainerVolumeMountRelation, error) {
	return GetContainerManager().GetVolumeMountRelations(c.GetPod(), c.Spec)
}

func (c *SContainer) PerformStart(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !sets.NewString(api.CONTAINER_STATUS_EXITED, api.CONTAINER_STATUS_START_FAILED).Has(c.Status) {
		return nil, httperrors.NewInvalidStatusError("Can't start container in status %s", c.Status)
	}
	return nil, c.StartStartTask(ctx, userCred, "")
}

func (c *SContainer) StartStartTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	c.SetStatus(ctx, userCred, api.CONTAINER_STATUS_STARTING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ContainerStartTask", c, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (c *SContainer) PerformStop(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *api.ContainerStopInput) (jsonutils.JSONObject, error) {
	if !data.Force {
		if !sets.NewString(
			api.CONTAINER_STATUS_RUNNING,
			api.CONTAINER_STATUS_PROBING,
			api.CONTAINER_STATUS_STOP_FAILED).Has(c.Status) {
			return nil, httperrors.NewInvalidStatusError("Can't stop container in status %s", c.Status)
		}
	}
	return nil, c.StartStopTask(ctx, userCred, data, "")
}

func (c *SContainer) StartStopTask(ctx context.Context, userCred mcclient.TokenCredential, data *api.ContainerStopInput, parentTaskId string) error {
	c.SetStatus(ctx, userCred, api.CONTAINER_STATUS_STOPPING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ContainerStopTask", c, userCred, jsonutils.Marshal(data).(*jsonutils.JSONDict), parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (c *SContainer) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, c.StartSyncStatusTask(ctx, userCred, "")
}

func (c *SContainer) StartSyncStatusTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	c.SetStatus(ctx, userCred, api.CONTAINER_STATUS_SYNC_STATUS, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ContainerSyncStatusTask", c, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (c *SContainer) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) error {
	return c.StartDeleteTask(ctx, userCred, "", jsonutils.QueryBoolean(data, "purge", false))
}

func (c *SContainer) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, purge bool) error {
	c.SetStatus(ctx, userCred, api.CONTAINER_STATUS_DELETING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ContainerDeleteTask", c, userCred, jsonutils.Marshal(map[string]interface{}{
		"purge": purge,
	}).(*jsonutils.JSONDict), parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (m *SContainerManager) GetImageCredential(ctx context.Context, userCred mcclient.TokenCredential, id string) (*apis.ContainerPullImageAuthConfig, error) {
	s := auth.GetSession(ctx, userCred, options.Options.Region)
	ret, err := identitymod.Credentials.GetById(s, id, nil)
	if err != nil {
		if errors.Cause(err) == errors.ErrNotFound || strings.Contains(err.Error(), "NotFound") {
			ret, err = identitymod.Credentials.GetByName(s, id, nil)
			if err != nil {
				return nil, errors.Wrapf(err, "get credential by id or name of %s", id)
			}
		}
		return nil, errors.Wrap(err, "get credentials by id")
	}
	credType, _ := ret.GetString("type")
	if credType != identityapi.CONTAINER_IMAGE_TYPE {
		return nil, httperrors.NewNotSupportedError("unsupported credential type %s", credType)
	}
	blobStr, err := ret.GetString("blob")
	if err != nil {
		return nil, errors.Wrap(err, "get blob")
	}
	obj, err := jsonutils.ParseString(blobStr)
	if err != nil {
		return nil, errors.Wrapf(err, "json parse string: %s", blobStr)
	}
	blob := new(identityapi.CredentialContainerImageBlob)
	if err := obj.Unmarshal(blob); err != nil {
		return nil, errors.Wrap(err, "unmarshal blob")
	}
	out := &apis.ContainerPullImageAuthConfig{
		Username:      blob.Username,
		Password:      blob.Password,
		Auth:          blob.Auth,
		ServerAddress: blob.ServerAddress,
		IdentityToken: blob.IdentityToken,
		RegistryToken: blob.RegistryToken,
	}
	return out, nil
}

func (c *SContainer) GetImageCredential(ctx context.Context, userCred mcclient.TokenCredential) (*apis.ContainerPullImageAuthConfig, error) {
	if c.Spec.ImageCredentialId == "" {
		return nil, errors.Wrap(errors.ErrEmpty, "image_credential_id is empty")
	}
	return GetContainerManager().GetImageCredential(ctx, userCred, c.Spec.ImageCredentialId)
}

func (c *SContainer) GetHostPullImageInput(ctx context.Context, userCred mcclient.TokenCredential) (*hostapi.ContainerPullImageInput, error) {
	input := &hostapi.ContainerPullImageInput{
		Image:      c.Spec.Image,
		PullPolicy: c.Spec.ImagePullPolicy,
	}
	if c.Spec.ImageCredentialId != "" {
		cred, err := c.GetImageCredential(ctx, userCred)
		if err != nil {
			return nil, errors.Wrapf(err, "GetImageCredential %s", c.Spec.ImageCredentialId)
		}
		input.Auth = cred
	}
	return input, nil
}

func (c *SContainer) StartPullImageTask(ctx context.Context, userCred mcclient.TokenCredential, input *hostapi.ContainerPullImageInput, parentTaskId string) error {
	c.SetStatus(ctx, userCred, api.CONTAINER_STATUS_PULLING_IMAGE, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ContainerPullImageTask", c, userCred, jsonutils.Marshal(input).(*jsonutils.JSONDict), parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (c *SContainer) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return c.SVirtualResourceBase.Delete(ctx, userCred)
}

func (m *SContainerManager) ConvertVolumeMountRelationToSpec(ctx context.Context, userCred mcclient.TokenCredential, relation []*ContainerVolumeMountRelation) ([]*hostapi.ContainerVolumeMount, error) {
	mounts := make([]*hostapi.ContainerVolumeMount, 0)
	for _, r := range relation {
		mount, err := r.ToHostMount(ctx, userCred)
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
	vmRelation, err := c.GetVolumeMountRelations()
	if err != nil {
		return nil, errors.Wrap(err, "GetVolumeMountRelations")
	}
	mounts, err := GetContainerManager().ConvertVolumeMountRelationToSpec(ctx, userCred, vmRelation)
	if err != nil {
		return nil, errors.Wrap(err, "ConvertVolumeRelationToSpec")
	}
	ctrDevs := make([]*hostapi.ContainerDevice, 0)
	for _, dev := range c.Spec.Devices {
		ctrDev, err := GetContainerDeviceDriver(dev.Type).ToHostDevice(dev)
		if err != nil {
			return nil, errors.Wrapf(err, "ToHostDevice %s", jsonutils.Marshal(dev))
		}
		ctrDevs = append(ctrDevs, ctrDev)
	}

	spec := c.Spec.ContainerSpec
	hSpec := &hostapi.ContainerSpec{
		ContainerSpec: spec,
		VolumeMounts:  mounts,
		Devices:       ctrDevs,
	}
	pullInput, err := c.GetHostPullImageInput(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "GetHostPullImageInput")
	}
	hSpec.ImageCredentialToken = base64.StdEncoding.EncodeToString([]byte(jsonutils.Marshal(pullInput.Auth).String()))
	return hSpec, nil
}

func (c *SContainer) GetJsonDescAtHost(ctx context.Context, userCred mcclient.TokenCredential) (*hostapi.ContainerDesc, error) {
	spec, err := c.ToHostContainerSpec(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "ToHostContainerSpec")
	}
	return &hostapi.ContainerDesc{
		Id:             c.GetId(),
		Name:           c.GetName(),
		Spec:           spec,
		StartedAt:      c.StartedAt,
		LastFinishedAt: c.LastFinishedAt,
		RestartCount:   c.RestartCount,
	}, nil
}

func (c *SContainer) PrepareSaveImage(ctx context.Context, userCred mcclient.TokenCredential, input *api.ContainerSaveVolumeMountToImageInput) (string, error) {
	imageInput := &CreateGlanceImageInput{
		Name:         input.Name,
		GenerateName: input.GenerateName,
		DiskFormat:   imageapi.IMAGE_DISK_FORMAT_TGZ,
		Properties: map[string]string{
			"notes": input.Notes,
		},
		// inherit the ownership of disk
		ProjectId: c.ProjectId,
	}
	if len(input.Dirs) > 0 {
		dirMap := make(map[string]string, 0)
		for _, dir := range input.Dirs {
			dirMap[dir] = dir
		}
		imageInput.Properties[imageapi.IMAGE_INTERNAL_PATH_MAP] = jsonutils.Marshal(dirMap).String()
	}
	if input.UsedByPostOverlay {
		imageInput.Properties[imageapi.IMAGE_USED_BY_POST_OVERLAY] = jsonutils.Marshal(input.UsedByPostOverlay).String()
	}
	// check class metadata
	cm, err := c.GetAllClassMetadata()
	if err != nil {
		return "", errors.Wrap(err, "unable to GetAllClassMetadata")
	}
	imageInput.ClassMetadata = cm
	return DiskManager.CreateGlanceImage(ctx, userCred, imageInput)
}

func (c *SContainer) PerformSaveVolumeMountImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.ContainerSaveVolumeMountToImageInput) (*hostapi.ContainerSaveVolumeMountToImageInput, error) {
	if c.GetStatus() != api.CONTAINER_STATUS_EXITED {
		return nil, httperrors.NewInvalidStatusError("Can't save volume disk of container in status %s", c.Status)
	}
	if c.GetPod().GetStatus() != api.VM_READY {
		return nil, httperrors.NewInvalidStatusError("Can't save volume disk of pod in status %s", c.GetPod().GetStatus())
	}
	vols := c.GetVolumeMounts()
	if input.Index < 0 || input.Index >= len(vols) {
		return nil, httperrors.NewInputParameterError("Only %d volume_mounts", len(vols))
	}

	imageId, err := c.PrepareSaveImage(ctx, userCred, input)
	if err != nil {
		return nil, errors.Wrap(err, "prepare to save image")
	}
	vrs, err := c.GetVolumeMountRelations()
	if err != nil {
		return nil, errors.Wrap(err, "GetVolumeMountRelations")
	}
	hvm, err := vrs[input.Index].ToHostMount(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "ToHostMount")
	}
	hostInput := &hostapi.ContainerSaveVolumeMountToImageInput{
		ImageId:          imageId,
		VolumeMountIndex: input.Index,
		VolumeMount:      hvm,
		VolumeMountDirs:  input.Dirs,
	}

	return hostInput, c.StartSaveVolumeMountImage(ctx, userCred, hostInput, "")
}

func (c *SContainer) StartSaveVolumeMountImage(ctx context.Context, userCred mcclient.TokenCredential, input *hostapi.ContainerSaveVolumeMountToImageInput, parentTaskId string) error {
	c.SetStatus(ctx, userCred, api.CONTAINER_STATUS_SAVING_IMAGE, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ContainerSaveVolumeMountImageTask", c, userCred, jsonutils.Marshal(input).(*jsonutils.JSONDict), parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (c *SContainer) GetPodDriver() IPodDriver {
	driver, err := c.GetPod().GetDriver()
	if err != nil {
		return nil
	}
	return driver.(IPodDriver)
}

func (c *SContainer) GetDetailsExecInfo(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*api.ContainerExecInfoOutput, error) {
	gst := c.GetPod()
	host, err := gst.GetHost()
	if err != nil {
		return nil, errors.Wrap(err, "GetHost")
	}
	out := &api.ContainerExecInfoOutput{
		HostUri:     host.ManagerUri,
		PodId:       c.GuestId,
		ContainerId: c.Id,
	}
	return out, nil
}

func (c *SContainer) PerformExecSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.ContainerExecSyncInput) (jsonutils.JSONObject, error) {
	if !api.ContainerRunningStatus.Has(c.GetStatus()) {
		return nil, httperrors.NewInvalidStatusError("Can't exec container in status %s", c.Status)
	}
	return c.GetPodDriver().RequestExecSyncContainer(ctx, userCred, c, input)
}

func (c *SContainer) PerformSetResourcesLimit(ctx context.Context, userCred mcclient.TokenCredential, _ jsonutils.JSONObject, input *api.ContainerResourcesSetInput) (jsonutils.JSONObject, error) {
	limit := &input.ContainerResources
	if err := c.ValidateResourcesLimit(limit, input.DisableLimitCheck); err != nil {
		return nil, errors.Wrap(err, "ValidateResourcesLimit")
	}
	if _, err := db.Update(c, func() error {
		c.Spec.ResourcesLimit = limit
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "Update spec.resources_limit")
	}
	if !api.ContainerRunningStatus.Has(c.GetStatus()) {
		return nil, nil
	}
	return c.GetPodDriver().RequestSetContainerResourcesLimit(ctx, userCred, c, limit)
}

func (c *SContainer) ValidateResourcesLimit(limit *apis.ContainerResources, disableLimitCheck bool) error {
	if limit == nil {
		return httperrors.NewInputParameterError("limit cannot be nil")
	}
	pod := c.GetPod()
	if limit.CpuCfsQuota != nil {
		if *limit.CpuCfsQuota <= 0 {
			return httperrors.NewInputParameterError("invalid cpu_cfs_quota %f", *limit.CpuCfsQuota)
		}
		if !disableLimitCheck {
			if *limit.CpuCfsQuota > float64(pod.VcpuCount) {
				return httperrors.NewInputParameterError("cpu_cfs_quota %f can't large than %d", *limit.CpuCfsQuota, pod.VcpuCount)
			}
		}
	}
	return nil
}

type ContainerReleasedDevice struct {
	*api.ContainerDevice
	DeviceType  string
	DeviceModel string
}

func NewContainerReleasedDevice(device *api.ContainerDevice, devType, devModel string) *ContainerReleasedDevice {
	return &ContainerReleasedDevice{
		ContainerDevice: device,
		DeviceType:      devType,
		DeviceModel:     devModel,
	}
}

func (c *SContainer) SaveReleasedDevices(ctx context.Context, userCred mcclient.TokenCredential, devs map[string]ContainerReleasedDevice) error {
	return c.SetMetadata(ctx, api.CONTAINER_METADATA_RELEASED_DEVICES, devs, userCred)
}

func (c *SContainer) GetReleasedDevices(ctx context.Context, userCred mcclient.TokenCredential) (map[string]ContainerReleasedDevice, error) {
	out := make(map[string]ContainerReleasedDevice, 0)
	if ret := c.GetMetadata(ctx, api.CONTAINER_METADATA_RELEASED_DEVICES, userCred); ret == "" {
		return out, nil
	}
	obj := c.GetMetadataJson(ctx, api.CONTAINER_METADATA_RELEASED_DEVICES, userCred)
	if obj == nil {
		return nil, errors.Error("get metadata released devices")
	}
	if err := obj.Unmarshal(&out); err != nil {
		return nil, errors.Wrap(err, "Unmarshal metadata released devices")
	}
	return out, nil
}

func (c *SContainer) PerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ContainerPerformStatusInput) (jsonutils.JSONObject, error) {
	if api.ContainerExitedStatus.Has(c.GetStatus()) {
		if sets.NewString(api.CONTAINER_STATUS_PROBE_FAILED, api.CONTAINER_STATUS_NET_FAILED).Has(input.Status) {
			return nil, httperrors.NewInputParameterError("can't set container status to %s when %s", input.Status, c.Status)
		}
	}
	if _, err := db.Update(c, func() error {
		if input.RestartCount > 0 {
			c.RestartCount = input.RestartCount
		}
		if api.ContainerRunningStatus.Has(input.Status) {
			// 当容器状态是运行时 restart_count 重新计数
			c.RestartCount = 0
		}
		if input.StartedAt != nil {
			c.StartedAt = *input.StartedAt
		}
		if input.LastFinishedAt != nil {
			c.LastFinishedAt = *input.LastFinishedAt
		}
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "Update container status")
	}
	return c.SVirtualResourceBase.PerformStatus(ctx, userCred, query, input.PerformStatusInput)
}

func (c *SContainer) getContainerHostCommitInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.ContainerCommitInput) (*hostapi.ContainerCommitInput, error) {
	var hostInput = &hostapi.ContainerCommitInput{
		Auth: new(apis.ContainerPullImageAuthConfig),
	}
	var repoUrl string
	imageName := input.ImageName
	tag := input.Tag
	if imageName == "" {
		imageName = c.GetName()
	}
	if tag == "" {
		tag = time.Now().Format("20060102150405")
	}
	if input.RegistryId != "" {
		s := auth.GetSession(ctx, userCred, consts.GetRegion())
		obj, err := kubemod.ContainerRegistries.Get(s, input.RegistryId, nil)
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
		reg := new(api.KubeServerContainerRegistryDetails)
		if err := obj.Unmarshal(reg); err != nil {
			return nil, errors.Wrap(err, "Unmarshal kube server registry details")
		}
		if reg.Config == nil {
			confObj, err := kubemod.ContainerRegistries.GetSpecific(s, input.RegistryId, "config", nil)
			if err != nil {
				return nil, errors.Wrap(err, "Get kube server registry config")
			}
			reg.Config = new(api.KubeServerContainerRegistryConfig)
			if err := confObj.Unmarshal(reg.Config); err != nil {
				return nil, errors.Wrap(err, "Unmarshal kube server registry config")
			}
		}
		repoUrl = reg.Url
		if reg.Config != nil {
			switch reg.Type {
			case "common":
				cfg := reg.Config.Common
				hostInput.Auth.Username = cfg.Username
				hostInput.Auth.Password = cfg.Password
			case "harbor":
				cfg := reg.Config.Harbor
				hostInput.Auth.Username = cfg.Username
				hostInput.Auth.Password = cfg.Password
			default:
				return nil, httperrors.NewInputParameterError("invalid registry type %s", reg.Type)
			}
		}
	} else if input.ExternalRegistry != nil {
		repoUrl = input.ExternalRegistry.Url
		if repoUrl == "" {
			return nil, httperrors.NewNotEmptyError("empty external registry url")
		}
		hostInput.Auth = input.ExternalRegistry.Auth
	} else {
		return nil, httperrors.NewInputParameterError("one of registry_id or external_registry must provided")
	}
	repoPrefix := strings.TrimPrefix(strings.TrimPrefix(repoUrl, "http://"), "https://")
	hostInput.Repository = fmt.Sprintf("%s:%s", filepath.Join(repoPrefix, imageName), tag)
	return hostInput, nil
}

func (c *SContainer) PerformCommit(ctx context.Context, userCred mcclient.TokenCredential, _ jsonutils.JSONObject, input *api.ContainerCommitInput) (*api.ContainerCommitOutput, error) {
	hostInput, err := c.getContainerHostCommitInput(ctx, userCred, input)
	if err != nil {
		return nil, err
	}
	out := &api.ContainerCommitOutput{
		Repository: hostInput.Repository,
	}
	return out, c.StartCommit(ctx, userCred, hostInput, "")
}

func (c *SContainer) StartCommit(ctx context.Context, userCred mcclient.TokenCredential, input *hostapi.ContainerCommitInput, parentTaskId string) error {
	c.SetStatus(ctx, userCred, api.CONTAINER_STATUS_COMMITTING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ContainerCommitTask", c, userCred, jsonutils.Marshal(input).(*jsonutils.JSONDict), parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (c *SContainer) isPostOverlayExist(vm *apis.ContainerVolumeMount, ov *apis.ContainerVolumeMountDiskPostOverlay) (*apis.ContainerVolumeMountDiskPostOverlay, bool) {
	for i := range vm.Disk.PostOverlay {
		cov := vm.Disk.PostOverlay[i]
		if cov.IsEqual(*ov) {
			return cov, true
		}
	}
	return nil, false
}

func (c *SContainer) validateVolumeMountPostOverlayAction(action string, index int) (*apis.ContainerVolumeMount, error) {
	if !api.ContainerExitedStatus.Has(c.Status) && !api.ContainerRunningStatus.Has(c.Status) {
		return nil, httperrors.NewInvalidStatusError("can't %s post overlay on status %s", action, c.Status)
	}
	if index >= len(c.Spec.VolumeMounts) {
		return nil, httperrors.NewInputParameterError("index %d out of volume_mount size %d", index, len(c.Spec.VolumeMounts))
	}
	vm := new(apis.ContainerVolumeMount)
	curVm := c.Spec.VolumeMounts[index]
	if err := jsonutils.Marshal(curVm).Unmarshal(vm); err != nil {
		return nil, errors.Wrap(err, "use json unmarshal to new volume mount")
	}
	if vm.Type != apis.CONTAINER_VOLUME_MOUNT_TYPE_DISK {
		return nil, httperrors.NewInputParameterError("invalid volume mount type %s", vm.Type)
	}
	return vm, nil
}

func (c *SContainer) GetVolumeMountCopy(index int) (*apis.ContainerVolumeMount, error) {
	if index >= len(c.Spec.VolumeMounts) {
		return nil, httperrors.NewInputParameterError("index %d out of volume_mount size %d", index, len(c.Spec.VolumeMounts))
	}
	vm := new(apis.ContainerVolumeMount)
	curVm := c.Spec.VolumeMounts[index]
	if err := jsonutils.Marshal(curVm).Unmarshal(vm); err != nil {
		return nil, errors.Wrap(err, "use json unmarshal to new volume mount")
	}
	return vm, nil
}

func (c *SContainer) getPostOverlayVolumeMount(
	index int,
	updateF func(mount *apis.ContainerVolumeMount) (*apis.ContainerVolumeMount, error),
) (*apis.ContainerVolumeMount, error) {
	vm, err := c.GetVolumeMountCopy(index)
	if err != nil {
		return nil, err
	}
	return updateF(vm)
}

func (c *SContainer) GetAddPostOverlayVolumeMount(index int, ovs []*apis.ContainerVolumeMountDiskPostOverlay) (*apis.ContainerVolumeMount, error) {
	return c.getPostOverlayVolumeMount(index, func(vm *apis.ContainerVolumeMount) (*apis.ContainerVolumeMount, error) {
		if vm.Disk.PostOverlay == nil {
			vm.Disk.PostOverlay = []*apis.ContainerVolumeMountDiskPostOverlay{}
		}
		vm.Disk.PostOverlay = append(vm.Disk.PostOverlay, ovs...)
		return vm, nil
	})
}

func (c *SContainer) GetRemovePostOverlayVolumeMount(index int, ovs []*apis.ContainerVolumeMountDiskPostOverlay) (*apis.ContainerVolumeMount, error) {
	return c.getPostOverlayVolumeMount(index, func(vm *apis.ContainerVolumeMount) (*apis.ContainerVolumeMount, error) {
		// remove post overlay
		for _, ov := range ovs {
			vm.Disk = c.removePostOverlay(vm.Disk, ov)
		}
		return vm, nil
	})
}

func (c *SContainer) PerformAddVolumeMountPostOverlay(ctx context.Context, userCred mcclient.TokenCredential, _ jsonutils.JSONObject, input *api.ContainerVolumeMountAddPostOverlayInput) (jsonutils.JSONObject, error) {
	vm, err := c.validateVolumeMountPostOverlayAction("add", input.Index)
	if err != nil {
		return nil, err
	}
	totalOvs := []*apis.ContainerVolumeMountDiskPostOverlay{}
	totalOvs = append(totalOvs, vm.Disk.PostOverlay...)
	drv := GetContainerVolumeMountDriver(vm.Type)
	dDrv := drv.(IContainerVolumeMountDiskDriver)
	for i := range input.PostOverlay {
		ov := input.PostOverlay[i]
		cov, isExist := c.isPostOverlayExist(vm, ov)
		if isExist {
			return nil, httperrors.NewInputParameterError("post overlay already exists: %s", jsonutils.Marshal(cov))
		}
		if err := dDrv.ValidatePostSingleOverlay(ctx, userCred, ov); err != nil {
			return nil, errors.Wrapf(err, "validate post overlay %s", jsonutils.Marshal(ov))
		}
		totalOvs = append(totalOvs, ov)
	}
	if err := dDrv.ValidatePostOverlayTargetDirs(totalOvs); err != nil {
		return nil, errors.Wrapf(err, "validate container target dirs")
	}
	return nil, c.StartAddVolumeMountPostOverlayTask(ctx, userCred, input, "")
}

func (c *SContainer) StartCacheImagesTask(ctx context.Context, userCred mcclient.TokenCredential, input *api.ContainerCacheImagesInput, parentTaskId string) error {
	c.SetStatus(ctx, userCred, api.CONTAINER_STATUS_CACHE_IMAGE, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ContainerCacheImagesTask", c, userCred, jsonutils.Marshal(input).(*jsonutils.JSONDict), parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "New ContainerCacheImagesTask")
	}
	return task.ScheduleRun(nil)
}

func (c *SContainer) StartAddVolumeMountPostOverlayTask(ctx context.Context, userCred mcclient.TokenCredential, input *api.ContainerVolumeMountAddPostOverlayInput, parentTaskId string) error {
	c.SetStatus(ctx, userCred, api.CONTAINER_STATUS_ADD_POST_OVERLY, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ContainerAddVolumeMountPostOverlayTask", c, userCred, jsonutils.Marshal(input).(*jsonutils.JSONDict), parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "New ContainerAddVolumeMountPostOverlayTask")
	}
	return task.ScheduleRun(nil)
}

func (c *SContainer) StartRemoveVolumeMountPostOverlayTask(ctx context.Context, userCred mcclient.TokenCredential, input *api.ContainerVolumeMountRemovePostOverlayInput, parentTaskId string) error {
	c.SetStatus(ctx, userCred, api.CONTAINER_STATUS_REMOVE_POST_OVERLY, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ContainerRemoveVolumeMountPostOverlayTask", c, userCred, jsonutils.Marshal(input).(*jsonutils.JSONDict), parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "New ContainerRemoveVolumeMountPostOverlayTask")
	}
	return task.ScheduleRun(nil)
}

func (c *SContainer) PerformRemoveVolumeMountPostOverlay(ctx context.Context, userCred mcclient.TokenCredential, _ jsonutils.JSONObject, input *api.ContainerVolumeMountRemovePostOverlayInput) (jsonutils.JSONObject, error) {
	vm, err := c.validateVolumeMountPostOverlayAction("remove", input.Index)
	if err != nil {
		return nil, err
	}
	if len(vm.Disk.PostOverlay) == 0 {
		return nil, httperrors.NewInputParameterError("no post overlay")
	}
	for i, ov := range input.PostOverlay {
		cov, isExist := c.isPostOverlayExist(vm, ov)
		if !isExist {
			return nil, httperrors.NewInputParameterError("post overlay not exists: %s", jsonutils.Marshal(ov))
		}
		input.PostOverlay[i] = cov
	}
	return nil, c.StartRemoveVolumeMountPostOverlayTask(ctx, userCred, input, "")
}

func (c *SContainer) removePostOverlay(vmd *apis.ContainerVolumeMountDisk, ov *apis.ContainerVolumeMountDiskPostOverlay) *apis.ContainerVolumeMountDisk {
	curOvs := vmd.PostOverlay
	for i, cov := range curOvs {
		if cov.IsEqual(*ov) {
			curOvs = append(curOvs[:i], curOvs[i+1:]...)
		}
	}
	vmd.PostOverlay = curOvs
	return vmd
}
