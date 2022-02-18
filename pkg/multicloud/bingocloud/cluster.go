/*
 * @Author: your name
 * @Date: 2022-02-17 21:54:45
 * @LastEditTime: 2022-02-18 16:15:02
 * @LastEditors: Please set LastEditors
 * @Description: 打开koroFileHeader查看配置 进行设置: https://github.com/OBKoro1/koro1FileHeader/wiki/%E9%85%8D%E7%BD%AE
 * @FilePath: \cloudpods\pkg\multicloud\bingocloud\vpc.go
 */
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
package bingocloud

import (
	"yunion.io/x/log"
)

type SCluster struct {
	Hypervisor           string `json:"hypervisor"`
	MaxVolumeStorage     string `json:"maxVolumeStorage"`
	ExtendDiskMode       string `json:"extendDiskMode"`
	CreateVolumeMode     string `json:"createVolumeMode"`
	ClusterControllerSet struct {
		Item struct {
			Role    string `json:"role"`
			Address string `json:"address"`
		}
	}
	ClusterId   string `json:"clusterId"`
	ClusterName string `json:"clusterName"`
	Status      string `json:"status"`
	SchedPolicy string `json:"schedPolicy"`
}

func (self *SRegion) GetClusters() ([]SCluster, error) {
	resp, err := self.invoke("DescribeClusters", nil)
	if err != nil {
		return nil, err
	}
	log.Errorf("resp=:%s", resp)
	result := struct {
		ClusterSet struct {
			Item []SCluster
		}
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, err
	}
	return result.ClusterSet.Item, nil
}

// func (self *SRegion) GetCluster(id string) (*SCluster, error) {
// 	clusters := &SCluster{}
// 	// return storage, self.get("storage_containers", id, nil, storage)
// 	return clusters, cloudprovider.ErrNotImplemented
// }
