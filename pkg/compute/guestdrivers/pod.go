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

package guestdrivers

import (
	"context"
	"fmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var _ models.IPodDriver = new(SPodDriver)

type SPodDriver struct {
	SKVMGuestDriver
}

func init() {
	driver := SPodDriver{}
	models.RegisterGuestDriver(&driver)
}

func (p *SPodDriver) newUnsupportOperationError(option string) error {
	return httperrors.NewUnsupportOperationError("Container not support %s", option)
}

func (p *SPodDriver) GetHypervisor() string {
	return api.HYPERVISOR_POD
}

func (p *SPodDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_ONECLOUD
}

func (p *SPodDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	if input.Pod == nil {
		return nil, httperrors.NewNotEmptyError("pod data is empty")
	}
	if len(input.Pod.Containers) == 0 {
		return nil, httperrors.NewNotEmptyError("containers data is empty")
	}
	// validate port mappings
	if err := p.validatePortMappings(input.Pod); err != nil {
		return nil, errors.Wrap(err, "validate port mappings")
	}

	sameName := ""
	for idx, ctr := range input.Pod.Containers {
		if err := p.validateContainerData(ctx, userCred, idx, input.Name, ctr, input); err != nil {
			return nil, errors.Wrapf(err, "data of %d container", idx)
		}
		if ctr.Name == sameName {
			return nil, httperrors.NewDuplicateNameError("same name %s of containers", ctr.Name)
		}
		sameName = ctr.Name
	}

	// always set auto_start to true
	input.AutoStart = true
	return input, nil
}

func (p *SPodDriver) validatePortMappings(input *api.PodCreateInput) error {
	for idx, pm := range input.PortMappings {
		// TODO: 判断 host port 是否重复
		if err := p.validatePortMapping(pm); err != nil {
			return errors.Wrapf(err, "validate portmapping %d", idx)
		}
	}
	return nil
}

func (p *SPodDriver) validateContainerData(ctx context.Context, userCred mcclient.TokenCredential, idx int, defaultNamePrefix string, ctr *api.PodContainerCreateInput, input *api.ServerCreateInput) error {
	if ctr.Name == "" {
		ctr.Name = fmt.Sprintf("%s-%d", defaultNamePrefix, idx)
	}
	if err := models.GetContainerManager().ValidateSpec(ctx, userCred, &ctr.ContainerSpec, nil); err != nil {
		return errors.Wrap(err, "validate container spec")
	}
	if err := p.validateContainerVolumeMounts(ctx, userCred, ctr, input); err != nil {
		return errors.Wrap(err, "validate container volumes")
	}
	return nil
}

func (p *SPodDriver) validateContainerVolumeMounts(ctx context.Context, userCred mcclient.TokenCredential, ctr *api.PodContainerCreateInput, input *api.ServerCreateInput) error {
	for idx, vm := range ctr.VolumeMounts {
		if err := p.validateContainerVolumeMount(ctx, userCred, vm, input); err != nil {
			return errors.Wrapf(err, "validate volume mount %d", idx)
		}
	}
	return nil
}

func (p *SPodDriver) validateContainerVolumeMount(ctx context.Context, userCred mcclient.TokenCredential, vm *apis.ContainerVolumeMount, input *api.ServerCreateInput) error {
	if vm.Type == "" {
		return httperrors.NewNotEmptyError("type is required")
	}
	if vm.MountPath == "" {
		return httperrors.NewNotEmptyError("mount_path is required")
	}
	drv, err := models.GetContainerVolumeMountDriverWithError(vm.Type)
	if err != nil {
		return errors.Wrapf(err, "get container volume mount driver %s", vm.Type)
	}
	if err := drv.ValidatePodCreateData(ctx, userCred, vm, input); err != nil {
		return errors.Wrapf(err, "validate %s create data", vm.Type)
	}
	return nil
}

func (p *SPodDriver) validatePortRange(port int32) error {
	if port <= 0 || port > 65535 {
		return httperrors.NewInputParameterError("port number %d isn't within 1 to 65535", port)
	}
	return nil
}

func (p *SPodDriver) validatePortMapping(pm *api.PodPortMapping) error {
	if err := p.validatePortRange(pm.HostPort); err != nil {
		return errors.Wrap(err, "validate host_port")
	}
	if err := p.validatePortRange(pm.ContainerPort); err != nil {
		return errors.Wrap(err, "validate container_port")
	}
	if pm.Protocol == "" {
		pm.Protocol = api.PodPortMappingProtocolTCP
	}
	if !sets.NewString(api.PodPortMappingProtocolSCTP, api.PodPortMappingProtocolUDP, api.PodPortMappingProtocolTCP).Has(string(pm.Protocol)) {
		return httperrors.NewInputParameterError("unsupported protocol %s", pm.Protocol)
	}
	return nil
}

func (p *SPodDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
	return cloudprovider.SInstanceCapability{
		Hypervisor: p.GetHypervisor(),
		Provider:   p.GetProvider(),
	}
}

// for backward compatibility, deprecated driver
func (p *SPodDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_ON_PREMISE
	keys.Provider = api.CLOUD_PROVIDER_ONECLOUD
	keys.Brand = api.ONECLOUD_BRAND_ONECLOUD
	keys.Hypervisor = api.HYPERVISOR_POD
	return keys
}

func (p *SPodDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_LOCAL
}

func (p *SPodDriver) GetMinimalSysDiskSizeGb() int {
	return options.Options.DefaultDiskSizeMB / 1024
}

func (p *SPodDriver) StartGuestCreateTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, pendingUsage quotas.IQuota, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "PodCreateTask", guest, userCred, data, parentTaskId, "", pendingUsage)
	if err != nil {
		return errors.Wrap(err, "New PodCreateTask")
	}
	return task.ScheduleRun(nil)
}

