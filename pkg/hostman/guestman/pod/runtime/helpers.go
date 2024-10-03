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
	"fmt"
	"strconv"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

func (m *runtimeManager) sandboxToContainer(s *runtimeapi.PodSandbox) (*Container, error) {
	if s == nil || s.Id == "" {
		return nil, errors.Errorf("unable to convert a nil pointer to a runtime container")
	}

	return &Container{
		ID:    ContainerID{Type: m.runtimeName, ID: s.Id},
		State: SandboxToContainerState(s.State),
	}, nil
}

func SandboxToContainerState(state runtimeapi.PodSandboxState) State {
	switch state {
	case runtimeapi.PodSandboxState_SANDBOX_READY:
		return ContainerStateRunning
	case runtimeapi.PodSandboxState_SANDBOX_NOTREADY:
		return ContainerStateExited
	}
	return ContainerStateUnknown
}

func (m *runtimeManager) toContainer(c *runtimeapi.Container) (*Container, error) {
	if c == nil || c.Id == "" || c.Image == nil {
		return nil, fmt.Errorf("unable to convert a nil pointer to a runtime container")
	}

	return &Container{
		ID:      ContainerID{Type: m.runtimeName, ID: c.Id},
		Name:    c.GetMetadata().GetName(),
		Image:   c.ImageRef,
		ImageID: c.Image.Image,
		State:   toContainerState(c.State),
	}, nil
}

// toContainerState converts runtime.ContainerState to State.
func toContainerState(state runtimeapi.ContainerState) State {
	switch state {
	case runtimeapi.ContainerState_CONTAINER_CREATED:
		return ContainerStateCreated
	case runtimeapi.ContainerState_CONTAINER_RUNNING:
		return ContainerStateRunning
	case runtimeapi.ContainerState_CONTAINER_EXITED:
		return ContainerStateExited
	case runtimeapi.ContainerState_CONTAINER_UNKNOWN:
		return ContainerStateUnknown
	}
	return ContainerStateUnknown
}

type labeledContainerInfo struct {
	ContainerName string
	PodName       string
	PodNamespace  string
	PodUid        string
}

func getStringValueFromLabel(labels map[string]string, label string) string {
	if labels == nil {
		return ""
	}
	if value, found := labels[label]; found {
		return value
	}
	// Do not report error, because there should be many old containers without label now.
	// Return empty string "" for these containers, the caller will get value by other ways.
	return ""
}

func getIntValueFromLabel(labels map[string]string, label string) (int, error) {
	if strValue, found := labels[label]; found {
		intValue, err := strconv.Atoi(strValue)
		if err != nil {
			// This really should not happen. Just set value to 0 to handle this abnormal case
			return 0, err
		}
		return intValue, nil
	}
	// Do not report error, because there should be many old containers without label now.
	log.Infof("Container doesn't have label %s, it may be an old or invalid container", label)
	// Just set the value to 0
	return 0, nil
}

func getContainerInfoFromLabels(labels, annotations map[string]string) *labeledContainerInfo {
	podName := getStringValueFromLabel(labels, PodNameLabel)
	if podName == "" {
		podName = getStringValueFromLabel(annotations, SandboxNameAnnotation)
	}
	podNamespace := getStringValueFromLabel(labels, PodNamespaceLabel)
	if podNamespace == "" {
		podNamespace = getStringValueFromLabel(annotations, SandboxNamespaceAnnotation)
	}
	podUid := getStringValueFromLabel(labels, PodUIDLabel)
	if podUid == "" {
		podUid = getStringValueFromLabel(annotations, SandboxUidAnnotation)
	}
	containerName := getStringValueFromLabel(labels, ContainerNameLabel)
	if containerName == "" {
		containerName = getStringValueFromLabel(annotations, ContainerNameAnnotation)
	}
	return &labeledContainerInfo{
		PodName:       podName,
		PodNamespace:  podNamespace,
		PodUid:        podUid,
		ContainerName: containerName,
	}
}

type annotatedContainerInfo struct {
	Hash                      uint64
	RestartCount              int
	PodDeletionGracePeriod    *int64
	PodTerminationGracePeriod *int64
	TerminationMessagePath    string
}

// getContainerInfoFromAnnotations gets annotatedContainerInfo from annotations.
func getContainerInfoFromAnnotations(annotations map[string]string) *annotatedContainerInfo {
	if annotations == nil {
		return nil
	}
	var err error
	containerInfo := &annotatedContainerInfo{}
	if containerInfo.RestartCount, err = getIntValueFromLabel(annotations, ContainerRestartCountLabel); err != nil {
		log.Errorf("Unable to get %q from annotations %v: %v", ContainerRestartCountLabel, annotations, err)
	}
	return containerInfo
}

type containerStatusByCreated []*Status

func (c containerStatusByCreated) Len() int           { return len(c) }
func (c containerStatusByCreated) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c containerStatusByCreated) Less(i, j int) bool { return c[i].CreatedAt.After(c[j].CreatedAt) }
