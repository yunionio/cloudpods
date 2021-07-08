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
	"path"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/hostutils/kubelet/eviction"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

const (
	KubeletConfigurationFileName = "config.yaml"
)

// KubeletConfig is a interface abstracts manipulation of kubelet run directory.
type KubeletConfig interface {
	HasDedicatedImageFs() bool
	GetNodeFsDevice() string
	GetImageFsDevice() string
	GetImageFs() string
	GetEvictionConfig() eviction.Config
}

// kubeletConfig implements KubeletRunDirectory interface.
type kubeletConfig struct {
	config         jsonutils.JSONObject
	dockerInfo     *DockerInfo
	evictionConfig eviction.Config
	nodeFsDevice   string
	imageFsDevice  string
}

func NewKubeletConfigByDirectory(dir string) (KubeletConfig, error) {
	configFile := path.Join(dir, KubeletConfigurationFileName)

	content, err := procutils.NewRemoteCommandAsFarAsPossible("cat", configFile).Output()
	if err != nil {
		return nil, errors.Wrapf(err, "Read config file %s", configFile)
	}

	dockerInfo, err := GetDockerInfoByRemote()
	if err != nil {
		return nil, errors.Wrap(err, "Get docker info")
	}

	return newKubeletConfig(content, dockerInfo)
}

func newKubeletConfig(yamlConfig []byte, dockerInfo *DockerInfo) (KubeletConfig, error) {
	obj, err := jsonutils.ParseYAML(string(yamlConfig))
	if err != nil {
		return nil, errors.Wrapf(err, "Parse yaml content %s", yamlConfig)
	}

	evictionConfig, err := eviction.NewConfig(yamlConfig)
	if err != nil {
		return nil, errors.Wrap(err, "New eviction config")
	}

	imageFsDev, err := GetDirectoryMountDevice(dockerInfo.DockerRootDir)
	if err != nil {
		return nil, errors.Wrap(err, "Find docker root directory device")
	}

	nodeFsDev, err := GetDirectoryMountDevice("/")
	if err != nil {
		return nil, errors.Wrap(err, "Find node FS directory device")
	}

	k := &kubeletConfig{
		config:         obj,
		dockerInfo:     dockerInfo,
		evictionConfig: evictionConfig,
		nodeFsDevice:   nodeFsDev,
		imageFsDevice:  imageFsDev,
	}

	return k, nil
}

func GetDirectoryMountDevice(dirPath string) (string, error) {
	content, err := procutils.NewRemoteCommandAsFarAsPossible("findmnt", "-n", "-o", "SOURCE", "--target", dirPath).Output()
	if err != nil {
		return "", errors.Wrapf(err, "Find directory %q mount source device", dirPath)
	}
	return strings.TrimSpace(string(content)), nil
}

func (k *kubeletConfig) GetNodeFsDevice() string {
	return k.nodeFsDevice
}

func (k *kubeletConfig) GetImageFsDevice() string {
	return k.imageFsDevice
}

func (k *kubeletConfig) HasDedicatedImageFs() bool {
	return k.imageFsDevice != k.nodeFsDevice
}

func (k *kubeletConfig) GetImageFs() string {
	return k.dockerInfo.DockerRootDir
}

func (k *kubeletConfig) GetEvictionConfig() eviction.Config {
	return k.evictionConfig
}
