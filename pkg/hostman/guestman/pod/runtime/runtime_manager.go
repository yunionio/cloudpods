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

package runtime

import (
	"context"
	"encoding/json"
	"net"
	"sort"
	"time"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/util/pod"
)

const (
	runtimeAPIVersion = "0.1.0"
)

type runtimeManager struct {
	runtimeName string
	// gRPC service clients.
	cri pod.CRI
}

func NewRuntimeManager(cri pod.CRI) (Runtime, error) {
	man := &runtimeManager{
		cri: cri,
	}

	typedVersion, err := man.getTypedVersion()
	if err != nil {
		return nil, errors.Wrap(err, "getTypedVersion")
	}
	man.runtimeName = typedVersion.RuntimeName
	log.Infof("Container runtime %s initialized, version: %s, apiVersion: %s", typedVersion.RuntimeName, typedVersion.RuntimeVersion, typedVersion.RuntimeApiVersion)

	return man, nil
}

func (m *runtimeManager) getTypedVersion() (*runtimeapi.VersionResponse, error) {
	resp, err := m.cri.GetRuntimeClient().Version(context.Background(), &runtimeapi.VersionRequest{Version: runtimeAPIVersion})
	if err != nil {
		return nil, errors.Wrap(err, "get remote runtime typed version")
	}
	return resp, nil
}

// Type returns the type of the container runtime.
func (m *runtimeManager) Type() string {
	return m.runtimeName
}

// getSandboxes lists all (or just the running) sandboxes.
func (m *runtimeManager) getSandboxes(all bool) ([]*runtimeapi.PodSandbox, error) {
	var filter *runtimeapi.PodSandboxFilter
	if !all {
		readyState := runtimeapi.PodSandboxState_SANDBOX_READY
		filter = &runtimeapi.PodSandboxFilter{
			State: &runtimeapi.PodSandboxStateValue{
				State: readyState,
			},
		}
	}

	resp, err := m.cri.GetRuntimeClient().ListPodSandbox(context.Background(), &runtimeapi.ListPodSandboxRequest{
		Filter: filter,
	})
	if err != nil {
		log.Errorf("ListPodSandbox failed: %v", err)
		return nil, err
	}

	return resp.Items, nil
}

type ContainerRuntimeSpec struct {
	Annotations map[string]string `json:"annotations"`
}

type ContainerExtraInfo struct {
	SandboxID   string               `json:"sandbox_id"`
	Pid         int                  `json:"pid"`
	RuntimeSpec ContainerRuntimeSpec `json:"runtimeSpec"`
}

// GetPods returns a list of containers grouped by pods. The boolean parameter
// specifies whether the runtime returns all containers including those already
// exited and dead containers (used for garbage collection).
func (m *runtimeManager) GetPods(all bool) ([]*Pod, error) {
	pods := make(map[string]*Pod)
	sandboxes, err := m.getSandboxes(all)
	if err != nil {
		return nil, err
	}
	for i := range sandboxes {
		s := sandboxes[i]
		if s.Metadata == nil {
			log.Infof("Sandbox does not have metadata: %#v", s)
			continue
		}
		podUid := s.Metadata.Uid
		if _, ok := pods[podUid]; !ok {
			pods[podUid] = &Pod{
				Id:        podUid,
				CRIId:     s.Id,
				Name:      s.Metadata.Name,
				Namespace: s.Metadata.Namespace,
			}
		}
		p := pods[podUid]
		converted, err := m.sandboxToContainer(s)
		if err != nil {
			log.Infof("Convert %q sandbox %v of pod %q failed: %v", m.runtimeName, s, podUid, err)
			continue
		}
		p.Sandboxes = append(p.Sandboxes, converted)
	}

	containers, err := m.getContainers(all)
	if err != nil {
		return nil, err
	}
	for i := range containers {
		c := containers[i]
		if c.Metadata == nil {
			log.Infof("Container does not have metadata: %+v", c)
			continue
		}

		labelledInfo := getContainerInfoFromLabels(c.Labels, c.Annotations)
		if labelledInfo.PodUid == "" {
			// 旧的容器没设置 labels 标签，需要从 status.info.runtimeSpec.annotations 里面找 pod 关联信息
			resp, err := m.cri.GetRuntimeClient().ContainerStatus(context.Background(), &runtimeapi.ContainerStatusRequest{
				ContainerId: c.Id,
				Verbose:     true,
			})
			if err != nil {
				log.Infof("get container %s status failed: %v", c.GetId(), err)
				continue
			}
			infoStr, ok := resp.GetInfo()["info"]
			if !ok {
				log.Infof("not found container %s info", c.GetId())
				continue
			}
			info := new(ContainerExtraInfo)
			if err := json.Unmarshal([]byte(infoStr), info); err != nil {
				log.Infof("unmarshal container %s info failed: %v", c.GetId(), err)
				continue
			}
			labelledInfo = getContainerInfoFromLabels(nil, info.RuntimeSpec.Annotations)
		}
		pod, found := pods[labelledInfo.PodUid]
		if !found {
			pod = &Pod{
				Id:        labelledInfo.PodUid,
				Name:      labelledInfo.PodName,
				Namespace: labelledInfo.PodNamespace,
			}
			pods[labelledInfo.PodUid] = pod
		}

		converted, err := m.toContainer(c)
		if err != nil {
			log.Warningf("Convert %s container %v of pod %q failed: %v", m.runtimeName, c, labelledInfo.PodUid, err)
			continue
		}
		pod.Containers = append(pod.Containers, converted)
	}

	// convert map to list.
	var result []*Pod
	for i := range pods {
		result = append(result, pods[i])
	}
	return result, nil
}

