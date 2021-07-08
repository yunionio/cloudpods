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

type DockerInfo struct {
	ID            string `json:"ID"`
	Driver        string `json:"Driver"`
	DockerRootDir string `json:"DockerRootDir"`
}

func GetDockerInfoByRemote() (*DockerInfo, error) {
	content, err := procutils.NewRemoteCommandAsFarAsPossible("docker", "info", "--format", "{{json .}}").Output()
	if err != nil {
		return nil, errors.Wrap(err, "Run command 'docker info'")
	}

	obj, err := jsonutils.Parse(content)
	if err != nil {
		return nil, errors.Wrap(err, "Parse docker info to json")
	}

	info := new(DockerInfo)
	if err := obj.Unmarshal(info); err != nil {
		return nil, errors.Wrap(err, "Unmarshal docker info")
	}

	return info, nil
}
