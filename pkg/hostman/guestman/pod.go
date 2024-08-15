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

package guestman

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/container/device"
	"yunion.io/x/onecloud/pkg/hostman/container/lifecycle"
	"yunion.io/x/onecloud/pkg/hostman/container/prober"
	proberesults "yunion.io/x/onecloud/pkg/hostman/container/prober/results"
	"yunion.io/x/onecloud/pkg/hostman/container/status"
	"yunion.io/x/onecloud/pkg/hostman/container/volume_mount"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	computemod "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	imagemod "yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/util/cgrouputils/cpuset"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/pod"
	"yunion.io/x/onecloud/pkg/util/pod/image"
	"yunion.io/x/onecloud/pkg/util/pod/logs"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func (m *SGuestManager) startContainerProbeManager() {
	livenessManager := proberesults.NewManager()
	startupManager := proberesults.NewManager()
	man := prober.NewManager(status.NewManager(), livenessManager, startupManager, newContainerRunner(m))
	m.containerProbeManager = man
	man.Start()
}

func (m *SGuestManager) GetContainerProbeManager() prober.Manager {
	return m.containerProbeManager
}

func newContainerRunner(man *SGuestManager) *containerRunner {
	return &containerRunner{man}
}

type containerRunner struct {
	manager *SGuestManager
}

func (cr *containerRunner) RunInContainer(pod *desc.SGuestDesc, containerId string, cmd []string, timeout time.Duration) ([]byte, error) {
	srv, ok := cr.manager.GetServer(pod.Uuid)
	if !ok {
		return nil, errors.Wrapf(httperrors.ErrNotFound, "server %s not found", pod.Uuid)
	}
	s := srv.(*sPodGuestInstance)
	ctrCriId, err := s.getContainerCRIId(containerId)
	if err != nil {
		return nil, errors.Wrap(err, "get container cri id")
	}
	cli := s.getCRI().GetRuntimeClient()
	resp, err := cli.ExecSync(context.Background(), &runtimeapi.ExecSyncRequest{
		ContainerId: ctrCriId,
		Cmd:         cmd,
		Timeout:     int64(timeout),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "exec sync %#v to %s", cmd, ctrCriId)
	}
	return append(resp.Stdout, resp.Stderr...), nil
}

type PodInstance interface {
	GuestRuntimeInstance

	CreateContainer(ctx context.Context, userCred mcclient.TokenCredential, id string, input *hostapi.ContainerCreateInput) (jsonutils.JSONObject, error)
	StartContainer(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, input *hostapi.ContainerCreateInput) (jsonutils.JSONObject, error)
	DeleteContainer(ctx context.Context, cred mcclient.TokenCredential, id string) (jsonutils.JSONObject, error)
	SyncContainerStatus(ctx context.Context, cred mcclient.TokenCredential, ctrId string) (jsonutils.JSONObject, error)
	StopContainer(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, body jsonutils.JSONObject) (jsonutils.JSONObject, error)
	PullImage(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, input *hostapi.ContainerPullImageInput) (jsonutils.JSONObject, error)
	SaveVolumeMountToImage(ctx context.Context, userCred mcclient.TokenCredential, input *hostapi.ContainerSaveVolumeMountToImageInput, ctrId string) (jsonutils.JSONObject, error)
	ExecContainer(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, input *computeapi.ContainerExecInput) (*url.URL, error)
	ContainerExecSync(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, input *computeapi.ContainerExecSyncInput) (jsonutils.JSONObject, error)

	ReadLogs(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, input *computeapi.PodLogOptions, stdout, stderr io.Writer) error
}

type sContainer struct {
	Id    string `json:"id"`
	Index int    `json:"index"`
	CRIId string `json:"cri_id"`
}

func newContainer(id string) *sContainer {
	return &sContainer{
		Id: id,
	}
}

type sPodGuestInstance struct {
	*sBaseGuestInstance
	containers map[string]*sContainer
}

func newPodGuestInstance(id string, man *SGuestManager) PodInstance {
	return &sPodGuestInstance{
		sBaseGuestInstance: newBaseGuestInstance(id, man, computeapi.HYPERVISOR_POD),
		containers:         make(map[string]*sContainer),
	}
}

func (s *sPodGuestInstance) CleanGuest(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	criId := s.getCRIId()
	if criId != "" {
		if err := s.getCRI().RemovePod(ctx, criId); err != nil {
			return nil, errors.Wrapf(err, "RemovePod with cri_id %q", criId)
		}
	}
	return nil, DeleteHomeDir(s)
}

func (s *sPodGuestInstance) ExitCleanup(clear bool) {
}

func (s *sPodGuestInstance) CleanDirtyGuest(ctx context.Context) error {
	_, err := s.CleanGuest(ctx, false)
	return err
}

func (s *sPodGuestInstance) ImportServer(pendingDelete bool) {
	// TODO: 参考SKVMGuestInstance，可以做更多的事，比如同步状态
	s.manager.SaveServer(s.Id, s)
	s.manager.RemoveCandidateServer(s)
	/*if s.IsDaemon() {
		ctx := context.Background()
		if !s.IsRunning() {
			if err := s.StartPod(ctx, hostutils.GetComputeSession(ctx).GetToken()); err != nil {
				log.Errorf("start pod (%s/%s) error: %v", s.GetId(), s.GetName(), err)
			}
		// TODO: start related containers and sync status
		}
	} else {
		s.SyncStatus("sync status after host started")
		s.getProbeManager().AddPod(s.Desc)
	}*/
	s.SyncStatus("sync status after host started")
	s.getProbeManager().AddPod(s.Desc)
}

func (s *sPodGuestInstance) SyncStatus(reason string) {
	// sync pod status
	var status = computeapi.VM_READY
	if s.IsRunning() {
		status = computeapi.VM_RUNNING
	}
	ctx := context.Background()
	if status == computeapi.VM_READY {
		// remove pod
		if err := s.stopPod(ctx, 1); err != nil {
			log.Warningf("stop cri pod when sync status: %s: %v", s.Id, err)
		}
	}
	statusInput := &apis.PerformStatusInput{
		Status:      status,
		Reason:      reason,
		PowerStates: GetPowerStates(s),
		HostId:      hostinfo.Instance().HostId,
	}

	if _, err := hostutils.UpdateServerStatus(ctx, s.Id, statusInput); err != nil {
		log.Errorf("failed update guest status %s", err)
	}
	// sync container's status
	for _, c := range s.containers {
		status, err := s.getContainerStatus(ctx, c.Id)
		if err != nil {
			log.Errorf("get container %s status of pod %s", c.Id, s.Id)
			continue
		}
		statusInput = &apis.PerformStatusInput{
			Status: status,
			Reason: reason,
			HostId: hostinfo.Instance().HostId,
		}
		if _, err := hostutils.UpdateContainerStatus(ctx, c.Id, statusInput); err != nil {
			log.Errorf("failed update container %s status: %s", c.Id, err)
		}
	}
}

func (s *sPodGuestInstance) DeployFs(ctx context.Context, userCred mcclient.TokenCredential, deployInfo *deployapi.DeployInfo) (jsonutils.JSONObject, error) {
	// update port_mappings
	/*podInput, err := s.getPodCreateParams()
	if err != nil {
		return nil, errors.Wrap(err, "getPodCreateParams")
	}
	if len(podInput.PortMappings) != 0 {
		pms, err := s.getPortMappings(podInput.PortMappings)
		if err != nil {
			return nil, errors.Wrap(err, "get port mappings")
		}
		if err := s.setPortMappings(ctx, userCred, s.convertToPodMetadataPortMappings(pms)); err != nil {
			return nil, errors.Wrap(err, "set port mappings")
		}
	}*/
	return nil, nil
}

func (s *sPodGuestInstance) IsStopped() bool {
	//TODO implement me
	panic("implement me")
}

func (s *sPodGuestInstance) IsSuspend() bool {
	return false
}

func (s *sPodGuestInstance) getCRI() pod.CRI {
	return s.manager.GetCRI()
}

func (s *sPodGuestInstance) getProbeManager() prober.Manager {
	return s.manager.GetContainerProbeManager()
}

func (s *sPodGuestInstance) getHostCPUMap() *pod.HostContainerCPUMap {
	return s.manager.GetContainerCPUMap()
}

func (s *sPodGuestInstance) getPod(ctx context.Context) (*runtimeapi.PodSandbox, error) {
	pods, err := s.getCRI().ListPods(ctx, pod.ListPodOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "ListPods")
	}
	for _, p := range pods {
		if p.Metadata.Uid == s.Id {
			return p, nil
		}
	}
	return nil, errors.Wrap(httperrors.ErrNotFound, "Not found pod from containerd")
}

func (s *sPodGuestInstance) IsRunning() bool {
	pod, err := s.getPod(context.Background())
	if err != nil {
		return false
	}
	if pod.GetState() == runtimeapi.PodSandboxState_SANDBOX_READY {
		return true
	}
	return false
}