func (p *SPodDriver) RequestGuestHotAddIso(ctx context.Context, guest *models.SGuest, path string, boot bool, task taskman.ITask) error {
	// do nothing, call next stage
	return task.ScheduleRun(nil)
}

func (p *SPodDriver) RequestStartOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential, task taskman.ITask) error {
	header := p.getTaskRequestHeader(task)

	config := jsonutils.NewDict()
	desc, err := guest.GetDriver().GetJsonDescAtHost(ctx, userCred, guest, host, nil)
	if err != nil {
		return errors.Wrapf(err, "GetJsonDescAtHost")
	}
	config.Add(desc, "desc")
	params := task.GetParams()
	if params.Length() > 0 {
		config.Add(params, "params")
	}
	url := fmt.Sprintf("%s/servers/%s/start", host.ManagerUri, guest.Id)
	_, body, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, config, false)
	if err != nil {
		return err
	}
	resp := new(api.PodStartResponse)
	body.Unmarshal(resp)
	if resp.IsRunning {
		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			return body, nil
		})
	}
	return nil
}

func (p *SPodDriver) RqeuestSuspendOnHost(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return p.newUnsupportOperationError("suspend")
}

func (p *SPodDriver) RequestSoftReset(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return p.newUnsupportOperationError("soft reset")
}

func (p *SPodDriver) GetGuestVncInfo(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost, input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	return nil, p.newUnsupportOperationError("VNC")
}

func (p *SPodDriver) OnGuestDeployTaskDataReceived(ctx context.Context, guest *models.SGuest, task taskman.ITask, data jsonutils.JSONObject) error {
	//guest.SaveDeployInfo(ctx, task.GetUserCred(), data)
	// do nothing here
	return nil
}

func (p *SPodDriver) RequestUndeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	task, err := taskman.TaskManager.NewTask(ctx, "PodDeleteTask", guest, task.GetUserCred(), nil, task.GetTaskId(), "", nil)
	if err != nil {
		return errors.Wrap(err, "New PodDeleteTask")
	}
	return task.ScheduleRun(nil)
}

func (p *SPodDriver) RequestUndeployPod(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	url := fmt.Sprintf("%s/servers/%s", host.ManagerUri, guest.Id)
	header := p.getTaskRequestHeader(task)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "DELETE", url, header, nil, false)
	return err
}

func (p *SPodDriver) GetJsonDescAtHost(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost, params *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	desc := guest.GetJsonDescAtHypervisor(ctx, host)
	ctrs, err := models.GetContainerManager().GetContainersByPod(guest.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "GetContainersByPod")
	}
	ctrDescs := make([]*api.ContainerDesc, len(ctrs))
	for idx, ctr := range ctrs {
		ctrDescs[idx] = ctr.GetJsonDescAtHost()
	}
	desc.Containers = ctrDescs
	return jsonutils.Marshal(desc), nil
}

func (p *SPodDriver) createContainersOnPod(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest) error {
	input, err := guest.GetCreateParams(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "GetCreateParams")
	}
	ctrs := make([]*models.SContainer, len(input.Pod.Containers))
	for idx, ctr := range input.Pod.Containers {
		if obj, err := models.GetContainerManager().CreateOnPod(ctx, userCred, guest.GetOwnerId(), guest, ctr); err != nil {
			return errors.Wrapf(err, "create container on pod: %s", guest.GetName())
		} else {
			ctrs[idx] = obj
		}
	}
	return nil
}

