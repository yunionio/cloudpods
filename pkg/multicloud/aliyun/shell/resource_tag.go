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
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type TagGetOptions struct {
		SERVICE string `help:"service, eg. ecs" choices:"ecs|kvs|rds|vpc|slb"`
		RESTYPE string `help:"resource type, eg. instance"`
		ID      string `help:"resource Id, eg. ins-123xxx"`
	}
	shellutils.R(&TagGetOptions{}, "tag-list", "List tag of a specific resource", func(cli *aliyun.SRegion, args *TagGetOptions) error {
		tags, err := cli.ListTags(args.SERVICE, args.RESTYPE, args.ID)
		if err != nil {
			return err
		}
		printObject(tags)
		return nil
	})

	type TagOptions struct {
		TagGetOptions
		KEY   string
		VALUE string
	}
	shellutils.R(&TagOptions{}, "tag-resource", "set tags of a specific resource", func(cli *aliyun.SRegion, args *TagOptions) error {
		return cli.TagResource(args.SERVICE, args.RESTYPE, args.ID, map[string]string{args.KEY: args.VALUE})
	})

	type UnTagOptions struct {
		TagGetOptions
		KEY []string
	}

	shellutils.R(&UnTagOptions{}, "untag-resource", "un tags of a specific resource", func(cli *aliyun.SRegion, args *UnTagOptions) error {
		return cli.UntagResource(args.SERVICE, args.RESTYPE, args.ID, args.KEY)
	})

	type TagSetOptions struct {
		TagGetOptions
		VALUES  []string
		Replace bool
	}

	shellutils.R(&TagSetOptions{}, "tag-set", "set tags of a specific resource", func(cli *aliyun.SRegion, args *TagSetOptions) error {
		tags := map[string]string{}
		for _, value := range args.VALUES {
			v := strings.Split(value, ":")
			if len(v) != 2 {
				return errors.Errorf("invalid tag %s", value)
			}
			tags[v[0]] = v[1]
		}
		return cli.SetResourceTags(args.SERVICE, args.RESTYPE, args.ID, tags, args.Replace)
	})

}
