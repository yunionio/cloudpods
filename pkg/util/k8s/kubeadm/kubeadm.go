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