func (s *sPodGuestInstance) HandleGuestStatus(ctx context.Context, status string, body *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	body.Set("status", jsonutils.NewString(status))
	hostutils.TaskComplete(ctx, body)
	return nil, nil
}

func (s *sPodGuestInstance) HandleGuestStart(ctx context.Context, userCred mcclient.TokenCredential, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	hostutils.DelayTaskWithWorker(ctx, func(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
		resp, err := s.startPod(ctx, userCred)
		if err != nil {
			return nil, errors.Wrap(err, "startPod")
		}
		return jsonutils.Marshal(resp), nil
	}, nil, s.manager.GuestStartWorker)
	return nil, nil
}

func (s *sPodGuestInstance) HandleStop(ctx context.Context, timeout int64) error {
	hostutils.DelayTask(ctx, func(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
		err := s.stopPod(ctx, timeout)
		if err != nil {
			return nil, errors.Wrap(err, "stopPod")
		}
		return nil, nil
	}, nil)
	return nil
}

func (s *sPodGuestInstance) getCreateParams() (jsonutils.JSONObject, error) {
	createParamsStr, ok := s.GetDesc().Metadata[computeapi.VM_METADATA_CREATE_PARAMS]
	if !ok {
		return nil, errors.Errorf("not found %s in metadata", computeapi.VM_METADATA_CREATE_PARAMS)
	}
	return jsonutils.ParseString(createParamsStr)
}

func (s *sPodGuestInstance) getPodCreateParams() (*computeapi.PodCreateInput, error) {
	createParams, err := s.getCreateParams()
	if err != nil {
		return nil, errors.Wrapf(err, "getCreateParams")
	}
	input := new(computeapi.PodCreateInput)
	if err := createParams.Unmarshal(input, "pod"); err != nil {
		return nil, errors.Wrapf(err, "unmarshal to pod creation input")
	}
	return input, nil
}

func (s *sPodGuestInstance) getPodLogDir() string {
	return filepath.Join(s.HomeDir(), "logs")
}

func (s *sPodGuestInstance) getShmDir() string {
	return filepath.Join(s.HomeDir(), "shm")
}

func (s *sPodGuestInstance) getContainerShmDir(containerName string) string {
	return filepath.Join(s.getShmDir(), fmt.Sprintf("%s-shm", containerName))
}

func (s *sPodGuestInstance) GetDisks() []*desc.SGuestDisk {
	return s.GetDesc().Disks
}

func (s *sPodGuestInstance) mountPodVolumes() error {
	for ctrId, vols := range s.getContainerVolumeMounts() {
		for _, vol := range vols {
			drv := volume_mount.GetDriver(vol.Type)
			if err := drv.Mount(s, ctrId, vol); err != nil {
				return errors.Wrapf(err, "mount volume %s, ctrId %s", jsonutils.Marshal(vol), ctrId)
			}
			if vol.FsUser != nil || vol.FsGroup != nil {
				// change mountpoint owner
				if err := volume_mount.ChangeDirOwner(s, drv, ctrId, vol); err != nil {
					return errors.Wrapf(err, "change dir owner")
				}
			}
		}
	}
	return nil
}

func (s *sPodGuestInstance) umountPodVolumes() error {
	for ctrId, vols := range s.getContainerVolumeMounts() {
		for _, vol := range vols {
			if err := volume_mount.GetDriver(vol.Type).Unmount(s, ctrId, vol); err != nil {
				return errors.Wrapf(err, "Unmount volume %s, ctrId %s", jsonutils.Marshal(vol), ctrId)
			}
		}
	}
	return nil
}

func (s *sPodGuestInstance) GetContainers() []*hostapi.ContainerDesc {
	return s.GetDesc().Containers
}

func (s *sPodGuestInstance) GetContainerById(ctrId string) *hostapi.ContainerDesc {
	ctrs := s.GetContainers()
	for i := range ctrs {
		ctr := ctrs[i]
		if ctr.Id == ctrId {
			return ctr
		}
	}
	return nil
}

func (s *sPodGuestInstance) getContainerVolumeMounts() map[string][]*hostapi.ContainerVolumeMount {
	result := make(map[string][]*hostapi.ContainerVolumeMount, 0)
	for _, ctr := range s.GetDesc().Containers {
		mnts, ok := result[ctr.Id]
		if !ok {
			mnts = make([]*hostapi.ContainerVolumeMount, 0)
		}
		for _, vol := range ctr.Spec.VolumeMounts {
			tmp := vol
			mnts = append(mnts, tmp)
		}
		result[ctr.Id] = mnts
	}
	return result
}

func (s *sPodGuestInstance) getContainerVolumeMountsByDiskId(ctrId, diskId string) []*hostapi.ContainerVolumeMount {
	ctrVols := s.getContainerVolumeMounts()
	vols, ok := ctrVols[ctrId]
	if !ok {
		return nil
	}
	volList := make([]*hostapi.ContainerVolumeMount, 0)
	for _, vol := range vols {
		if vol.Disk != nil {
			if vol.Disk.Id == diskId {
				tmpVol := vol
				volList = append(volList, tmpVol)
			}
		}
	}
	return volList
}

func (s *sPodGuestInstance) GetVolumesDir() string {
	return filepath.Join(s.HomeDir(), "volumes")
}

func (s *sPodGuestInstance) GetVolumesOverlayDir() string {
	return filepath.Join(s.GetVolumesDir(), "overlay")
}

func (s *sPodGuestInstance) GetDiskMountPoint(disk storageman.IDisk) string {
	return s.GetDiskMountPointById(disk.GetId())
}

func (s *sPodGuestInstance) GetDiskMountPointById(diskId string) string {
	return filepath.Join(s.GetVolumesDir(), diskId)
}

func (s *sPodGuestInstance) getPodPrivilegedMode(input *computeapi.PodCreateInput) bool {
	for _, ctr := range input.Containers {
		if ctr.Privileged {
			return true
		}
	}
	return false
}

func (s *sPodGuestInstance) getOtherPods() []*sPodGuestInstance {
	man := s.manager
	otherPods := make([]*sPodGuestInstance, 0)
	man.Servers.Range(func(id, value any) bool {
		if id == s.Id {
			return true
		}
		ins := value.(GuestRuntimeInstance)
		pod, ok := ins.(*sPodGuestInstance)
		if !ok {
			return true
		}
		otherPods = append(otherPods, pod)
		return true
	})
	return otherPods
}

func (s *sPodGuestInstance) getCgroupParent() string {
	return "/cloudpods"
}

type podStartTask struct {
	ctx      context.Context
	userCred mcclient.TokenCredential
	pod      *sPodGuestInstance
}

func newPodStartTask(ctx context.Context, userCred mcclient.TokenCredential, pod *sPodGuestInstance) *podStartTask {
	return &podStartTask{
		ctx:      ctx,
		userCred: userCred,
		pod:      pod,
	}
}

func (t *podStartTask) Run() {
	if _, err := t.pod.startPod(t.ctx, t.userCred); err != nil {
		log.Errorf("start pod(%s/%s) err: %s", t.pod.GetId(), t.pod.GetName(), err.Error())
	}
	t.pod.SyncStatus("sync status after pod start")
}

func (t *podStartTask) Dump() string {
	return fmt.Sprintf("pod start task %s/%s", t.pod.GetId(), t.pod.GetName())
}

func (s *sPodGuestInstance) StartPod(ctx context.Context, userCred mcclient.TokenCredential) error {
	s.manager.GuestStartWorker.Run(newPodStartTask(ctx, userCred, s), nil, nil)
	return nil
}

func (s *sPodGuestInstance) startPod(ctx context.Context, userCred mcclient.TokenCredential) (*computeapi.PodStartResponse, error) {
	retries := 3
	sec := 5 * time.Second
	var err error
	var resp *computeapi.PodStartResponse
	for i := 1; i <= retries; i++ {
		resp, err = s._startPod(ctx, userCred)
		if err == nil {
			return resp, nil
		}
		log.Errorf("start pod %s/%s error with %d times: %v", s.GetId(), s.GetName(), i, err)
		time.Sleep(sec * time.Duration(i))
	}
	return resp, err
}

