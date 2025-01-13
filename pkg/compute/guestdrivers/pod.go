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
	"net/http"
	"net/url"

	"k8s.io/apimachinery/pkg/util/proxy"

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
	"yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/pod/remotecommand/spdy"
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
	for i, d := range input.Disks {
		if d.Format != "" {
			if d.Format != image.IMAGE_DISK_FORMAT_RAW {
				return nil, httperrors.NewInputParameterError("not support format %s for disk %d", d.Format, i)
			}
		}
	}
	if input.Pod == nil {
		return nil, httperrors.NewNotEmptyError("pod data is empty")
	}
	if len(input.Pod.Containers) == 0 {
		return nil, httperrors.NewNotEmptyError("containers data is empty")
	}
	// validate port mappings
	/*if err := p.validatePortMappings(input.Pod); err != nil {
		return nil, errors.Wrap(err, "validate port mappings")
	}*/

	ctrNames := sets.NewString()
	volUniqNames := sets.NewString()
	for idx, ctr := range input.Pod.Containers {
		if err := p.validateContainerData(ctx, userCred, idx, input.Name, ctr, input); err != nil {
			return nil, errors.Wrapf(err, "data of %d container", idx)
		}
		if ctrNames.Has(ctr.Name) {
			return nil, httperrors.NewDuplicateNameError("same name %s of containers", ctr.Name)
		}
		ctrNames.Insert(ctr.Name)
		for volIdx := range ctr.VolumeMounts {
			vol := ctr.VolumeMounts[volIdx]
			if vol.UniqueName != "" {
				if volUniqNames.Has(vol.UniqueName) {
					return nil, httperrors.NewDuplicateNameError("same volume unique name %s", fmt.Sprintf("container %s volume_mount %d %s", ctr.Name, volIdx, vol.UniqueName))
				} else {
					volUniqNames.Insert(vol.UniqueName)
				}
			}
		}
	}

	return input, nil
}

/*func (p *SPodDriver) validatePortMappings(input *api.PodCreateInput) error {
	usedPorts := make(map[api.PodPortMappingProtocol]sets.Int)
	for idx, pm := range input.PortMappings {
		ports, ok := usedPorts[pm.Protocol]
		if !ok {
			ports = sets.NewInt()
		}
		if pm.HostPort != nil {
			if ports.Has(*pm.HostPort) {
				return httperrors.NewInputParameterError("%s host_port %d is already specified", pm.Protocol, *pm.HostPort)
			}
			ports.Insert(*pm.HostPort)
		}
		usedPorts[pm.Protocol] = ports
		if err := p.validatePortMapping(pm); err != nil {
			return errors.Wrapf(err, "validate portmapping %d", idx)
		}
	}
	return nil
}*/

func (p *SPodDriver) validateHostPortMapping(hostId string, pm *api.PodPortMapping) error {
	// TODO:
	return nil
}

