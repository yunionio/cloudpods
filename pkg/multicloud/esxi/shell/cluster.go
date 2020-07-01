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

package shell

import (
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/multicloud/esxi"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ClusterListOptions struct {
		DATACENTER string `help:"List clusters in datacenter"`
	}
	shellutils.R(&ClusterListOptions{}, "cluster-list", "List all clusters", func(cli *esxi.SESXiClient, args *ClusterListOptions) error {
		dc, err := cli.FindDatacenterByMoId(args.DATACENTER)
		if err != nil {
			return err
		}
		clusters, err := dc.ListClusters()
		if err != nil {
			return err
		}
		printList(clusters, nil)
		return nil
	})

	type ClusterPoolListOptions struct {
		DATACENTER string `help:"List clusters in datacenter"`
		CLUSTER    string `help:"List cluster resource pool"`
	}

	shellutils.R(&ClusterPoolListOptions{}, "cluster-pool-list", "List all cluster resource pool", func(cli *esxi.SESXiClient, args *ClusterPoolListOptions) error {
		dc, err := cli.FindDatacenterByMoId(args.DATACENTER)
		if err != nil {
			return err
		}
		cluster, err := dc.GetCluster(args.CLUSTER)
		if err != nil {
			return err
		}
		pools, err := cluster.ListResourcePools()
		if err != nil {
			return errors.Wrap(err, "ListResourcePools")
		}
		printList(pools, nil)
		return nil
	})

	type ClusterPoolSyncOptions struct {
		DATACENTER string `help:"List clusters in datacenter"`
		CLUSTER    string `help:"List cluster resource pool"`
		GroupId    string `help:"Resource pool Id"`
		Name       string `help:"Resource pool name"`
	}

	shellutils.R(&ClusterPoolSyncOptions{}, "cluster-pool-sync", "Sync cluster resource pool", func(cli *esxi.SESXiClient, args *ClusterPoolSyncOptions) error {
		dc, err := cli.FindDatacenterByMoId(args.DATACENTER)
		if err != nil {
			return err
		}
		cluster, err := dc.GetCluster(args.CLUSTER)
		if err != nil {
			return err
		}
		pool, err := cluster.SyncResourcePool(args.GroupId, args.Name)
		if err != nil {
			return errors.Wrap(err, "SyncResourcePool")
		}
		printObject(pool)
		return nil
	})

}
