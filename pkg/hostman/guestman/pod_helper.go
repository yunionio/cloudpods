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
	"strings"

	"github.com/shirou/gopsutil/v3/disk"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/container/volume_mount"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/pod/image"
	"yunion.io/x/onecloud/pkg/util/pod/nerdctl"
)

func IsContainerNotFoundError(err error) bool {
	if errors.Cause(err) == errors.ErrNotFound {
		return true
	}
	for _, errMsg := range []string{
		"NotFound",
		"not found",
	} {
		if strings.Contains(err.Error(), errMsg) {
			return true
		}
	}
	return false
}

func GetContainerdConnectionInfo() (string, string) {
	addr := options.HostOptions.ContainerRuntimeEndpoint
	addr = strings.TrimPrefix(addr, "unix://")
	namespace := "k8s.io"
	return addr, namespace
}

func NewContainerdImageTool() image.ImageTool {
	addr, namespace := GetContainerdConnectionInfo()
	return image.NewImageTool(addr, namespace)
}

func NewContainerdNerdctl() nerdctl.Nerdctl {
	addr, namespace := GetContainerdConnectionInfo()
	return nerdctl.NewNerdctl(addr, namespace)
}

func PullContainerdImage(input *hostapi.ContainerPullImageInput) error {
	opt := &image.PullOptions{
		RepoCommonOptions: image.RepoCommonOptions{
			SkipVerify: true,
		},
	}
	if input.Auth != nil {
		opt.Username = input.Auth.Username
		opt.Password = input.Auth.Password
	}
	imgTool := NewContainerdImageTool()
	errs := make([]error, 0)
	_, err := imgTool.Pull(input.Image, opt)
	if err != nil {
		// try http protocol
		errs = append(errs, errors.Errorf("pullImageByCtrCmd by https: %s", trimPullImageError(err.Error())))
		opt.PlainHttp = true
		log.Infof("try pull image %s by http", input.Image)
		if _, err := imgTool.Pull(input.Image, opt); err != nil {
			errs = append(errs, errors.Errorf("pullImageByCtrCmd by http: %s", trimPullImageError(err.Error())))
			return errors.NewAggregate(errs)
		}
	}
	return nil
}

func trimPullImageError(err string) string {
	lines := strings.Split(err, "\n")
	filterLines := []string{}
	for _, line := range lines {
		// 过滤掉 containerd 拉取镜像的输出
		if strings.Contains(line, "[0m") {
			continue
		}
		if strings.Contains(line, "elapsed:") && strings.Contains(line, "total:") {
			continue
		}
		filterLines = append(filterLines, line)
	}
	return strings.Join(filterLines, "\n")
}

func PushContainerdImage(input *hostapi.ContainerPushImageInput) error {
	opt := &image.PushOptions{
		RepoCommonOptions: image.RepoCommonOptions{
			SkipVerify: true,
		},
	}
	if input.Auth != nil {
		opt.Username = input.Auth.Username
		opt.Password = input.Auth.Password
	}
	imgTool := NewContainerdImageTool()
	err := imgTool.Push(input.Image, opt)
	errs := make([]error, 0)
	if err != nil {
		// try http protocol
		errs = append(errs, errors.Wrap(err, "pushImageByCtrCmd: %s"))
		opt.PlainHttp = true
		log.Infof("try push image %s by http", input.Image)
		if err := imgTool.Push(input.Image, opt); err != nil {
			errs = append(errs, errors.Wrapf(err, "pushImageByCtrCmd by http"))
			return errors.NewAggregate(errs)
		}
	}
	return nil
}

type ContainerVolumeKey struct {
	Id       string
	HostPath string
}

func (s *sPodGuestInstance) GetVolumeMountUsages() (map[ContainerVolumeKey]*volume_mount.ContainerVolumeMountUsage, error) {
	errs := []error{}
	result := make(map[ContainerVolumeKey]*volume_mount.ContainerVolumeMountUsage)
	for ctrId, vols := range s.getContainerVolumeMounts() {
		for i := range vols {
			vol := vols[i]
			drv, ok := volume_mount.GetDriver(vol.Type).(volume_mount.IUsageVolumeMount)
			if !ok {
				continue
			}
			vu, err := s.getVolumeMountUsage(drv, ctrId, vol)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "get container %s %s volume usage: %s", ctrId, drv.GetType(), jsonutils.Marshal(vol)))
			} else {
				result[ContainerVolumeKey{
					Id:       ctrId,
					HostPath: vu.HostPath,
				}] = vu
			}
		}
	}
	return result, errors.NewAggregate(errs)
}