func (s *sPodGuestInstance) _startPod(ctx context.Context, userCred mcclient.TokenCredential) (*computeapi.PodStartResponse, error) {
	podInput, err := s.getPodCreateParams()
	if err != nil {
		return nil, errors.Wrap(err, "getPodCreateParams")
	}
	if err := s.mountPodVolumes(); err != nil {
		return nil, errors.Wrap(err, "mountPodVolumes")
	}
	if err := s.ensurePodRemoved(ctx, 0); err != nil {
		log.Warningf("ensure pod removed before starting %s: %v", s.GetId(), err)
	}
	podCfg := &runtimeapi.PodSandboxConfig{
		Metadata: &runtimeapi.PodSandboxMetadata{
			Name:      s.GetDesc().Name,
			Uid:       s.GetId(),
			Namespace: s.GetDesc().TenantId,
			Attempt:   1,
		},
		Hostname:     s.GetDesc().Hostname,
		LogDirectory: s.getPodLogDir(),
		DnsConfig:    nil,
		PortMappings: nil,
		Labels:       nil,
		Annotations:  nil,
		Linux: &runtimeapi.LinuxPodSandboxConfig{
			CgroupParent: s.getCgroupParent(),
			SecurityContext: &runtimeapi.LinuxSandboxSecurityContext{
				NamespaceOptions:   nil,
				SelinuxOptions:     nil,
				RunAsUser:          nil,
				RunAsGroup:         nil,
				ReadonlyRootfs:     false,
				SupplementalGroups: nil,
				Privileged:         s.getPodPrivilegedMode(podInput),
				Seccomp: &runtimeapi.SecurityProfile{
					ProfileType: runtimeapi.SecurityProfile_Unconfined,
				},
				Apparmor: &runtimeapi.SecurityProfile{
					ProfileType: runtimeapi.SecurityProfile_Unconfined,
				},
				SeccompProfilePath: "",
			},
			Sysctls: nil,
		},
		Windows: nil,
	}

	// inject pod security context
	podSec := podInput.SecurityContext
	if podSec != nil {
		/*podCfg.Linux.Sysctls = map[string]string{
			"net.ipv4.ip_unprivileged_port_start": "80",
		}*/
		if podSec.RunAsUser != nil {
			podCfg.Linux.SecurityContext.RunAsUser = &runtimeapi.Int64Value{
				Value: *podSec.RunAsUser,
			}
		}
		if podSec.RunAsGroup != nil {
			podCfg.Linux.SecurityContext.RunAsGroup = &runtimeapi.Int64Value{
				Value: *podSec.RunAsGroup,
			}
		}
	}

	/*metaPms, err := s.GetPortMappings()
	if err != nil {
		return nil, errors.Wrap(err, "GetPortMappings")
	}
	if len(metaPms) != 0 {
		pms := make([]*runtimeapi.PortMapping, len(metaPms))
		for idx := range metaPms {
			pm := metaPms[idx]
			proto := runtimeapi.Protocol_TCP
			switch pm.Protocol {
			case computeapi.PodPortMappingProtocolTCP:
				proto = runtimeapi.Protocol_TCP
			case computeapi.PodPortMappingProtocolUDP:
				proto = runtimeapi.Protocol_UDP
			}
			pms[idx] = &runtimeapi.PortMapping{
				Protocol:      proto,
				ContainerPort: pm.ContainerPort,
				HostPort:      pm.HostPort,
				HostIp:        pm.HostIp,
			}
		}
		podCfg.PortMappings = pms
	}*/

	criId, err := s.getCRI().RunPod(ctx, podCfg, "")
	if err != nil {
		return nil, errors.Wrap(err, "cri.RunPod")
	}
	if err := s.setCRIInfo(ctx, userCred, criId, podCfg); err != nil {
		return nil, errors.Wrap(err, "setCRIId")
	}
	// set pod cgroup resources
	if err := s.setPodCgroupResources(criId, s.GetDesc().Mem, s.GetDesc().Cpu); err != nil {
		return nil, errors.Wrapf(err, "set pod %s cgroup memMB %d, cpu %d", criId, s.GetDesc().Mem, s.GetDesc().Cpu)
	}

	s.getProbeManager().AddPod(s.Desc)
	return &computeapi.PodStartResponse{
		CRIId:     criId,
		IsRunning: false,
	}, nil
}

func (s *sPodGuestInstance) setPodCgroupResources(criId string, memMB int64, cpuCnt int64) error {
	if err := s.getCGUtil().SetMemoryLimitBytes(criId, memMB*1024*1024); err != nil {
		return errors.Wrap(err, "set cgroup memory limit")
	}
	if err := s.getCGUtil().SetCPUCfs(criId, cpuCnt*s.getDefaultCPUPeriod(), s.getDefaultCPUPeriod()); err != nil {
		return errors.Wrap(err, "set cgroup cfs")
	}
	return nil
}

