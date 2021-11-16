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
	"yunion.io/x/onecloud/pkg/multicloud/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ElasticacheClusterListOption struct {
	}
	shellutils.R(&ElasticacheClusterListOption{}, "elastic-cluster-list", "List elasticacheCluster", func(cli *aws.SRegion, args *ElasticacheClusterListOption) error {
		clusters, e := cli.GetElasticacheClusters()
		if e != nil {
			return e
		}
		printList(clusters, len(clusters), 0, len(clusters), []string{})
		return nil
	})

	type ElasticacheReplicaGroupListOption struct {
		Id string
	}
	shellutils.R(&ElasticacheReplicaGroupListOption{}, "elastic-cache-list", "List elasticaReplicaGroup", func(cli *aws.SRegion, args *ElasticacheReplicaGroupListOption) error {
		clusters, e := cli.GetElasticaches(args.Id)
		if e != nil {
			return e
		}
		printList(clusters, len(clusters), 0, len(clusters), []string{})
		return nil
	})

	type ElasticacheSubnetGroupOption struct {
		Id string `help:"subnetgroupId"`
	}
	shellutils.R(&ElasticacheSubnetGroupOption{}, "elastic-cache-subnetgroup-show", "List elasticacheSubnetGroup", func(cli *aws.SRegion, args *ElasticacheSubnetGroupOption) error {
		subnetGroups, e := cli.GetCacheSubnetGroups(args.Id)
		if e != nil {
			return e
		}
		printList(subnetGroups, len(subnetGroups), 0, len(subnetGroups), []string{})
		return nil
	})

	type ElasticacheSnapshotOption struct {
		ReplicaGroupId string `help:"replicaGroupId"`
		SnapshotId     string `help:"SnapshotId"`
	}
	shellutils.R(&ElasticacheSnapshotOption{}, "elastic-cache-subnetgroup-list", "List elasticacheSnapshot", func(cli *aws.SRegion, args *ElasticacheSnapshotOption) error {
		snapshots, e := cli.GetElasticacheSnapshots(args.ReplicaGroupId, args.SnapshotId)
		if e != nil {
			return e
		}
		printList(snapshots, len(snapshots), 0, len(snapshots), []string{})
		return nil
	})

	type ElasticacheParameterOption struct {
		ParameterGroupId string
	}
	shellutils.R(&ElasticacheParameterOption{}, "elastic-cache-parameter-list", "List elasticacheParameter", func(cli *aws.SRegion, args *ElasticacheParameterOption) error {
		parameters, e := cli.GetCacheParameters(args.ParameterGroupId)
		if e != nil {
			return e
		}
		printList(parameters, len(parameters), 0, len(parameters), []string{})
		return nil
	})

	type ElasticacheUserOption struct {
		Engine string `help:"redis"`
	}
	shellutils.R(&ElasticacheUserOption{}, "elastic-cache-user-list", "List elasticacheUser", func(cli *aws.SRegion, args *ElasticacheUserOption) error {
		users, e := cli.GetElasticacheUsers(args.Engine)
		if e != nil {
			return e
		}
		printList(users, len(users), 0, len(users), []string{})
		return nil
	})
}
