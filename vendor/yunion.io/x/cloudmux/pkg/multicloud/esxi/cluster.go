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

package esxi

import (
	"strings"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type SCluster struct {
	SManagedObject
}

func NewCluster(manager *SESXiClient, cluster *mo.ClusterComputeResource, dc *SDatacenter) *SCluster {
	return &SCluster{SManagedObject: newManagedObject(manager, cluster, dc)}
}

func (cluster *SCluster) listResourcePools() ([]mo.ResourcePool, error) {
	var pools []mo.ResourcePool
	err := cluster.manager.scanMObjects(cluster.object.Entity().Self, RESOURCEPOOL_PROPS, &pools)
	if err != nil {
		return nil, errors.Wrap(err, "scanMObjects")
	}
	return pools, nil
}

func (cluster *SCluster) SyncResourcePool(name string) (*object.ResourcePool, error) {
	pools, err := cluster.listResourcePools()
	if err != nil {
		return nil, errors.Wrap(err, "listResourcePools")
	}
	for i := range pools {
		pool := NewResourcePool(cluster.manager, &pools[i], cluster.datacenter)
		if strings.EqualFold(pool.GetId(), name) || strings.EqualFold(strings.Join(pool.GetPath(), "/"), name) ||
			strings.EqualFold(strings.Join(pool.GetPath(), "|"), name) ||
			strings.EqualFold(pool.GetName(), name) {
			log.Infof("SyncResourcePool: %s found", strings.Join(pool.GetPath(), "|"))
			return object.NewResourcePool(cluster.manager.client.Client, pools[i].Reference()), nil
		}
	}
	log.Infof("SyncResourcePool: %s not found", name)
	return nil, errors.Wrap(cloudprovider.ErrNotFound, "SyncResourcePool")
}