func (p *SPodDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	deployAction, err := task.GetParams().GetString("deploy_action")
	if err != nil {
		return errors.Wrapf(err, "get deploy_action from task params: %s", task.GetParams())
	}
	if deployAction == "create" {
		if err := p.createContainersOnPod(ctx, task.GetUserCred(), guest); err != nil {
			return errors.Wrap(err, "create containers on pod")
		}
	}
	config, err := guest.GetDeployConfigOnHost(ctx, task.GetUserCred(), host, task.GetParams())
	if err != nil {
		log.Errorf("GetDeployConfigOnHost error: %v", err)
		return err
	}
	action, err := config.GetString("action")
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/servers/%s/%s", host.ManagerUri, guest.Id, action)
	header := p.getTaskRequestHeader(task)
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, config, false)
	return err
}

func (p *SPodDriver) performContainerAction(ctx context.Context, userCred mcclient.TokenCredential, task models.IContainerTask, action string, data jsonutils.JSONObject) error {
	pod := task.GetPod()
	ctr := task.GetContainer()
	host, _ := pod.GetHost()
	url := fmt.Sprintf("%s/pods/%s/containers/%s/%s", host.ManagerUri, pod.GetId(), ctr.GetId(), action)
	header := p.getTaskRequestHeader(task)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, data, false)
	return err
}

func (p *SPodDriver) getContainerCreateInput(ctx context.Context, userCred mcclient.TokenCredential, ctr *models.SContainer) (*hostapi.ContainerCreateInput, error) {
	spec, err := ctr.ToHostContainerSpec(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "ToHostContainerSpec")
	}
	input := &hostapi.ContainerCreateInput{
		Name:    ctr.GetName(),
		GuestId: ctr.GuestId,
		Spec:    spec,
	}
	return input, nil
}

func (p *SPodDriver) RequestCreateContainer(ctx context.Context, userCred mcclient.TokenCredential, task models.IContainerTask) error {
	ctr := task.GetContainer()
	input, err := p.getContainerCreateInput(ctx, userCred, ctr)
	if err != nil {
		return errors.Wrap(err, "getContainerCreateInput")
	}
	return p.performContainerAction(ctx, userCred, task, "create", jsonutils.Marshal(input))
}

func (p *SPodDriver) RequestStartContainer(ctx context.Context, userCred mcclient.TokenCredential, task models.IContainerTask) error {
	ctr := task.GetContainer()
	input, err := p.getContainerCreateInput(ctx, userCred, ctr)
	if err != nil {
		return errors.Wrap(err, "getContainerCreateInput")
	}
	return p.performContainerAction(ctx, userCred, task, "start", jsonutils.Marshal(input))
}

func (p *SPodDriver) RequestStopContainer(ctx context.Context, userCred mcclient.TokenCredential, task models.IContainerTask) error {
	return p.performContainerAction(ctx, userCred, task, "stop", task.GetParams())
}

func (p *SPodDriver) RequestDeleteContainer(ctx context.Context, userCred mcclient.TokenCredential, task models.IContainerTask) error {
	return p.performContainerAction(ctx, userCred, task, "delete", nil)
}

func (p *SPodDriver) RequestSyncContainerStatus(ctx context.Context, userCred mcclient.TokenCredential, task models.IContainerTask) error {
	return p.performContainerAction(ctx, userCred, task, "sync-status", nil)
}

func (p *SPodDriver) RequestPullContainerImage(ctx context.Context, userCred mcclient.TokenCredential, task models.IContainerTask) error {
	return p.performContainerAction(ctx, userCred, task, "pull-image", task.GetParams())
}

func (p *SPodDriver) OnDeleteGuestFinalCleanup(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential) error {
	// clean disk records in DB
	return guest.DeleteAllDisksInDB(ctx, userCred)
}

func (p *SPodDriver) RequestSyncConfigOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	// do nothing, call next stage
	task.ScheduleRun(nil)
	return nil
}

func (p *SPodDriver) RequestChangeVmConfig(ctx context.Context, guest *models.SGuest, task taskman.ITask, instanceType string, vcpuCount, cpuSockets, vmemSize int64) error {
	return p.newUnsupportOperationError("change config")
}

func (p *SPodDriver) RequestRebuildRootDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	// do nothing, call next stage
	return p.newUnsupportOperationError("rebuild root")
}

func (p *SPodDriver) GetRandomNetworkTypes() []string {
	return []string{api.NETWORK_TYPE_CONTAINER, api.NETWORK_TYPE_GUEST}
}

func (p *SPodDriver) StartGuestRestartTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, isForce bool, parentTaskId string) error {
	return fmt.Errorf("Not Implement")
}

func (p *SPodDriver) IsSupportGuestClone() bool {
	return false
}

func (p *SPodDriver) IsSupportCdrom(guest *models.SGuest) (bool, error) {
	return false, nil
}

func (p *SPodDriver) IsSupportFloppy(guest *models.SGuest) (bool, error) {
	return false, nil
}