func (p *SPodDriver) validateContainerData(ctx context.Context, userCred mcclient.TokenCredential, idx int, defaultNamePrefix string, ctr *api.PodContainerCreateInput, input *api.ServerCreateInput) error {
	if ctr.Name == "" {
		ctr.Name = fmt.Sprintf("%s-%d", defaultNamePrefix, idx)
	}
	if err := models.GetContainerManager().ValidateSpec(ctx, userCred, &ctr.ContainerSpec, nil, nil); err != nil {
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

func (p *SPodDriver) validatePortRange(portRange *api.PodPortMappingPortRange) error {
	if portRange != nil {
		if portRange.Start > portRange.End {
			return httperrors.NewInputParameterError("port range start %d is large than %d", portRange.Start, portRange.End)
		}
		if portRange.Start <= api.POD_PORT_MAPPING_RANGE_START {
			return httperrors.NewInputParameterError("port range start %d <= %d", api.POD_PORT_MAPPING_RANGE_START, portRange.Start)
		}
		if portRange.End > api.POD_PORT_MAPPING_RANGE_END {
			return httperrors.NewInputParameterError("port range end %d > %d", api.POD_PORT_MAPPING_RANGE_END, portRange.End)
		}
	}
	return nil
}

func (p *SPodDriver) validatePort(port int, start int, end int) error {
	if port < start || port > end {
		return httperrors.NewInputParameterError("port number %d isn't within %d to %d", port, start, end)
	}
	return nil
}

func (p *SPodDriver) validatePortMapping(pm *api.PodPortMapping) error {
	if err := p.validatePortRange(pm.HostPortRange); err != nil {
		return err
	}
	if pm.HostPort != nil {
		if err := p.validatePort(*pm.HostPort, api.POD_PORT_MAPPING_RANGE_START, api.POD_PORT_MAPPING_RANGE_END); err != nil {
			return errors.Wrap(err, "validate host_port")
		}
	}
	if err := p.validatePort(pm.ContainerPort, 1, 65535); err != nil {
		return errors.Wrap(err, "validate container_port")
	}
	if pm.Protocol == "" {
		pm.Protocol = api.PodPortMappingProtocolTCP
	}
	if !sets.NewString(api.PodPortMappingProtocolUDP, api.PodPortMappingProtocolTCP).Has(string(pm.Protocol)) {
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

func (p *SPodDriver) PerformStart(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, data *jsonutils.JSONDict, parentTaskId string) error {
	guest.SetStatus(ctx, userCred, api.VM_START_START, "")
	task, err := taskman.TaskManager.NewTask(ctx, "PodStartTask", guest, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "New PodStartTask")
	}
	return task.ScheduleRun(nil)
}

func (p *SPodDriver) RequestStartOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential, task taskman.ITask) error {
	header := p.getTaskRequestHeader(task)

	config := jsonutils.NewDict()
	drv, err := guest.GetDriver()
	if err != nil {
		return err
	}
	desc, err := drv.GetJsonDescAtHost(ctx, task.GetUserCred(), guest, host, nil)
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

func (p *SPodDriver) StartGuestStopTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "PodStopTask", guest, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "New PodStopTask")
	}
	return task.ScheduleRun(nil)
}

func (p *SPodDriver) StartGuestRestartTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, isForce bool, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Set("is_force", jsonutils.NewBool(isForce))
	if err := guest.SetStatus(ctx, userCred, api.VM_STOPPING, ""); err != nil {
		return err
	}
	task, err := taskman.TaskManager.NewTask(ctx, "PodRestartTask", guest, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (p *SPodDriver) StartDeleteGuestTask(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "PodDeleteTask", guest, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "New PodDeleteTask")
	}
	return task.ScheduleRun(nil)
}

func (p *SPodDriver) StartGuestSyncstatusTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "PodSyncstatusTask", guest, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "New PodSyncstatusTask")
	}
	return task.ScheduleRun(nil)
}