func (s *sPodGuestInstance) ensurePodRemoved(ctx context.Context, timeout int64) error {
	if timeout == 0 {
		timeout = 5
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	/*if err := s.getCRI().StopPod(ctx, &runtimeapi.StopPodSandboxRequest{
		PodSandboxId: s.getCRIId(),
	}); err != nil {
		return errors.Wrapf(err, "stop cri pod: %s", s.getCRIId())
	}*/
	criId := s.getCRIId()
	if criId != "" {
		if err := s.getCRI().RemovePod(ctx, s.getCRIId()); err != nil {
			return errors.Wrapf(err, "remove cri pod: %s", s.getCRIId())
		}
	}
	p, _ := s.getPod(ctx)
	if p != nil {
		if err := s.getCRI().RemovePod(ctx, p.GetId()); err != nil {
			return errors.Wrapf(err, "remove cri pod: %s", p.GetId())
		}
	}

	s.getProbeManager().RemovePod(s.Desc)
	return nil
}

func (s *sPodGuestInstance) stopPod(ctx context.Context, timeout int64) error {
	ReleaseGuestCpuset(s.manager, s)
	if err := s.umountPodVolumes(); err != nil {
		return errors.Wrapf(err, "umount pod volumes")
	}
	if timeout == 0 {
		timeout = 5
	}

	return s.ensurePodRemoved(ctx, timeout)
}

func (s *sPodGuestInstance) LoadDesc() error {
	if err := LoadDesc(s); err != nil {
		return errors.Wrap(err, "LoadDesc")
	}
	if err := s.loadContainers(); err != nil {
		return errors.Wrap(err, "loadContainers")
	}
	return nil
}

func (s *sPodGuestInstance) loadContainers() error {
	s.containers = make(map[string]*sContainer)
	ctrFile := s.getContainersFilePath()
	if !fileutils2.Exists(ctrFile) {
		log.Warningf("pod %s containers file %s doesn't exist", s.Id, ctrFile)
		return nil
	}
	ctrStr, err := ioutil.ReadFile(ctrFile)
	if err != nil {
		return errors.Wrapf(err, "read %s", ctrFile)
	}
	obj, err := jsonutils.Parse(ctrStr)
	if err != nil {
		return errors.Wrapf(err, "jsonutils.Parse %s", ctrStr)
	}
	ctrs := make(map[string]*sContainer)
	if err := obj.Unmarshal(ctrs); err != nil {
		return errors.Wrapf(err, "unmarshal %s to container map", obj.String())
	}
	s.containers = ctrs
	return nil
}

func (s *sPodGuestInstance) PostLoad(m *SGuestManager) error {
	return LoadGuestCpuset(m, s)
}

func (s *sPodGuestInstance) SyncConfig(ctx context.Context, guestDesc *desc.SGuestDesc, fwOnly bool) (jsonutils.JSONObject, error) {
	if err := SaveDesc(s, guestDesc); err != nil {
		return nil, errors.Wrap(err, "SaveDesc")
	}
	if err := SaveLiveDesc(s, s.Desc); err != nil {
		return nil, errors.Wrap(err, "SaveLiveDesc")
	}
	return nil, nil
}

func (s *sPodGuestInstance) getContainerMeta(id string) *sContainer {
	return s.containers[id]
}

func (s *sPodGuestInstance) getContainerCRIId(ctrId string) (string, error) {
	ctr := s.getContainerMeta(ctrId)
	if ctr == nil {
		return "", errors.Wrapf(errors.ErrNotFound, "Not found container %s", ctrId)
	}
	return ctr.CRIId, nil
}

func (s *sPodGuestInstance) StartContainer(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, input *hostapi.ContainerCreateInput) (jsonutils.JSONObject, error) {
	_, hasCtr := s.containers[ctrId]
	needRecreate := false
	if hasCtr {
		status, err := s.getContainerStatus(ctx, ctrId)
		if err != nil {
			if errors.Cause(err) == errors.ErrNotFound || strings.Contains(err.Error(), "not found") {
				needRecreate = true
			} else {
				return nil, errors.Wrap(err, "get container status")
			}
		} else {
			if status == computeapi.CONTAINER_STATUS_EXITED {
				needRecreate = true
			} else if status != computeapi.CONTAINER_STATUS_CREATED {
				return nil, errors.Wrapf(err, "can't start container when status is %s", status)
			}
		}
	}
	if !hasCtr || needRecreate {
		log.Infof("recreate container %s before starting. hasCtr: %v, needRecreate: %v", ctrId, hasCtr, needRecreate)
		// delete and recreate the container before starting
		if hasCtr {
			if _, err := s.DeleteContainer(ctx, userCred, ctrId); err != nil {
				return nil, errors.Wrap(err, "delete container before starting")
			}
		}
		if _, err := s.CreateContainer(ctx, userCred, ctrId, input); err != nil {
			return nil, errors.Wrap(err, "recreate container before starting")
		}
	}

	criId, err := s.getContainerCRIId(ctrId)
	if err != nil {
		return nil, errors.Wrap(err, "get container cri id")
	}
	if err := s.getCRI().StartContainer(ctx, criId); err != nil {
		return nil, errors.Wrap(err, "CRI.StartContainer")
	}
	if err := s.setContainerCgroupDevicesAllow(criId, input.Spec.CgroupDevicesAllow); err != nil {
		return nil, errors.Wrap(err, "set cgroup devices allow")
	}
	if err := s.doContainerStartPostLifecycle(ctx, criId, input); err != nil {
		return nil, errors.Wrap(err, "do container lifecycle")
	}
	return nil, nil
}

func (s *sPodGuestInstance) allocateCpuNumaPin() error {
	if len(s.Desc.CpuNumaPin) != 0 || len(s.Desc.VcpuPin) != 0 {
		return nil
	}

	var cpus = make([]int, 0)
	var perferNumaNode int8 = -1
	for i := range s.Desc.IsolatedDevices {
		if s.Desc.IsolatedDevices[i].NumaNode >= 0 {
			perferNumaNode = s.Desc.IsolatedDevices[i].NumaNode
			break
		}
	}

	nodeNumaCpus, err := s.manager.cpuSet.AllocCpuset(int(s.Desc.Cpu), s.Desc.Mem*1024, perferNumaNode)
	if err != nil {
		return err
	}
	for _, numaCpus := range nodeNumaCpus {
		cpus = append(cpus, numaCpus.Cpuset...)
	}

	if !s.manager.numaAllocate {
		s.Desc.VcpuPin = []desc.SCpuPin{
			{
				Vcpus: fmt.Sprintf("0-%d", s.Desc.Cpu-1),
				Pcpus: cpuset.NewCPUSet(cpus...).String(),
			},
		}
	}

	return SaveLiveDesc(s, s.Desc)
}

func (s *sPodGuestInstance) doContainerStartPostLifecycle(ctx context.Context, criId string, input *hostapi.ContainerCreateInput) error {
	ls := input.Spec.Lifecyle
	if ls == nil {
		return nil
	}
	if ls.PostStart == nil {
		return nil
	}
	drv := lifecycle.GetDriver(ls.PostStart.Type)
	if err := drv.Run(ctx, ls.PostStart, s.getCRI(), criId); err != nil {
		return errors.Wrapf(err, "run %s", ls.PostStart.Type)
	}
	return nil
}

func (s *sPodGuestInstance) StopContainer(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	criId, err := s.getContainerCRIId(ctrId)
	if err != nil {
		if errors.Cause(err) == errors.ErrNotFound {
			// cri id not found, should already stopped
			return nil, nil
		}
		return nil, errors.Wrap(err, "get container cri id")
	}
	var timeout int64 = 15
	if body.Contains("timeout") {
		timeout, _ = body.Int("timeout")
	}
	if body.Contains("shm_size_mb") {
		shmSizeMB, _ := body.Int("shm_size_mb")
		if shmSizeMB > 64 {
			name, err := body.GetString("container_name")
			if err != nil {
				return nil, errors.Wrapf(err, "not found name from body: %s", body)
			}
			if err := s.unmountDevShm(name); err != nil {
				return nil, errors.Wrapf(err, "unmount shm %s", name)
			}
		}
	}
	if err := s.getCRI().StopContainer(ctx, criId, timeout); err != nil {
		return nil, errors.Wrap(err, "CRI.StopContainer")
	}
	return nil, nil
}

func (s *sPodGuestInstance) getCRIId() string {
	return s.GetSourceDesc().Metadata[computeapi.POD_METADATA_CRI_ID]
}

func (s *sPodGuestInstance) convertToPodMetadataPortMappings(pms []*runtimeapi.PortMapping) []*computeapi.PodMetadataPortMapping {
	ret := make([]*computeapi.PodMetadataPortMapping, len(pms))
	for idx := range pms {
		pm := pms[idx]
		var proto computeapi.PodPortMappingProtocol = computeapi.PodPortMappingProtocolTCP
		if pm.Protocol == runtimeapi.Protocol_UDP {
			proto = computeapi.PodPortMappingProtocolUDP
		}
		ret[idx] = &computeapi.PodMetadataPortMapping{
			Protocol:      proto,
			ContainerPort: pm.ContainerPort,
			HostPort:      pm.HostPort,
			HostIp:        pm.HostIp,
		}
	}
	return ret
}

func (s *sPodGuestInstance) setPortMappings(ctx context.Context, userCred mcclient.TokenCredential, pms []*computeapi.PodMetadataPortMapping) error {
	pmStr := jsonutils.Marshal(pms).String()
	s.Desc.Metadata[computeapi.POD_METADATA_PORT_MAPPINGS] = pmStr
	session := auth.GetSession(ctx, userCred, options.HostOptions.Region)
	if _, err := computemod.Servers.SetMetadata(session, s.GetId(), jsonutils.Marshal(map[string]string{
		computeapi.POD_METADATA_PORT_MAPPINGS: pmStr,
	})); err != nil {
		return errors.Wrapf(err, "set cri_id of pod %s", s.GetId())
	}
	return SaveDesc(s, s.Desc)
}

func (s *sPodGuestInstance) setCRIInfo(ctx context.Context, userCred mcclient.TokenCredential, criId string, cfg *runtimeapi.PodSandboxConfig) error {
	s.Desc.Metadata[computeapi.POD_METADATA_CRI_ID] = criId
	cfgStr := jsonutils.Marshal(cfg).String()
	s.Desc.Metadata[computeapi.POD_METADATA_CRI_CONFIG] = cfgStr

	session := auth.GetSession(ctx, userCred, options.HostOptions.Region)
	if _, err := computemod.Servers.SetMetadata(session, s.GetId(), jsonutils.Marshal(map[string]string{
		computeapi.POD_METADATA_CRI_ID:     criId,
		computeapi.POD_METADATA_CRI_CONFIG: cfgStr,
	})); err != nil {
		return errors.Wrapf(err, "set cri_id of pod %s", s.GetId())
	}
	return SaveDesc(s, s.Desc)
}

func (s *sPodGuestInstance) setContainerCRIInfo(ctx context.Context, userCred mcclient.TokenCredential, ctrId, criId string) error {
	session := auth.GetSession(ctx, userCred, options.HostOptions.Region)
	if _, err := computemod.Containers.SetMetadata(session, ctrId, jsonutils.Marshal(map[string]string{
		computeapi.CONTAINER_METADATA_CRI_ID: criId,
	})); err != nil {
		return errors.Wrapf(err, "set cri_id of container %s", ctrId)
	}
	return nil
}

func (s *sPodGuestInstance) getPodSandboxConfig() (*runtimeapi.PodSandboxConfig, error) {
	cfgStr := s.GetSourceDesc().Metadata[computeapi.POD_METADATA_CRI_CONFIG]
	obj, err := jsonutils.ParseString(cfgStr)
	if err != nil {
		return nil, errors.Wrapf(err, "ParseString to json object: %s", cfgStr)
	}
	podCfg := new(runtimeapi.PodSandboxConfig)
	if err := obj.Unmarshal(podCfg); err != nil {
		return nil, errors.Wrap(err, "Unmarshal to PodSandboxConfig")
	}
	return podCfg, nil
}

func (s *sPodGuestInstance) GetPortMappings() (computeapi.GuestPortMappings, error) {
	srcDesc := s.GetSourceDesc()
	nics := srcDesc.Nics
	pms := make([]*computeapi.GuestPortMapping, 0)
	for _, nic := range nics {
		for _, pm := range nic.PortMappings {
			tmpPm := pm
			pms = append(pms, tmpPm)
		}
	}
	return pms, nil
}

func (s *sPodGuestInstance) saveContainer(id string, criId string) error {
	_, ok := s.containers[id]
	if ok {
		return errors.Errorf("container %s already exists", criId)
	}
	ctr := newContainer(id)
	ctr.CRIId = criId
	s.containers[id] = ctr
	if err := s.saveContainersFile(s.containers); err != nil {
		return errors.Wrap(err, "saveContainersFile")
	}
	return nil
}

func (s *sPodGuestInstance) saveContainersFile(containers map[string]*sContainer) error {
	content := jsonutils.Marshal(containers).String()
	if err := fileutils2.FilePutContents(s.getContainersFilePath(), content, false); err != nil {
		return errors.Wrapf(err, "put content %s to containers file", content)
	}
	return nil
}

func (s *sPodGuestInstance) getContainersFilePath() string {
	return path.Join(s.HomeDir(), "containers")
}

func (s *sPodGuestInstance) CreateContainer(ctx context.Context, userCred mcclient.TokenCredential, id string, input *hostapi.ContainerCreateInput) (jsonutils.JSONObject, error) {
	ctrCriId, err := s.createContainer(ctx, userCred, id, input)
	if err != nil {
		return nil, errors.Wrap(err, "CRI.CreateContainer")
	}
	if err := s.setContainerCRIInfo(ctx, userCred, id, ctrCriId); err != nil {
		return nil, errors.Wrap(err, "setContainerCRIInfo")
	}
	return nil, nil
}

func (s *sPodGuestInstance) getContainerLogPath(ctrId string) string {
	return filepath.Join(fmt.Sprintf("%s.log", ctrId))
}

func (s *sPodGuestInstance) getLxcfsMounts() []*runtimeapi.Mount {
	// lxcfsPath := "/var/lib/lxc/lxcfs"
	lxcfsPath := options.HostOptions.LxcfsPath
	const (
		procCpuinfo   = "/proc/cpuinfo"
		procDiskstats = "/proc/diskstats"
		procMeminfo   = "/proc/meminfo"
		procStat      = "/proc/stat"
		procSwaps     = "/proc/swaps"
		procUptime    = "/proc/uptime"
	)
	newLxcfsMount := func(fp string) *runtimeapi.Mount {
		return &runtimeapi.Mount{
			ContainerPath: fp,
			HostPath:      fmt.Sprintf("%s%s", lxcfsPath, fp),
		}
	}
	return []*runtimeapi.Mount{
		newLxcfsMount(procUptime),
		newLxcfsMount(procMeminfo),
		newLxcfsMount(procStat),
		newLxcfsMount(procCpuinfo),
		newLxcfsMount(procSwaps),
		newLxcfsMount(procDiskstats),
	}
}

func (s *sPodGuestInstance) getContainerMounts(ctrId string, input *hostapi.ContainerCreateInput) ([]*runtimeapi.Mount, error) {
	inputMounts := input.Spec.VolumeMounts
	if len(inputMounts) == 0 {
		return make([]*runtimeapi.Mount, 0), nil
	}
	mounts := make([]*runtimeapi.Mount, len(inputMounts))

	for idx, im := range inputMounts {
		mnt := &runtimeapi.Mount{
			ContainerPath:  im.MountPath,
			Readonly:       im.ReadOnly,
			SelinuxRelabel: im.SelinuxRelabel,
			Propagation:    volume_mount.GetRuntimeVolumeMountPropagation(im.Propagation),
		}
		hostPath, err := volume_mount.GetDriver(im.Type).GetRuntimeMountHostPath(s, ctrId, im)
		if err != nil {
			return nil, errors.Wrapf(err, "get runtime host mount path of %s", jsonutils.Marshal(im))
		}
		mnt.HostPath = hostPath
		mounts[idx] = mnt
	}
	return mounts, nil
}

func (s *sPodGuestInstance) getCGUtil() pod.CgroupUtil {
	return pod.NewPodCgroupV1Util(s.getCgroupParent())
}

func (s *sPodGuestInstance) setContainerCgroupDevicesAllow(ctrId string, allowStrs []string) error {
	return s.getCGUtil().SetDevicesAllow(ctrId, allowStrs)
}

func (s *sPodGuestInstance) getDefaultCPUPeriod() int64 {
	return 100000
}

func (s *sPodGuestInstance) createContainer(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, input *hostapi.ContainerCreateInput) (string, error) {
	podCfg, err := s.getPodSandboxConfig()
	if err != nil {
		return "", errors.Wrap(err, "getPodSandboxConfig")
	}
	spec := input.Spec
	mounts, err := s.getContainerMounts(ctrId, input)
	if err != nil {
		return "", errors.Wrap(err, "get container mounts")
	}
	if spec.SimulateCpu {
		systemCpuMounts, err := s.simulateContainerSystemCpu(ctx, ctrId)
		if err != nil {
			return "", errors.Wrapf(err, "simulate container system cpu")
		}
		newMounts := systemCpuMounts
		newMounts = append(newMounts, mounts...)
		mounts = newMounts
	}

	// process shm size
	if spec.ShmSizeMB > 64 {
		// mount empty dir
		shmPath, err := s.mountDevShm(input, spec.ShmSizeMB)
		if err != nil {
			return "", errors.Wrapf(err, "mount dev shm")
		}
		mounts = append(mounts, &runtimeapi.Mount{
			ContainerPath: "/dev/shm",
			HostPath:      shmPath,
		})
	}

	if err := s.allocateCpuNumaPin(); err != nil {
		return "", errors.Wrap(err, "allocateCpuNumaPin")
	}

	var cpuSetCpus string
	{
		cpuSets := sets.NewString()
		if len(s.Desc.CpuNumaPin) > 0 {
			for _, cpuNum := range s.GetDesc().CpuNumaPin {
				for _, cpuPin := range cpuNum.VcpuPin {
					cpuSets.Insert(fmt.Sprintf("%d", cpuPin.Pcpu))
				}
			}
			cpuSetCpus = strings.Join(cpuSets.List(), ",")
		} else if len(s.Desc.VcpuPin) > 0 {
			for _, vcpuPin := range s.Desc.VcpuPin {
				cpuSets.Insert(vcpuPin.Pcpus)
			}
			cpuSetCpus = strings.Join(cpuSets.List(), ",")
		}
	}

	ctrCfg := &runtimeapi.ContainerConfig{
		Metadata: &runtimeapi.ContainerMetadata{
			Name: input.Name,
		},
		Image: &runtimeapi.ImageSpec{
			Image: spec.Image,
		},
		Linux: &runtimeapi.LinuxContainerConfig{
			Resources: &runtimeapi.LinuxContainerResources{
				// REF: https://docs.docker.com/config/containers/resource_constraints/#configure-the-default-cfs-scheduler
				CpuPeriod: s.getDefaultCPUPeriod(),
				CpuQuota:  s.GetDesc().Cpu * s.getDefaultCPUPeriod(),
				//CpuShares:              defaultCPUPeriod,
				MemoryLimitInBytes:     s.GetDesc().Mem * 1024 * 1024,
				OomScoreAdj:            0,
				CpusetCpus:             cpuSetCpus,
				CpusetMems:             "",
				HugepageLimits:         nil,
				Unified:                nil,
				MemorySwapLimitInBytes: 0,
			},
			SecurityContext: &runtimeapi.LinuxContainerSecurityContext{
				Capabilities:       &runtimeapi.Capability{},
				Privileged:         spec.Privileged,
				NamespaceOptions:   nil,
				SelinuxOptions:     nil,
				RunAsUser:          nil,
				RunAsGroup:         nil,
				RunAsUsername:      "",
				ReadonlyRootfs:     false,
				SupplementalGroups: nil,
				NoNewPrivs:         true,
				MaskedPaths:        nil,
				ReadonlyPaths:      nil,
				Seccomp: &runtimeapi.SecurityProfile{
					ProfileType: runtimeapi.SecurityProfile_Unconfined,
				},
				Apparmor: &runtimeapi.SecurityProfile{
					ProfileType: runtimeapi.SecurityProfile_Unconfined,
				},
				ApparmorProfile:    "",
				SeccompProfilePath: "",
			},
		},
		LogPath: s.getContainerLogPath(ctrId),
		Envs:    make([]*runtimeapi.KeyValue, 0),
		Devices: []*runtimeapi.Device{},
		Mounts:  mounts,
	}

	// inherit security context
	if spec.SecurityContext != nil {
		secInput := spec.SecurityContext
		if secInput.RunAsUser != nil {
			ctrCfg.Linux.SecurityContext.RunAsUser = &runtimeapi.Int64Value{
				Value: *secInput.RunAsUser,
			}
		}
		if secInput.RunAsGroup != nil {
			ctrCfg.Linux.SecurityContext.RunAsGroup = &runtimeapi.Int64Value{
				Value: *secInput.RunAsGroup,
			}
		}
	}

	if spec.EnableLxcfs {
		ctrCfg.Mounts = append(ctrCfg.Mounts, s.getLxcfsMounts()...)
	}
	if spec.Capabilities != nil {
		ctrCfg.Linux.SecurityContext.Capabilities.AddCapabilities = spec.Capabilities.Add
		ctrCfg.Linux.SecurityContext.Capabilities.DropCapabilities = spec.Capabilities.Drop
	}
	for _, env := range spec.Envs {
		ctrCfg.Envs = append(ctrCfg.Envs, &runtimeapi.KeyValue{
			Key:   env.Key,
			Value: env.Value,
		})
	}
	pms, err := s.GetPortMappings()
	if err != nil {
		return "", errors.Wrapf(err, "get pod port mappings")
	}
	if len(pms) != 0 {
		for _, pm := range pms {
			envKey := fmt.Sprintf("CLOUDPODS_%s_PORT_%d", strings.ToUpper(string(pm.Protocol)), pm.Port)
			envVal := fmt.Sprintf("%d", *pm.HostPort)
			ctrCfg.Envs = append(ctrCfg.Envs, &runtimeapi.KeyValue{
				Key:   envKey,
				Value: envVal,
			})
		}
	}
	if s.GetDesc().HostAccessIp != "" {
		ctrCfg.Envs = append(ctrCfg.Envs, &runtimeapi.KeyValue{
			Key:   "CLOUDPODS_HOST_ACCESS_IP",
			Value: s.GetDesc().HostAccessIp,
		})
	}
	if s.GetDesc().HostEIP != "" {
		ctrCfg.Envs = append(ctrCfg.Envs, &runtimeapi.KeyValue{
			Key:   "CLOUDPODS_HOST_EIP",
			Value: s.GetDesc().HostEIP,
		})
	}
	if len(spec.Devices) != 0 {
		devsByType := map[apis.ContainerDeviceType][]*hostapi.ContainerDevice{}
		for i := range spec.Devices {
			if devs, ok := devsByType[spec.Devices[i].Type]; ok {
				devsByType[spec.Devices[i].Type] = append(devs, spec.Devices[i])
			} else {
				devsByType[spec.Devices[i].Type] = []*hostapi.ContainerDevice{spec.Devices[i]}
			}
		}

		for cdType, devs := range devsByType {
			ctrDevs, err := device.GetDriver(cdType).GetRuntimeDevices(input, devs)
			if err != nil {
				return "", errors.Wrapf(err, "GetRuntimeDevices of %s", jsonutils.Marshal(devs))
			}
			ctrCfg.Devices = append(ctrCfg.Devices, ctrDevs...)
		}
		if err := s.getIsolatedDeviceExtraConfig(spec, ctrCfg); err != nil {
			return "", err
		}
	}
	if len(spec.Command) != 0 {
		ctrCfg.Command = spec.Command
	}
	if len(spec.Args) != 0 {
		ctrCfg.Args = spec.Args
	}
	criId, err := s.getCRI().CreateContainer(ctx, s.getCRIId(), podCfg, ctrCfg, false)
	if err != nil {
		return "", errors.Wrap(err, "cri.CreateContainer")
	}
	if err := s.saveContainer(ctrId, criId); err != nil {
		return "", errors.Wrap(err, "saveContainer")
	}
	return criId, nil
}

func (s *sPodGuestInstance) getIsolatedDeviceExtraConfig(spec *hostapi.ContainerSpec, ctrCfg *runtimeapi.ContainerConfig) error {
	devTypes := []isolated_device.ContainerDeviceType{
		isolated_device.ContainerDeviceTypeNvidiaGpu,
		isolated_device.ContainerDeviceTypeNvidiaMps,
		isolated_device.ContainerDeviceTypeAscendNpu,
	}
	for _, devType := range devTypes {
		devMan, err := isolated_device.GetContainerDeviceManager(devType)
		if err != nil {
			return errors.Wrapf(err, "GetContainerDeviceManager by type %q", devType)
		}
		envs, mounts := devMan.GetContainerExtraConfigures(spec.Devices)
		if len(envs) > 0 {
			ctrCfg.Envs = append(ctrCfg.Envs, envs...)
		}
		if len(mounts) > 0 {
			ctrCfg.Mounts = append(ctrCfg.Mounts, mounts...)
		}
	}
	return nil
}

func (s *sPodGuestInstance) getContainerSystemCpusDir(ctrId string) string {
	return filepath.Join(s.HomeDir(), "cpus", ctrId)
}

func (s *sPodGuestInstance) ensureContainerSystemCpuDir(cpuDir string, cpuCnt int64) error {
	// create cpu dir like /var/lib/docker/cpus/$ctr_name
	if err := pod.EnsureContainerSystemCpuDir(cpuDir, cpuCnt); err != nil {
		return errors.Wrap(err, "ensure container system cpu dir")
	}

	return nil
}

func (s *sPodGuestInstance) findHostCpuPath(ctrId string, cpuIndex int) (int, error) {
	return s.getHostCPUMap().Get(ctrId, cpuIndex)
}

func (s *sPodGuestInstance) simulateContainerSystemCpu(ctx context.Context, ctrId string) ([]*runtimeapi.Mount, error) {
	cpuDir := s.getContainerSystemCpusDir(ctrId)
	cpuCnt := s.GetDesc().Cpu
	if err := s.ensureContainerSystemCpuDir(cpuDir, cpuCnt); err != nil {
		return nil, err
	}
	sysCpuPath := "/sys/devices/system/cpu"
	ret := []*runtimeapi.Mount{
		{
			ContainerPath: sysCpuPath,
			HostPath:      cpuDir,
		},
	}
	for i := 0; i < int(cpuCnt); i++ {
		hostCpuIdx, err := s.findHostCpuPath(ctrId, i)
		if err != nil {
			return nil, errors.Wrapf(err, "find host cpu by container %s with index %d", ctrId, i)
		}
		hostCpuPath := filepath.Join(sysCpuPath, fmt.Sprintf("cpu%d", hostCpuIdx))
		ret = append(ret, &runtimeapi.Mount{
			ContainerPath: filepath.Join(sysCpuPath, fmt.Sprintf("cpu%d", i)),
			HostPath:      hostCpuPath,
		})
	}
	pathMap := func(baseName string) *runtimeapi.Mount {
		p := filepath.Join(sysCpuPath, baseName)
		return &runtimeapi.Mount{
			ContainerPath: p,
			HostPath:      p,
			Readonly:      true,
		}
	}
	for _, baseName := range []string{
		"modalias",
		"power",
		"cpuidle",
		"hotplug",
		"isolated",
		"cpufreq",
		"uevent",
	} {
		ret = append(ret, pathMap(baseName))
	}
	return ret, nil
}

func (s *sPodGuestInstance) DeleteContainer(ctx context.Context, userCred mcclient.TokenCredential, ctrId string) (jsonutils.JSONObject, error) {
	criId, err := s.getContainerCRIId(ctrId)
	if err != nil && errors.Cause(err) != errors.ErrNotFound {
		return nil, errors.Wrap(err, "getContainerCRIId")
	}
	if criId != "" {
		if err := s.getCRI().RemoveContainer(ctx, criId); err != nil && !strings.Contains(err.Error(), "not found") {
			return nil, errors.Wrap(err, "cri.RemoveContainer")
		}
	}
	// refresh local containers file
	delete(s.containers, ctrId)
	if err := s.saveContainersFile(s.containers); err != nil {
		return nil, errors.Wrap(err, "saveContainersFile")
	}
	if err := s.getHostCPUMap().Delete(ctrId); err != nil {
		log.Warningf("delete container %s cpu map: %v", ctrId, err)
	}
	return nil, nil
}

func (s *sPodGuestInstance) getContainerStatus(ctx context.Context, ctrId string) (string, error) {
	criId, err := s.getContainerCRIId(ctrId)
	if err != nil {
		if errors.Cause(err) == errors.ErrNotFound {
			// not found, already stopped
			return computeapi.CONTAINER_STATUS_EXITED, nil
		}
		return "", errors.Wrapf(err, "get container cri_id by %s", ctrId)
	}
	resp, err := s.getCRI().ContainerStatus(ctx, criId)
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") {
			return computeapi.CONTAINER_STATUS_EXITED, nil
		}
		return "", errors.Wrap(err, "cri.ContainerStatus")
	}
	status := computeapi.CONTAINER_STATUS_UNKNOWN
	switch resp.Status.State {
	case runtimeapi.ContainerState_CONTAINER_CREATED:
		status = computeapi.CONTAINER_STATUS_CREATED
	case runtimeapi.ContainerState_CONTAINER_RUNNING:
		status = computeapi.CONTAINER_STATUS_RUNNING
	case runtimeapi.ContainerState_CONTAINER_EXITED:
		status = computeapi.CONTAINER_STATUS_EXITED
	case runtimeapi.ContainerState_CONTAINER_UNKNOWN:
		status = computeapi.CONTAINER_STATUS_UNKNOWN
	}
	if status == computeapi.CONTAINER_STATUS_RUNNING {
		ctr := s.GetContainerById(ctrId)
		if ctr == nil {
			return "", errors.Wrapf(httperrors.ErrNotFound, "not found container by id %s", ctrId)
		}
		if ctr.Spec.NeedProbe() {
			status = computeapi.CONTAINER_STATUS_PROBING
		}
	}
	return status, nil
}