func (m *runtimeManager) getContainers(allContainers bool) ([]*runtimeapi.Container, error) {
	filter := &runtimeapi.ContainerFilter{}
	if !allContainers {
		filter.State = &runtimeapi.ContainerStateValue{
			State: runtimeapi.ContainerState_CONTAINER_RUNNING,
		}
	}

	containers, err := m.cri.GetRuntimeClient().ListContainers(context.Background(), &runtimeapi.ListContainersRequest{Filter: filter})
	if err != nil {
		return nil, errors.Wrap(err, "ListContainers failed")
	}
	return containers.Containers, nil
}

func (m *runtimeManager) GetPodStatus(uid, name, namespace string) (*PodStatus, error) {
	// Now we retain restart count of container as a container label. Each time a container
	// restarts, pod will read the restart count from the registered dead container, increment
	// it to get the new restart count, and then add a label with the new restart count on
	// the newly started container.
	// However, there are some limitations of this method:
	//	1. When all dead containers were garbage collected, the container status could
	//	not get the historical value and would be *inaccurate*. Fortunately, the chance
	//	is really slim.
	//	2. When working with old version containers which have no restart count label,
	//	we can only assume their restart count is 0.
	// Anyhow, we only promised "best-effort" restart count reporting, we can just ignore
	// these limitations now.
	podSandboxIDs, err := m.getSandboxIDByPodUID(uid, nil)
	if err != nil {
		return nil, err
	}

	podFullName := BuildPodFullName(name, namespace)

	log.Debugf("getSandboxIDByPodUID got sandbox IDs %q for pod %q", podSandboxIDs, podFullName)

	sandboxStatuses := make([]*runtimeapi.PodSandboxStatus, len(podSandboxIDs))
	podIPs := []string{}
	for idx, podSandboxID := range podSandboxIDs {
		req := &runtimeapi.PodSandboxStatusRequest{
			PodSandboxId: podSandboxID,
		}
		resp, err := m.cri.GetRuntimeClient().PodSandboxStatus(context.Background(), req)
		if err != nil {
			log.Errorf("PodSandboxStatus of sandbox %q for pod %q error: %v", podSandboxID, podFullName, err)
			return nil, err
		}
		podSandboxStatus := resp.Status
		sandboxStatuses[idx] = podSandboxStatus

		// Only get pod IP from latest sandbox
		if idx == 0 && podSandboxStatus.State == runtimeapi.PodSandboxState_SANDBOX_READY {
			podIPs = m.determinePodSandboxIPs(podSandboxStatus)
		}
	}

	// Get statuses of all containers visible in the pod.
	containerStatuses, err := m.getPodContainerStatuses(uid, podSandboxIDs)
	if err != nil {
		log.Errorf("getPodContainerStatuses for pod %q failed: %v", podFullName, err)
		return nil, err
	}

	return &PodStatus{
		ID:                uid,
		Name:              name,
		Namespace:         namespace,
		IPs:               podIPs,
		SandboxStatuses:   sandboxStatuses,
		ContainerStatuses: containerStatuses,
	}, nil
}

func GetSandboxIDByPodUID(cri pod.CRI, podUID string, state *runtimeapi.PodSandboxState) ([]string, error) {
	filter := &runtimeapi.PodSandboxFilter{
		LabelSelector: map[string]string{
			PodUIDLabel: podUID,
		},
	}
	if state != nil {
		filter.State = &runtimeapi.PodSandboxStateValue{
			State: *state,
		}
	}
	resp, err := cri.GetRuntimeClient().ListPodSandbox(context.Background(), &runtimeapi.ListPodSandboxRequest{Filter: filter})
	if err != nil {
		return nil, errors.Wrap(err, "ListPodSandbox failed")
	}
	sandboxes := resp.Items
	if len(sandboxes) == 0 {
		// 兼容旧版没有打标签的 pods
		pods, err := cri.ListPods(context.Background(), pod.ListPodOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "List all pods failed")
		}
		for i := range pods {
			item := pods[i]
			if item.Metadata.Uid == podUID {
				sandboxes = append(sandboxes, item)
			}
		}
	}

	// Sort with newest first.
	sandboxIDs := make([]string, len(sandboxes))
	for i, s := range sandboxes {
		sandboxIDs[i] = s.Id
	}

	return sandboxIDs, nil
}

