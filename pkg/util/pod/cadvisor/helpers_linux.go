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

package cadvisor

import (
	cadvisorfs "github.com/google/cadvisor/fs"

	"yunion.io/x/pkg/errors"
)

const (
	DockerContainerRuntime = "docker"
	RemoteContainerRuntime = "remote"
)

const (
	// CrioSocket is the path to the CRI-O socket.
	// Please keep this in sync with the one in:
	// github.com/google/cadvisor/container/crio/client.go
	CrioSocket = "/var/run/crio/crio.sock"
)

// imageFsInfoProvider knows how to translate the configured runtime
// to its file system label for images.
type imageFsInfoProvider struct {
	runtime         string
	runtimeEndpoint string
}

func (i *imageFsInfoProvider) ImageFsInfoLabel() (string, error) {
	switch i.runtime {
	case DockerContainerRuntime:
		return cadvisorfs.LabelDockerImages, nil
	case RemoteContainerRuntime:
		// This is a temporary workaround to get stats for cri-o from cadvisor
		// and should be removed.
		// Related to https://github.com/kubernetes/kubernetes/issues/51798
		if i.runtimeEndpoint == CrioSocket || i.runtimeEndpoint == "unix://"+CrioSocket {
			return cadvisorfs.LabelCrioImages, nil
		}
	}
	return "", errors.Errorf("no imagefs label for configured runtime: %s", i.runtime)
}

func NewImageFsInfoProvider(runtime, endpoint string) ImageFsInfoProvider {
	return &imageFsInfoProvider{
		runtime:         runtime,
		runtimeEndpoint: endpoint,
	}
}