func (s *sPodGuestInstance) SyncContainerStatus(ctx context.Context, userCred mcclient.TokenCredential, ctrId string) (jsonutils.JSONObject, error) {
	status, err := s.getContainerStatus(ctx, ctrId)
	if err != nil {
		return nil, errors.Wrap(err, "get container status")
	}
	return jsonutils.Marshal(computeapi.ContainerSyncStatusResponse{Status: status}), nil
}

func (s *sPodGuestInstance) PullImage(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, input *hostapi.ContainerPullImageInput) (jsonutils.JSONObject, error) {
	policy := input.PullPolicy
	if policy == apis.ImagePullPolicyIfNotPresent || policy == "" {
		// check if image is presented
		img, err := s.getCRI().ImageStatus(ctx, &runtimeapi.ImageStatusRequest{
			Image: &runtimeapi.ImageSpec{
				Image: input.Image,
			},
		})
		if err != nil {
			return nil, errors.Wrapf(err, "cri.ImageStatus %s", input.Image)
		}
		if img.Image != nil {
			log.Infof("image %s already exists, skipping pulling it when policy is %s", input.Image, policy)
			return jsonutils.Marshal(&runtimeapi.PullImageResponse{
				ImageRef: img.Image.Id,
			}), nil
		}
	}
	return s.pullImageByCtrCmd(ctx, userCred, ctrId, input)
	// return s.pullImageByCRI(ctx, userCred, ctrId, input)
}