func (m *runtimeManager) getSandboxIDByPodUID(podUID string, state *runtimeapi.PodSandboxState) ([]string, error) {
	return GetSandboxIDByPodUID(m.cri, podUID, state)
}

// determinePodSandboxIP determines the IP addresses of the given pod sandbox.
func (m *runtimeManager) determinePodSandboxIPs(podSandbox *runtimeapi.PodSandboxStatus) []string {
	podIPs := make([]string, 0)
	if podSandbox.Network == nil {
		log.Warningf("Pod Sandbox status doesn't have network information, cannot report IPs")
		return podIPs
	}

	// ip could be an empty string if runtime is not responsible for the
	// IP (e.g., host networking).

	// pick primary IP
	if len(podSandbox.Network.Ip) != 0 {
		if net.ParseIP(podSandbox.Network.Ip) == nil {
			log.Warningf("Pod Sandbox reported an unparseable IP (Primary) %v", podSandbox.Network.Ip)
			return nil
		}
		podIPs = append(podIPs, podSandbox.Network.Ip)
	}

	// pick additional ips, if cri reported them
	for _, podIP := range podSandbox.Network.AdditionalIps {
		if nil == net.ParseIP(podIP.Ip) {
			log.Warningf("Pod Sandbox reported an unparseable IP (additional) %v", podIP.Ip)
			return nil
		}
		podIPs = append(podIPs, podIP.Ip)
	}

	return podIPs
}

func (m *runtimeManager) getPodContainerStatuses(uid string, criId []string) ([]*Status, error) {
	resp, err := m.cri.GetRuntimeClient().ListContainers(context.Background(), &runtimeapi.ListContainersRequest{Filter: &runtimeapi.ContainerFilter{
		LabelSelector: map[string]string{PodUIDLabel: uid},
	}})
	if err != nil {
		return nil, errors.Wrap(err, "ListContainers with label selector failed")
	}
	containers := resp.Containers
	if len(containers) == 0 {
		// 兼容旧版没有打标签的容器
		allContainers, err := m.cri.ListContainers(context.Background(), pod.ListContainerOptions{})
		if err != nil {
			return nil, errors.Wrapf(err, "ListContainers by pod uid: %s", uid)
		}
		for i := range allContainers {
			container := allContainers[i]
			if utils.IsInStringArray(container.PodSandboxId, criId) {
				containers = append(containers, container)
			}
		}
	}

	statuses := make([]*Status, len(containers))
	for i, c := range containers {
		sResp, err := m.cri.ContainerStatus(context.Background(), c.Id)
		if err != nil {
			return nil, errors.Wrapf(err, "ContainerStatus by container id: %s", c.Id)
		}
		status := sResp.Status
		cStatus := ToContainerStatus(status, m.runtimeName)
		cStatus.PodSandboxID = c.PodSandboxId
		statuses[i] = cStatus
	}

	sort.Sort(containerStatusByCreated(statuses))
	return statuses, nil
}

func ToContainerStatus(status *runtimeapi.ContainerStatus, runtimeName string) *Status {
	annotatedInfo := getContainerInfoFromAnnotations(status.Annotations)
	labeledInfo := getContainerInfoFromLabels(status.Labels, status.Annotations)
	cStatus := &Status{
		ID: ContainerID{
			Type: runtimeName,
			ID:   status.Id,
		},
		Name:      labeledInfo.ContainerName,
		Image:     status.Image.Image,
		ImageID:   status.ImageRef,
		State:     toContainerState(status.State),
		CreatedAt: time.Unix(0, status.CreatedAt),
	}
	if annotatedInfo != nil {
		// cStatus.Hash = annotatedInfo.Hash
		cStatus.RestartCount = annotatedInfo.RestartCount
	}

	if status.State != runtimeapi.ContainerState_CONTAINER_CREATED {
		// If container is not in the created state, we have tried and
		// started the container. Set the StartedAt time.
		cStatus.StartedAt = time.Unix(0, status.StartedAt)
	}
	if status.State == runtimeapi.ContainerState_CONTAINER_EXITED {
		cStatus.Reason = status.Reason
		cStatus.Message = status.Message
		cStatus.ExitCode = int(status.ExitCode)
		cStatus.FinishedAt = time.Unix(0, status.FinishedAt)
	}
	return cStatus
}
