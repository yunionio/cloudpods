/*
 * @Author: your name
 * @Date: 2022-02-17 21:54:45
 * @LastEditTime: 2022-02-18 16:15:09
 * @LastEditors: Please set LastEditors
 * @Description: 打开koroFileHeader查看配置 进行设置: https://github.com/OBKoro1/koro1FileHeader/wiki/%E9%85%8D%E7%BD%AE
 * @FilePath: \cloudpods\pkg\multicloud\bingocloud\shell\cluster.go
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

package shell

import (
	"yunion.io/x/onecloud/pkg/multicloud/bingocloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ClusterListOptions struct {
		ClusterId string
	}
	shellutils.R(&ClusterListOptions{}, "cluster-list", "list clusters", func(cli *bingocloud.SRegion, args *ClusterListOptions) error {
		clusters, err := cli.GetClusters()
		if err != nil {
			return err
		}
		printList(clusters, 0, 0, 0, []string{})
		return nil
	})

	type ClusterIdOptions struct {
		ID string
	}

	// shellutils.R(&ClusterIdOptions{}, "cluster-show", "show clusters", func(cli *bingocloud.SRegion, args *ClusterIdOptions) error {
	// 	cluster, err := cli.GetCluster(args.ID)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	printObject(cluster)
	// 	return nil
	// })

}