func (s *sPodGuestInstance) pullImageByCtrCmd(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, input *hostapi.ContainerPullImageInput) (jsonutils.JSONObject, error) {
	opt := &image.PullOptions{
		SkipVerify: true,
	}
	if input.Auth != nil {
		opt.Username = input.Auth.Username
		opt.Password = input.Auth.Password
	}
	addr := options.HostOptions.ContainerRuntimeEndpoint
	addr = strings.TrimPrefix(addr, "unix://")
	imgTool := image.NewImageTool(addr, "k8s.io")
	output, err := imgTool.Pull(input.Image, opt)
	errs := make([]error, 0)
	if err != nil {
		// try http protocol
		errs = append(errs, errors.Wrapf(err, "pullImageByCtrCmd: %s", output))
		opt.PlainHttp = true
		log.Infof("try pull image %s by http", input.Image)
		if output2, err := imgTool.Pull(input.Image, opt); err != nil {
			errs = append(errs, errors.Wrapf(err, "pullImageByCtrCmd by http: %s", output2))
			return nil, errors.NewAggregate(errs)
		}
	}
	return jsonutils.Marshal(&runtimeapi.PullImageResponse{
		ImageRef: input.Image,
	}), nil
}

func (s *sPodGuestInstance) pullImageByCRI(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, input *hostapi.ContainerPullImageInput) (jsonutils.JSONObject, error) {
	/*podCfg, err := s.getPodSandboxConfig()
	if err != nil {
		return nil, errors.Wrap(err, "get pod sandbox config")
	}*/
	req := &runtimeapi.PullImageRequest{
		Image: &runtimeapi.ImageSpec{
			Image: input.Image,
		},
		// SandboxConfig: podCfg,
	}
	if input.Auth != nil {
		authCfg := &runtimeapi.AuthConfig{
			Username:      input.Auth.Username,
			Password:      input.Auth.Password,
			Auth:          input.Auth.Auth,
			ServerAddress: input.Auth.ServerAddress,
			IdentityToken: input.Auth.IdentityToken,
			RegistryToken: input.Auth.RegistryToken,
		}
		req.Auth = authCfg
	}
	resp, err := s.getCRI().PullImage(ctx, req)
	if err != nil {
		return nil, errors.Wrapf(err, "cri.PullImage %s", input.Image)
	}
	return jsonutils.Marshal(resp), nil
}

