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
	"context"
	"fmt"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SCluster struct {
	SManagedObject
}

func NewCluster(manager *SESXiClient, cluster *mo.ClusterComputeResource, dc *SDatacenter) *SCluster {
	return &SCluster{SManagedObject: newManagedObject(manager, cluster, dc)}
}

func (cluster *SCluster) ListResourcePools() ([]mo.ResourcePool, error) {
	var pools, result []mo.ResourcePool
	err := cluster.manager.scanMObjects(cluster.object.Entity().Self, RESOURCEPOOL_PROPS, &pools)
	if err != nil {
		return nil, errors.Wrap(err, "scanMObjects")
	}
	for i := range pools {
		if pools[i].Parent.Type == "ClusterComputeResource" {
			continue
		}
		result = append(result, pools[i])
	}
	return result, nil
}

func (cluster *SCluster) getDefaultResourcePool() (mo.ResourcePool, error) {
	pools := []mo.ResourcePool{}
	err := cluster.manager.scanMObjects(cluster.object.Entity().Self, RESOURCEPOOL_PROPS, &pools)
	if err != nil {
		return mo.ResourcePool{}, errors.Wrap(err, "scanMObjects")
	}
	for i := range pools {
		if pools[i].Parent.Type == "ClusterComputeResource" {
			return pools[i], nil
		}
	}
	return mo.ResourcePool{}, cloudprovider.ErrNotFound
}

func (cluster *SCluster) CreateResourcePool(name string) (*mo.ResourcePool, error) {
	if len(name) == 0 {
		return nil, errors.Error("empty name str")
	}
	root, err := cluster.getDefaultResourcePool()
	if err != nil {
		return nil, errors.Wrap(err, "getDefaultResourcePool")
	}
	pool := object.NewResourcePool(cluster.manager.client.Client, root.Reference())
	pool.InventoryPath = fmt.Sprintf("/%s/host/%s/Resources", cluster.datacenter.GetName(), cluster.GetName())
	_, err = pool.Create(context.Background(), name, types.DefaultResourceConfigSpec())
	if err != nil {
		return nil, errors.Wrap(err, "pool.Create")
	}

	pools, err := cluster.ListResourcePools()
	if err != nil {
		return nil, errors.Wrap(err, "listResourcePools")
	}
	for i := range pools {
		p := NewResourcePool(cluster.manager, &pools[i], cluster.datacenter)
		if p.GetName() == name {
			return &pools[i], nil
		}
	}
	return nil, errors.Wrap(cloudprovider.ErrNotFound, "AfterCreate")
}

func (cluster *SCluster) SyncResourcePool(name string) (*mo.ResourcePool, error) {
	pools, err := cluster.ListResourcePools()
	if err != nil {
		return nil, errors.Wrap(err, "ListResourcePools")
	}
	for i := range pools {
		if pools[i].Entity().Name == name {
			return &pools[i], nil
		}
	}
	return cluster.CreateResourcePool(name)
}

func (cluster *SCluster) getoCluster() *mo.ClusterComputeResource {
	return cluster.object.(*mo.ClusterComputeResource)
}
