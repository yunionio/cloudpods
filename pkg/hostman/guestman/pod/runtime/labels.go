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

const (
	PodNameLabel               = "io.yunion.pod.name"
	PodNamespaceLabel          = "io.yunion.pod.namespace"
	PodUIDLabel                = "io.yunion.pod.uid"
	ContainerNameLabel         = "io.yunion.container.name"
	ContainerRestartCountLabel = "io.yunion.container.restart_count"
)

const (
	ContainerNameAnnotation    = "io.kubernetes.cri.container-name"
	ContainerTypeAnnotation    = "io.kubernetes.cri.container-type"
	ImageNameAnnotation        = "io.kubernetes.cri.image-name"
	SandboxIdAnnotation        = "io.kubernetes.cri.sandbox-id"
	SandboxNameAnnotation      = "io.kubernetes.cri.sandbox-name"
	SandboxNamespaceAnnotation = "io.kubernetes.cri.sandbox-namespace"
	SandboxUidAnnotation       = "io.kubernetes.cri.sandbox-uid"
)

func GetContainerName(labels map[string]string) string {
	return labels[ContainerNameLabel]
}

func GetPodName(labels map[string]string) string {
	return labels[PodNameLabel]
}

func GetPodUID(labels map[string]string) string {
	return labels[PodUIDLabel]
}

func GetPodNamespace(labels map[string]string) string {
	return labels[PodNamespaceLabel]
}
