package kubeadm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestGetClusterConfigurationFromConfigMap(t *testing.T) {
	testData := `apiVersion: v1
data:
  ClusterConfiguration: |
    apiServer:
      extraArgs:
        default-not-ready-toleration-seconds: "10"
        default-unreachable-toleration-seconds: "10"
      timeoutForControlPlane: 4m0s
    apiVersion: kubeadm.k8s.io/v1beta2
    certificatesDir: /etc/kubernetes/pki
    clusterName: kubernetes
    controlPlaneEndpoint: 10.168.26.182:6443
    controllerManager:
      extraArgs:
        node-monitor-grace-period: 16s
        node-monitor-period: 2s
    dns:
      type: CoreDNS
    etcd:
      local:
        dataDir: /var/lib/etcd
        imageTag: 3.4.6
    imageRepository: registry.cn-beijing.aliyuncs.com/yunionio
    kind: ClusterConfiguration
    kubernetesVersion: v1.15.8
    networking:
      dnsDomain: cluster.local
      podSubnet: 10.40.0.0/16
      serviceSubnet: 10.96.0.0/12
    scheduler: {}
  ClusterStatus: |
    apiEndpoints:
      lzx-oc-node:
        advertiseAddress: 10.168.26.182
        bindPort: 6443
    apiVersion: kubeadm.k8s.io/v1beta2
    kind: ClusterStatus
kind: ConfigMap
metadata:
  creationTimestamp: "2020-03-12T10:26:20Z"
  name: kubeadm-config
  namespace: kube-system
  resourceVersion: "74686202"
  selfLink: /api/v1/namespaces/kube-system/configmaps/kubeadm-config
  uid: 75d3fb7b-2f60-4355-9d01-8d8f9076bf2e`

	d := scheme.Codecs.UniversalDeserializer()
	obj, _, err := d.Decode([]byte(testData), nil, nil)
	if err != nil {
		t.Fatalf("Decode configMap data %s", testData)
	}
	cfgMap := obj.(*v1.ConfigMap)

	assert := assert.New(t)

	config, err := GetClusterConfigurationFromConfigMap(cfgMap)
	assert.Nil(err)
	assert.Equal(config, &ClusterConfiguration{
		ControlPlaneEndpoint: "10.168.26.182:6443",
		KubernetesVersion:    "v1.15.8",
	})
}
