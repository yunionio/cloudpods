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

package kubelet

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

const containerdRootDir = "/var/lib/containerd"

type RuntimeInfo struct {
	RootDir string
}

type dockerInfo struct {
	ID            string `json:"ID"`
	Driver        string `json:"Driver"`
	DockerRootDir string `json:"DockerRootDir"`
}

func getDockerRuntimeInfoByRemote() (*RuntimeInfo, error) {
	content, err := procutils.NewRemoteCommandAsFarAsPossible("docker", "info", "--format", "{{json .}}").Output()
	if err != nil {
		return nil, errors.Wrap(err, "Run command 'docker info'")
	}
	return parseDockerRuntimeInfo(content)
}

func parseDockerRuntimeInfo(content []byte) (*RuntimeInfo, error) {
	obj, err := jsonutils.Parse(content)
	if err != nil {
		return nil, errors.Wrap(err, "Parse docker info to json")
	}

	info := new(dockerInfo)
	if err := obj.Unmarshal(info); err != nil {
		return nil, errors.Wrap(err, "Unmarshal docker info")
	}
	if len(info.DockerRootDir) == 0 {
		return nil, errors.Error("docker info returned an empty root directory")
	}

	return &RuntimeInfo{RootDir: info.DockerRootDir}, nil
}

func getContainerdRuntimeInfoByRemote() (*RuntimeInfo, error) {
	if err := procutils.NewRemoteCommandAsFarAsPossible("test", "-d", containerdRootDir).Run(); err != nil {
		return nil, errors.Wrap(err, "Find containerd root directory")
	}
	return &RuntimeInfo{RootDir: containerdRootDir}, nil
}

func GetContainerRuntimeInfoByRemote() (*RuntimeInfo, error) {
	dockerInfo, dockerErr := getDockerRuntimeInfoByRemote()
	if dockerErr == nil {
		return dockerInfo, nil
	}
	containerdInfo, containerdErr := getContainerdRuntimeInfoByRemote()
	if containerdErr == nil {
		return containerdInfo, nil
	}
	return nil, errors.Wrapf(containerdErr, "Get container runtime info (docker: %v)", dockerErr)
}
