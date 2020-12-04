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

package tokens

import (
	"context"
	"encoding/base64"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	bootstrapapi "k8s.io/cluster-bootstrap/token/api"
	bootstraputil "k8s.io/cluster-bootstrap/token/util"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/wait"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/util/k8s/kubeadm"
)

func GetClusterConfig() (*rest.Config, error) {
	// Load kubernetes config inside cluster
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "get kubernetes config inside cluster")
	}
	return cfg, nil
}

func GetClient() (kubernetes.Interface, error) {
	cfg, err := GetClusterConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}

func GetCoreClient() (corev1.CoreV1Interface, error) {
	cli, err := GetClient()
	if err != nil {
		return nil, err
	}
	return cli.CoreV1(), nil
}

func IsInsideKubernetesCluster() (bool, error) {
	_, err := GetCoreClient()
	if err != nil {
		return false, err
	}
	return true, nil
}

func GetControlPlaneEndpoint() (string, error) {
	coreCli, err := GetCoreClient()
	if err != nil {
		return "", errors.Wrap(err, "get cluster control plane endpoint")
	}
	configMap, err := coreCli.ConfigMaps(metav1.NamespaceSystem).Get(context.Background(), kubeadm.KubeadmConfigConfigMap, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "get kubeadm cluster config")
	}
	clusterConfig, err := kubeadm.GetClusterConfigurationFromConfigMap(configMap)
	if err != nil {
		return "", errors.Wrap(err, "get kubeadm cluster configuration")
	}
	return clusterConfig.ControlPlaneEndpoint, nil
}

func GetNodeJoinToken() (string, error) {
	coreCli, err := GetCoreClient()
	if err != nil {
		return "", errors.Wrap(err, "get node join token")
	}

	bootstrapToken, err := NewBootstrap(coreCli, 24*time.Hour)
	if err != nil {
		return "", errors.Wrap(err, "failed to create new bootstrap token")
	}
	return bootstrapToken, nil
}

func GetImageRegistries() ([]string, error) {
	cli, err := GetClient()
	if err != nil {
		return nil, errors.Wrap(err, "get k8s client")
	}
	nodes, err := cli.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "get k8s nodes")
	}
	masterNodes := make([]*v1.Node, 0)
	for _, n := range nodes.Items {
		if _, ok := n.Labels["node-role.kubernetes.io/master"]; ok {
			masterNodes = append(masterNodes, &n)
		}
	}
	if len(masterNodes) == 0 {
		return nil, errors.Wrap(err, "not found master nodes")
	}
	regs := make([]string, 0)
	getImg := func(img v1.ContainerImage) {
		for _, name := range img.Names {
			parts := strings.Split(name, "/")
			if len(parts) == 0 {
				continue
			}
			imgRepo := parts[0]
			if !strings.Contains(imgRepo, ".") {
				// filter image like: grafana, nginx
				continue
			}
			if utils.IsInStringArray(imgRepo, regs) {
				continue
			}
			regs = append(regs, imgRepo)
		}
	}
	for _, n := range masterNodes {
		for _, img := range n.Status.Images {
			getImg(img)
		}
	}
	return regs, nil
}

// TODO: move to other packages
type DockerDaemonConfig struct {
	Bridge             string            `json:"bridge"`
	Iptables           bool              `json:"iptables"`
	ExecOpts           []string          `json:"exec-opts"`
	DataRoot           string            `json:"data-root"`
	LogDriver          string            `json:"log-driver"`
	LogOpts            map[string]string `json:"log-opts"`
	RegistryMirrors    []string          `json:"registry-mirrors"`
	InsecureRegistries []string          `json:"insecure-registries"`
	LiveRestore        bool              `json:"live-restore"`
}

func GetDockerDaemonConfig() (*DockerDaemonConfig, error) {
	regs, err := GetImageRegistries()
	if err != nil {
		return nil, err
	}
	return &DockerDaemonConfig{
		Bridge:    "none",
		Iptables:  false,
		ExecOpts:  []string{"native.cgroupdriver=systemd"},
		DataRoot:  "/opt/docker",
		LogDriver: "json-file",
		LogOpts: map[string]string{
			"max-size": "100m",
		},
		InsecureRegistries: regs,
		LiveRestore:        true,
	}, nil
}

func GetDockerDaemonContent() (string, error) {
	cfg, err := GetDockerDaemonConfig()
	if err != nil {
		return "", nil
	}
	content := jsonutils.Marshal(cfg).PrettyString()
	return base64.StdEncoding.EncodeToString([]byte(content)), nil
}

var (
	MaximumRetries = 5
)

// NewBootstrap attempts to create a token with the given ID.
func NewBootstrap(client corev1.SecretsGetter, ttl time.Duration) (string, error) {
	token, err := bootstraputil.GenerateBootstrapToken()
	if err != nil {
		return "", errors.Wrap(err, "unable to generate bootstrap token")
	}

	substrs := bootstraputil.BootstrapTokenRegexp.FindStringSubmatch(token)
	if len(substrs) != 3 {
		return "", errors.Wrapf(err, "the bootstrap token %q was not of the form %q", token, bootstrapapi.BootstrapTokenPattern)
	}
	tokenID := substrs[1]
	tokenSecret := substrs[2]

	secretName := bootstraputil.BootstrapTokenSecretName(tokenID)
	secretToken := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: metav1.NamespaceSystem,
		},
		Type: bootstrapapi.SecretTypeBootstrapToken,
		Data: map[string][]byte{
			bootstrapapi.BootstrapTokenIDKey:               []byte(tokenID),
			bootstrapapi.BootstrapTokenSecretKey:           []byte(tokenSecret),
			bootstrapapi.BootstrapTokenExpirationKey:       []byte(time.Now().UTC().Add(ttl).Format(time.RFC3339)),
			bootstrapapi.BootstrapTokenUsageSigningKey:     []byte("true"),
			bootstrapapi.BootstrapTokenUsageAuthentication: []byte("true"),
			bootstrapapi.BootstrapTokenExtraGroupsKey:      []byte("system:bootstrappers:kubeadm:default-node-token"),
			bootstrapapi.BootstrapTokenDescriptionKey:      []byte("Node join token generate by 'onecloud region server'"),
		},
	}

	err = TryRunCommand(func() error {
		_, err := client.Secrets(secretToken.ObjectMeta.Namespace).Create(context.Background(), secretToken, metav1.CreateOptions{})
		log.Errorf("create secrets %s/%s error: %v", secretToken.GetNamespace(), secretToken.GetName(), err)
		return err
	}, MaximumRetries)
	if err != nil {
		return "", errors.Wrap(err, "unable to create secret")
	}

	return token, nil
}

// TryRunCommand runs a function a maximum of failureThreshold times, and retries on error. If failureThreshold is hit; the last error is returned
func TryRunCommand(f func() error, failureThreshold int) error {
	backoff := wait.Backoff{
		Duration: 5 * time.Second,
		Factor:   2, // double the timeout for every failure
		Steps:    failureThreshold,
	}
	return wait.ExponentialBackoff(backoff, func() (bool, error) {
		err := f()
		if err != nil {
			// Retry until the timeout
			return false, nil
		}
		// The last f() call was a success, return cleanly
		return true, nil
	})

}