func (s *sPodGuestInstance) getVolumeMountUsage(drv volume_mount.IUsageVolumeMount, ctrId string, vol *hostapi.ContainerVolumeMount) (*volume_mount.ContainerVolumeMountUsage, error) {
	hp, err := drv.GetRuntimeMountHostPath(s, ctrId, vol)
	if err != nil {
		return nil, errors.Wrap(err, "GetRuntimeMountHostPath")
	}
	us, err := disk.Usage(hp)
	if err != nil {
		return nil, errors.Wrapf(err, "disk.Usage of %s", hp)
	}
	usage := &volume_mount.ContainerVolumeMountUsage{
		Id:         ctrId,
		MountPath:  vol.MountPath,
		HostPath:   hp,
		VolumeType: string(drv.GetType()),
		Usage:      us,
		Tags:       make(map[string]string),
	}
	drv.InjectUsageTags(usage, vol)
	return usage, nil
}

func (s *sPodGuestInstance) RestartLocalPodAndContainers(ctx context.Context, cred mcclient.TokenCredential) {
	s.manager.GuestStartWorker.Run(newLocalPodRestartTask(ctx, cred, s), nil, nil)
}

type localPodRestartTask struct {
	ctx      context.Context
	userCred mcclient.TokenCredential
	pod      *sPodGuestInstance
}

func newLocalPodRestartTask(ctx context.Context, userCred mcclient.TokenCredential, pod *sPodGuestInstance) *localPodRestartTask {
	return &localPodRestartTask{
		ctx:      ctx,
		userCred: userCred,
		pod:      pod,
	}
}

func (t *localPodRestartTask) Run() {
	log.Infof("restart pod and containers locally (%s/%s)", t.pod.Id, t.pod.GetName())
	for _, ctr := range t.pod.GetContainers() {
		log.Infof("stop container locally (%s/%s/%s/%s)", t.pod.Id, t.pod.GetName(), ctr.Id, ctr.Name)
		if _, err := t.pod.StopContainer(t.ctx, t.userCred, ctr.Id, &hostapi.ContainerStopInput{
			Timeout:       0,
			ShmSizeMB:     ctr.Spec.ShmSizeMB,
			ContainerName: ctr.Name,
		}); err != nil {
			log.Errorf("stop container %s error: %v", ctr.Name, err)
		}
	}

	if _, err := t.pod.startPod(t.ctx, t.userCred); err != nil {
		log.Errorf("start pod(%s/%s) err: %s", t.pod.GetId(), t.pod.GetName(), err.Error())
		return
	}
	for _, ctr := range t.pod.GetContainers() {
		log.Infof("start container locally (%s/%s/%s/%s)", t.pod.Id, t.pod.GetName(), ctr.Id, ctr.Name)
		if _, err := t.pod.StartLocalContainer(t.ctx, t.userCred, ctr.Id); err != nil {
			log.Errorf("start container %s err: %s", ctr.Id, err.Error())
		}
	}
	t.pod.SyncStatus("sync status after pod and containers restart locally")
}

func (t *localPodRestartTask) Dump() string {
	return fmt.Sprintf("pod restart task %s/%s", t.pod.GetId(), t.pod.GetName())
}

func GetPodStatusByContainerStatus(status string, cStatus string) string {
	if cStatus == computeapi.CONTAINER_STATUS_CRASH_LOOP_BACK_OFF {
		status = computeapi.POD_STATUS_CRASH_LOOP_BACK_OFF
	}
	if cStatus == computeapi.CONTAINER_STATUS_EXITED && status != computeapi.VM_READY {
		status = computeapi.POD_STATUS_CONTAINER_EXITED
	}
	return status
}