func (p *SPodDriver) RequestUndeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
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
	ctrDescs := make([]*hostapi.ContainerDesc, len(ctrs))
	for idx, ctr := range ctrs {
		desc, err := ctr.GetJsonDescAtHost(ctx, userCred)
		if err != nil {
			return nil, errors.Wrapf(err, "GetJsonDescAtHost of container %s", ctr.GetId())
		}
		ctrDescs[idx] = desc
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
		Name:         ctr.GetName(),
		GuestId:      ctr.GuestId,
		Spec:         spec,
		RestartCount: ctr.RestartCount,
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
	ctr := task.GetContainer()
	params := task.GetParams()
	params.Add(jsonutils.NewString(ctr.GetName()), "container_name")
	params.Add(jsonutils.NewInt(int64(ctr.Spec.ShmSizeMB)), "shm_size_mb")
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

func (p *SPodDriver) RequestAddVolumeMountPostOverlay(ctx context.Context, userCred mcclient.TokenCredential, task models.IContainerTask) error {
	return p.performContainerAction(ctx, userCred, task, "add-volume-mount-post-overlay", task.GetParams())
}

func (p *SPodDriver) RequestRemoveVolumeMountPostOverlay(ctx context.Context, userCred mcclient.TokenCredential, task models.IContainerTask) error {
	return p.performContainerAction(ctx, userCred, task, "remove-volume-mount-post-overlay", task.GetParams())
}

type responder struct {
	errorMessage string
}

func (r *responder) Error(w http.ResponseWriter, req *http.Request, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func (p *SPodDriver) RequestExecContainer(ctx context.Context, userCred mcclient.TokenCredential, ctr *models.SContainer, input *api.ContainerExecInput) error {
	pod := ctr.GetPod()
	host, _ := pod.GetHost()
	urlPath := fmt.Sprintf("%s/pods/%s/containers/%s/%s?%s", host.ManagerUri, pod.GetId(), ctr.GetId(), "exec", jsonutils.Marshal(input).QueryString())
	loc, _ := url.Parse(urlPath)
	tokenHeader := mcclient.GetTokenHeaders(userCred)
	trans, _, _ := spdy.RoundTripperFor()
	handler := proxy.NewUpgradeAwareHandler(loc, trans, false, true, new(responder))
	appParams := appsrv.AppContextGetParams(ctx)
	newHeader := appParams.Request.Header
	for key, vals := range tokenHeader {
		for _, val := range vals {
			newHeader.Add(key, val)
		}
	}
	appParams.Request.Header = newHeader
	appParams.Request.Method = "POST"
	handler.ServeHTTP(appParams.Response, appParams.Request)
	return nil
}

func (p *SPodDriver) requestContainerSyncAction(ctx context.Context, userCred mcclient.TokenCredential, container *models.SContainer, action string, input jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	pod := container.GetPod()
	host, _ := pod.GetHost()
	url := fmt.Sprintf("%s/pods/%s/containers/%s/%s", host.ManagerUri, pod.GetId(), container.GetId(), action)
	header := mcclient.GetTokenHeaders(userCred)
	_, ret, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, input, false)
	return ret, err
}

func (p *SPodDriver) RequestExecSyncContainer(ctx context.Context, userCred mcclient.TokenCredential, container *models.SContainer, input *api.ContainerExecSyncInput) (jsonutils.JSONObject, error) {
	return p.requestContainerSyncAction(ctx, userCred, container, "exec-sync", jsonutils.Marshal(input))
}

func (p *SPodDriver) RequestSetContainerResourcesLimit(ctx context.Context, userCred mcclient.TokenCredential, container *models.SContainer, limit *apis.ContainerResources) (jsonutils.JSONObject, error) {
	return p.requestContainerSyncAction(ctx, userCred, container, "set-resources-limit", jsonutils.Marshal(limit))
}

func (p *SPodDriver) OnDeleteGuestFinalCleanup(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential) error {
	// clean disk records in DB
	return guest.DeleteAllDisksInDB(ctx, userCred)
}

func (p *SPodDriver) RequestRebuildRootDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	// do nothing, call next stage
	return p.newUnsupportOperationError("rebuild root")
}

