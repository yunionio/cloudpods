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

package k8s

import (
	"fmt"
	"net/http"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/transport"
)

const (
	K8sWrapTransportTimeout = 30
)

type WrapTransport func(rt http.RoundTripper) http.RoundTripper

func NewClientByFile(kubeConfigPath string, k8sWrapTransport WrapTransport) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(setConfigField(config, k8sWrapTransport))
}

func GetK8sClientConfig(kubeConfig []byte) (*rest.Config, error) {
	var config *rest.Config
	var err error
	if kubeConfig != nil {
		apiconfig, err := clientcmd.Load(kubeConfig)
		if err != nil {
			return nil, err
		}

		clientConfig := clientcmd.NewDefaultClientConfig(*apiconfig, &clientcmd.ConfigOverrides{})
		config, err = clientConfig.ClientConfig()
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("kubeconfig value is nil")
	}
	if err != nil {
		return nil, fmt.Errorf("create kubernetes config failed: %v", err)
	}
	return config, nil
}

func NewClientByContent(kubeConfig []byte, k8sWrapTransport WrapTransport) (*kubernetes.Clientset, error) {
	config, err := GetK8sClientConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("Create kubernetes config: %v", err)
	}
	return kubernetes.NewForConfig(setConfigField(config, k8sWrapTransport))
}

func setConfigField(c *rest.Config, tr WrapTransport) *rest.Config {
	if tr != nil {
		c.WrapTransport = transport.WrapperFunc(tr)
	}
	c.Timeout = time.Second * time.Duration(K8sWrapTransportTimeout)
	return c
}
