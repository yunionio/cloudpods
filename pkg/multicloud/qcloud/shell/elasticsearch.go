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
	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ElasticSearchListOptions struct {
		Ids    []string
		Offset int
		Limit  int
	}
	shellutils.R(&ElasticSearchListOptions{}, "elastic-search-list", "List Elasticsearch", func(cli *qcloud.SRegion, args *ElasticSearchListOptions) error {
		es, _, err := cli.GetElasticSearchs(args.Ids, args.Limit, args.Offset)
		if err != nil {
			return err
		}
		printList(es, 0, 0, 0, []string{})
		return nil
	})

	type ElasticSearchIdOptions struct {
		ID string
	}

	shellutils.R(&ElasticSearchIdOptions{}, "elastic-search-delete", "Delete Elasticsearch", func(cli *qcloud.SRegion, args *ElasticSearchIdOptions) error {
		return cli.DeleteElasticSearch(args.ID)
	})

}
