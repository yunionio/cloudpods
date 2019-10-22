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
	"context"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/mcclient/auth"
	kubeserver "yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

type SKubeClusterManager struct {
	k8sConfigLock *sync.RWMutex
	k8sConfig     string
	interval      time.Duration
	region        string
}

func NewKubeClusterManager(region string, interval time.Duration) *SKubeClusterManager {
	return &SKubeClusterManager{
		k8sConfigLock: new(sync.RWMutex),
		interval:      interval,
		region:        region,
	}
}

func (man *SKubeClusterManager) GetK8sConfig() string {
	man.k8sConfigLock.RLock()
	defer man.k8sConfigLock.RUnlock()
	return man.k8sConfig
}

func (man *SKubeClusterManager) GetK8sClient() (*kubernetes.Clientset, error) {
	cli, err := NewClientByContent([]byte(man.GetK8sConfig()), nil)
	if err != nil {
		log.Warningf("Init kubernetes client error: %v", err)
		return nil, err
	}
	return cli, nil
}

func (man *SKubeClusterManager) Start() {
	go man.startRefreshKubeConfig()
}

func (man *SKubeClusterManager) setK8sConfig(conf string) {
	man.k8sConfigLock.Lock()
	defer man.k8sConfigLock.Unlock()
	man.k8sConfig = conf
}

func (man *SKubeClusterManager) isK8sHealthy() bool {
	if man.GetK8sConfig() == "" {
		return false
	}
	cli, err := man.GetK8sClient()
	if err != nil {
		return false
	}
	_, err = cli.Discovery().ServerVersion()
	if err != nil {
		log.Errorf("Discovery k8s version: %v", err)
		return false
	}
	return true
}

func (man *SKubeClusterManager) startRefreshKubeConfig() {
	man.refreshKubeConfig()
	tick := time.Tick(man.interval)
	for {
		select {
		case <-tick:
			man.refreshKubeConfig()
		}
	}
}

func (man *SKubeClusterManager) refreshKubeConfig() {
	if man.isK8sHealthy() {
		return
	}
	kubeConfig, err := man.getKubeClusterConfig()
	if err != nil {
		log.Errorf("Get default k8s config from kube server error: %v", err)
	}
	man.setK8sConfig(kubeConfig)
}

func (man *SKubeClusterManager) getKubeClusterConfig() (string, error) {
	session := auth.GetAdminSession(context.Background(), man.region, "v1")
	params := jsonutils.NewDict()
	params.Add(jsonutils.JSONTrue, "directly")
	ret, err := kubeserver.KubeClusters.PerformAction(session, "default", "generate-kubeconfig", params)
	if err != nil {
		return "", err
	}
	return ret.GetString("kubeconfig")
}
