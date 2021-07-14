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

package kubeadm

import (
	"fmt"

	v1 "k8s.io/api/core/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

func GetClusterConfigurationFromConfigMap(cfgMap *v1.ConfigMap) (*ClusterConfiguration, error) {
	data := cfgMap.Data
	key := ClusterConfigurationConfigMapKey

	clusterConfigData, ok := data[key]
	if !ok {
		return nil, errors.Error(fmt.Sprintf("%s key value pair missing", key))
	}

	return getClusterConfigurationFromConfigMapData(clusterConfigData)
}

func getClusterConfigurationFromConfigMapData(configStr string) (*ClusterConfiguration, error) {
	configObj, err := jsonutils.ParseYAML(configStr)
	if err != nil {
		return nil, errors.Wrapf(err, "parse ClusterConfiguration content %q", configStr)
	}

	config := new(ClusterConfiguration)
	if err := configObj.Unmarshal(config); err != nil {
		return nil, errors.Wrapf(err, "unmarshal cluster config %s", configObj)
	}

	return config, nil
}
