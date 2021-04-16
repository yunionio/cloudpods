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

	"yunion.io/x/onecloud/pkg/util/rbdutils"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ClusterStatOption struct {
	}

	shellutils.R(&ClusterStatOption{}, "cluster-stat", "Show Cluster State", func(cli *rbdutils.SPool, args *ClusterStatOption) error {
		cluster := cli.GetCluster()
		stat, err := cluster.GetClusterStats()
		if err != nil {
			return errors.Wrapf(err, "GetClusterStats")
		}
		printObject(stat)
		return nil
	})

	type CmdOptions struct {
		CMD string
	}

	shellutils.R(&CmdOptions{}, "run", "Run Cluster command", func(cli *rbdutils.SPool, args *CmdOptions) error {
		ret, err := cli.GetCluster().MonCommand([]byte(args.CMD))
		if err != nil {
			return errors.Wrapf(err, "MonCommand")
		}
		printObject(ret)
		return nil
	})

}