func (s *sPodGuestInstance) SaveVolumeMountToImage(ctx context.Context, userCred mcclient.TokenCredential, input *hostapi.ContainerSaveVolumeMountToImageInput, ctrId string) (jsonutils.JSONObject, error) {
	vol := input.VolumeMount
	drv := volume_mount.GetDriver(vol.Type)
	if err := drv.Mount(s, ctrId, vol); err != nil {
		return nil, errors.Wrapf(err, "mount volume %s, ctrId %s", jsonutils.Marshal(vol), ctrId)
	}
	defer func() {
		if err := drv.Unmount(s, ctrId, vol); err != nil {
			log.Warningf("unmount volume %s: %v", jsonutils.Marshal(vol), err)
		}
	}()

	hostPath, err := drv.GetRuntimeMountHostPath(s, ctrId, vol)
	if err != nil {
		return nil, errors.Wrapf(err, "get runtime host mount path of %s", jsonutils.Marshal(vol))
	}
	// 1. tar hostPath to tgz
	imgPath, err := s.tarGzDir(input, ctrId, hostPath)
	if err != nil {
		return nil, errors.Wrapf(err, "tar and zip directory %s", hostPath)
	}
	defer func() {
		out, err := procutils.NewRemoteCommandAsFarAsPossible("rm", "-f", imgPath).Output()
		if err != nil {
			log.Warningf("rm -f %s: %s", imgPath, out)
		}
	}()

	// 2. upload target tgz to glance
	if err := s.saveTarGzToGlance(ctx, input, imgPath); err != nil {
		return nil, errors.Wrapf(err, "saveTarGzToGlance: %s", imgPath)
	}
	return nil, nil
}

func (s *sPodGuestInstance) tarGzDir(input *hostapi.ContainerSaveVolumeMountToImageInput, ctrId string, hostPath string) (string, error) {
	fp := fmt.Sprintf("volimg-%s-ctr-%s-%d.tar.gz", input.ImageId, ctrId, input.VolumeMountIndex)
	outputFp := filepath.Join(s.GetVolumesDir(), fp)
	cmd := fmt.Sprintf("tar -czf %s -C %s .", outputFp, hostPath)
	if out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", cmd).Output(); err != nil {
		return "", errors.Wrapf(err, "%s: %s", cmd, out)
	}
	return outputFp, nil
}

func (s *sPodGuestInstance) saveTarGzToGlance(ctx context.Context, input *hostapi.ContainerSaveVolumeMountToImageInput, imgPath string) error {
	f, err := os.Open(imgPath)
	if err != nil {
		return err
	}
	defer f.Close()
	finfo, err := f.Stat()
	if err != nil {
		return err
	}
	size := finfo.Size()

	var params = jsonutils.NewDict()
	params.Set("image_id", jsonutils.NewString(input.ImageId))

	if _, err := imagemod.Images.Upload(hostutils.GetImageSession(ctx), params, f, size); err != nil {
		return errors.Wrap(err, "upload image")
	}

	return err
}

func (s *sPodGuestInstance) ExecContainer(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, input *computeapi.ContainerExecInput) (*url.URL, error) {
	rCli := s.getCRI().GetRuntimeClient()
	criId, err := s.getContainerCRIId(ctrId)
	if err != nil {
		return nil, errors.Wrap(err, "get container cri id")
	}
	req := &runtimeapi.ExecRequest{
		ContainerId: criId,
		Cmd:         input.Command,
		Tty:         input.Tty,
		Stdin:       true,
		Stdout:      true,
		//Stderr:      true,
	}
	resp, err := rCli.Exec(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "exec")
	}
	return url.Parse(resp.Url)
}

func (s *sPodGuestInstance) mountDevShm(input *hostapi.ContainerCreateInput, mb int) (string, error) {
	shmPath := s.getContainerShmDir(input.Name)
	if !fileutils2.Exists(shmPath) {
		out, err := procutils.NewRemoteCommandAsFarAsPossible("mkdir", "-p", shmPath).Output()
		if err != nil {
			return "", errors.Wrapf(err, "mkdir -p %s: %s", shmPath, out)
		}
	}
	if err := procutils.NewRemoteCommandAsFarAsPossible("mountpoint", shmPath).Run(); err == nil {
		log.Warningf("mountpoint %s is already mounted", shmPath)
		return "", nil
	}
	out, err := procutils.NewRemoteCommandAsFarAsPossible("mount", "-t", "tmpfs", "-o", fmt.Sprintf("size=%dM", mb), "tmpfs", shmPath).Output()
	if err != nil {
		return "", errors.Wrapf(err, "mount tmpfs %s: %s", shmPath, out)
	}
	return shmPath, nil
}

func (s *sPodGuestInstance) unmountDevShm(containerName string) error {
	shmPath := s.getContainerShmDir(containerName)
	if err := procutils.NewRemoteCommandAsFarAsPossible("mountpoint", shmPath).Run(); err != nil {
		return nil
	}
	out, err := procutils.NewRemoteCommandAsFarAsPossible("umount", shmPath).Output()
	if err != nil {
		return errors.Wrapf(err, "mount tmpfs %s: %s", shmPath, out)
	}
	return nil
}