func (p *SPodDriver) GetRandomNetworkTypes() []api.TNetworkType {
	return []api.TNetworkType{api.NETWORK_TYPE_CONTAINER, api.NETWORK_TYPE_GUEST, api.NETWORK_TYPE_HOSTLOCAL}
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

func (p *SPodDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (p *SPodDriver) RequestSaveVolumeMountImage(ctx context.Context, userCred mcclient.TokenCredential, task models.IContainerTask) error {
	return p.performContainerAction(ctx, userCred, task, "save-volume-mount-to-image", task.GetParams())
}

func (p *SPodDriver) RequestCommitContainer(ctx context.Context, userCred mcclient.TokenCredential, task models.IContainerTask) error {
	return p.performContainerAction(ctx, userCred, task, "commit", task.GetParams())
}

func (p *SPodDriver) RequestDiskSnapshot(ctx context.Context, guest *models.SGuest, task taskman.ITask, snapshotId, diskId string) error {
	/*if guest.GetStatus() != api.VM_READY {
		return httperrors.NewNotAcceptableError("pod status %s is not ready", guest.GetStatus())
	}*/
	return p.SKVMGuestDriver.RequestDiskSnapshot(ctx, guest, task, snapshotId, diskId)
}

func (p *SPodDriver) RequestDeleteSnapshot(ctx context.Context, guest *models.SGuest, task taskman.ITask, params *jsonutils.JSONDict) error {
	/*if guest.GetStatus() != api.VM_READY {
		return httperrors.NewNotAcceptableError("pod status %s is not ready", guest.GetStatus())
	}*/
	return p.SKVMGuestDriver.RequestDeleteSnapshot(ctx, guest, task, params)
}

func (p *SPodDriver) BeforeDetachIsolatedDevice(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, dev *models.SIsolatedDevice) error {
	ctrs, err := models.GetContainerManager().GetContainersByPod(guest.GetId())
	if err != nil {
		return errors.Wrapf(err, "get containers by pod %s", guest.GetId())
	}
	for _, ctr := range ctrs {
		ctrPtr := &ctr
		spec := ctrPtr.Spec
		devs := spec.Devices
		newDevs := make([]*api.ContainerDevice, 0)
		releasedDevs := make(map[string]models.ContainerReleasedDevice)
		for _, curDev := range devs {
			if curDev.IsolatedDevice == nil {
				continue
			}
			if curDev.IsolatedDevice.Id != dev.GetId() {
				tmpDev := curDev
				newDevs = append(newDevs, tmpDev)
			} else {
				releasedDevs[curDev.IsolatedDevice.Id] = *models.NewContainerReleasedDevice(curDev, dev.DevType, dev.Model)
			}
		}
		if err := ctrPtr.SaveReleasedDevices(ctx, userCred, releasedDevs); err != nil {
			return errors.Wrapf(err, "save release devices for container %s", ctr.GetId())
		}
		if _, err := db.Update(ctrPtr, func() error {
			ctrPtr.Spec.Devices = newDevs
			return nil
		}); err != nil {
			return errors.Wrapf(err, "update container %s devs", ctrPtr.GetId())
		}
	}
	return nil
}

func (p *SPodDriver) BeforeAttachIsolatedDevice(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, dev *models.SIsolatedDevice) error {
	ctrs, err := models.GetContainerManager().GetContainersByPod(guest.GetId())
	if err != nil {
		return errors.Wrapf(err, "get containers by pod %s", guest.GetId())
	}
	for _, ctr := range ctrs {
		ctrPtr := &ctr
		if err := p.attachIsolatedDeviceToContainer(ctx, userCred, ctrPtr, dev); err != nil {
			return errors.Wrapf(err, "attach isolated device to container %s", ctr.GetId())
		}
	}
	return nil
}

func (p *SPodDriver) attachIsolatedDeviceToContainer(ctx context.Context, userCred mcclient.TokenCredential, ctrPtr *models.SContainer, dev *models.SIsolatedDevice) error {
	rlsDevs, err := ctrPtr.GetReleasedDevices(ctx, userCred)
	if err != nil {
		return errors.Wrapf(err, "get release devices for container %s", ctrPtr.GetId())
	}
	spec := new(api.ContainerSpec)
	if err := jsonutils.Marshal(ctrPtr.Spec).Unmarshal(spec); err != nil {
		return errors.Wrap(err, "deep copy spec")
	}
	// attach it
	if spec.Devices == nil {
		spec.Devices = make([]*api.ContainerDevice, 0)
	}
	shouldUpdate := true
	for _, curDev := range spec.Devices {
		if curDev.IsolatedDevice == nil {
			continue
		}
		if curDev.IsolatedDevice.Id == dev.GetId() {
			shouldUpdate = false
			break
		}
	}
	if shouldUpdate {
		spec.Devices = append(spec.Devices, &api.ContainerDevice{
			Type: apis.CONTAINER_DEVICE_TYPE_ISOLATED_DEVICE,
			IsolatedDevice: &api.ContainerIsolatedDevice{
				Id: dev.GetId(),
			},
		})
		if _, err := db.Update(ctrPtr, func() error {
			ctrPtr.Spec = spec
			return nil
		}); err != nil {
			return errors.Wrapf(err, "update container %s devs", ctrPtr.GetId())
		}
	}
	for id, rlsDev := range rlsDevs {
		if rlsDev.IsolatedDevice == nil {
			continue
		}
		if rlsDev.DeviceModel == dev.Model && rlsDev.DeviceType == dev.DevType {
			delete(rlsDevs, id)
			if err := ctrPtr.SaveReleasedDevices(ctx, userCred, rlsDevs); err != nil {
				return errors.Wrapf(err, "save release devices for container %s", ctrPtr.GetId())
			}
			return nil
		}
	}
	return nil
}