func (s *sPodGuestInstance) DoSnapshot(ctx context.Context, params *SDiskSnapshot) (jsonutils.JSONObject, error) {
	if s.IsRunning() {
		return nil, errors.Errorf("Pod dosen't support live snapshot")
	}
	if params.BackupDiskConfig == nil {
		return nil, httperrors.NewMissingParameterError("missing backup_disk_config")
	}
	if params.BackupDiskConfig.BackupAsTar == nil {
		return nil, httperrors.NewMissingParameterError("missing backup_disk_config.backup_as_tar")
	}
	input := params.BackupDiskConfig.BackupAsTar
	if input.ContainerId == "" {
		return nil, httperrors.NewMissingParameterError("missing backup_disk_config.backup_as_tar.container_id")
	}
	vols := s.getContainerVolumeMountsByDiskId(input.ContainerId, params.Disk.GetId())
	if len(vols) == 0 {
		return nil, httperrors.NewNotFoundError("not found container volume_mount by container_id %s and disk_id %s", input.ContainerId, params.Disk.GetId())
	}

	tmpBackRootDir, err := storageman.EnsureBackupDir()
	if err != nil {
		return nil, errors.Wrap(err, "EnsureBackupDir")
	}
	defer storageman.CleanupDirOrFile(tmpBackRootDir)
	for _, vol := range vols {
		drv := volume_mount.GetDriver(vol.Type)
		if err := drv.Mount(s, input.ContainerId, vol); err != nil {
			return nil, errors.Wrapf(err, "mount %s to %s", input.ContainerId, jsonutils.Marshal(vol))
		}
		mntPath, err := drv.GetRuntimeMountHostPath(s, input.ContainerId, vol)
		if err != nil {
			return nil, errors.Wrapf(err, "GetRuntimeMountHostPath containerId: %s, vol: %s", input.ContainerId, jsonutils.Marshal(vol))
		}
		isMntPathFile := false
		targetBindMntPath := tmpBackRootDir
		if vol.Disk.SubDirectory != "" {
			// mkdir tmpBackRootDir/subdirectory
			targetBindMntPath = filepath.Join(tmpBackRootDir, vol.Disk.SubDirectory)
		}
		if vol.Disk.StorageSizeFile != "" {
			targetBindMntPath = filepath.Join(tmpBackRootDir, vol.Disk.StorageSizeFile)
			isMntPathFile = true
		}
		if isMntPathFile {
			if out, err := procutils.NewRemoteCommandAsFarAsPossible("touch", targetBindMntPath).Output(); err != nil {
				return nil, errors.Wrapf(err, "touch %s: %s", targetBindMntPath, out)
			}
		} else {
			if out, err := procutils.NewRemoteCommandAsFarAsPossible("mkdir", "-p", targetBindMntPath).Output(); err != nil {
				return nil, errors.Wrapf(err, "mkdir -p %s: %s", targetBindMntPath, out)
			}
		}
		// do bind mount
		out, err := procutils.NewRemoteCommandAsFarAsPossible("mount", "--bind", mntPath, targetBindMntPath).Output()
		if err != nil {
			return nil, errors.Wrapf(err, "bind mount %s to %s: %s", mntPath, targetBindMntPath, out)
		}
	}
	snapshotPath, err := s.createSnapshot(params, tmpBackRootDir)
	if err != nil {
		return nil, errors.Wrap(err, "create snapshot")
	}
	for _, vol := range vols {
		// unbind mount
		// umount
		targetBindMntPath := filepath.Join(tmpBackRootDir, vol.Disk.SubDirectory)
		if vol.Disk.StorageSizeFile != "" {
			targetBindMntPath = filepath.Join(tmpBackRootDir, vol.Disk.StorageSizeFile)
		}
		out, err := procutils.NewRemoteCommandAsFarAsPossible("umount", targetBindMntPath).Output()
		if err != nil {
			return nil, errors.Wrapf(err, "umount bind point %s: %s", targetBindMntPath, out)
		}
	}
	for _, vol := range vols {
		drv := volume_mount.GetDriver(vol.Type)
		if err := drv.Unmount(s, input.ContainerId, vol); err != nil {
			return nil, errors.Wrapf(err, "unmount %s to %s", input.ContainerId, jsonutils.Marshal(vol))
		}
	}
	res := jsonutils.NewDict()
	res.Set("location", jsonutils.NewString(snapshotPath))
	return res, nil
}

func (s *sPodGuestInstance) createSnapshot(params *SDiskSnapshot, hostPath string) (string, error) {
	d := params.Disk
	snapshotDir := d.GetSnapshotDir()
	log.Infof("snapshotDir of LocalDisk %s: %s", d.GetId(), snapshotDir)
	if !fileutils2.Exists(snapshotDir) {
		output, err := procutils.NewCommand("mkdir", "-p", snapshotDir).Output()
		if err != nil {
			log.Errorf("mkdir %s failed: %s", snapshotDir, output)
			return "", errors.Wrapf(err, "mkdir %s failed: %s", snapshotDir, output)
		}
	}
	snapshotPath := s.getSnapshotPath(d, params.SnapshotId)
	// tar hostPath to snapshotPath
	input := params.BackupDiskConfig.BackupAsTar
	if err := s.tarHostDir(hostPath, snapshotPath, input.IncludeFiles, input.ExcludeFiles); err != nil {
		return "", errors.Wrapf(err, "tar host dir %s to %s", hostPath, snapshotPath)
	}
	return snapshotPath, nil
}

func (s *sPodGuestInstance) tarHostDir(srcDir, targetPath string, includeFiles, excludeFiles []string) error {
	baseCmd := "tar"
	for _, exclude := range excludeFiles {
		baseCmd = fmt.Sprintf("%s --exclude='%s'", baseCmd, exclude)
	}
	includeStr := "."
	if len(includeFiles) > 0 {
		includeStr = strings.Join(includeFiles, " ")
	}
	cmd := fmt.Sprintf("%s -cf %s -C %s %s", baseCmd, targetPath, srcDir, includeStr)
	log.Infof("tar cmd: %s", cmd)
	if out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", cmd).Output(); err != nil {
		return errors.Wrapf(err, "%s: %s", cmd, out)
	}
	return nil
}

func (s *sPodGuestInstance) getSnapshotPath(d storageman.IDisk, snapshotId string) string {
	snapshotDir := d.GetSnapshotDir()
	snapshotPath := path.Join(snapshotDir, fmt.Sprintf("%s.tar", snapshotId))
	return snapshotPath
}

func (s *sPodGuestInstance) DeleteSnapshot(ctx context.Context, params *SDeleteDiskSnapshot) (jsonutils.JSONObject, error) {
	snapshotPath := s.getSnapshotPath(params.Disk, params.DeleteSnapshot)
	out, err := procutils.NewCommand("rm", "-f", snapshotPath).Output()
	if err != nil {
		return nil, errors.Wrapf(err, "rm -f %s: %s", snapshotPath, out)
	}
	res := jsonutils.NewDict()
	res.Set("deleted", jsonutils.JSONTrue)
	return res, nil
}

func (s *sPodGuestInstance) ContainerExecSync(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, input *computeapi.ContainerExecSyncInput) (jsonutils.JSONObject, error) {
	ctrCriId, err := s.getContainerCRIId(ctrId)
	if err != nil {
		return nil, errors.Wrap(err, "get container cri id")
	}
	cli := s.getCRI().GetRuntimeClient()
	resp, err := cli.ExecSync(ctx, &runtimeapi.ExecSyncRequest{
		ContainerId: ctrCriId,
		Cmd:         input.Command,
		Timeout:     input.Timeout,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "exec sync %#v to %s", input.Command, ctrCriId)
	}
	return jsonutils.Marshal(&computeapi.ContainerExecSyncResponse{
		Stdout:   string(resp.Stdout),
		Stderr:   string(resp.Stderr),
		ExitCode: resp.ExitCode,
	}), nil
}

func (s *sPodGuestInstance) ReadLogs(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, input *computeapi.PodLogOptions, stdout, stderr io.Writer) error {
	// Do a zero-byte write to stdout before handing off to the container runtime.
	// This ensures at least one Write call is made to the writer when copying starts,
	// even if we then block waiting for log output from the container.
	if _, err := stdout.Write([]byte{}); err != nil {
		return err
	}
	ctrCriId, err := s.getContainerCRIId(ctrId)
	if err != nil {
		return errors.Wrapf(err, "get container cri id %s", ctrId)
	}
	resp, err := s.getCRI().ContainerStatus(ctx, ctrCriId)
	if err != nil {
		return errors.Wrapf(err, "get container status %s", ctrCriId)
	}
	logPath := resp.GetStatus().GetLogPath()
	opts := logs.NewLogOptions(input, time.Now())
	return logs.ReadLogs(ctx, logPath, ctrCriId, opts, s.getCRI().GetRuntimeClient(), stdout, stderr)
}
