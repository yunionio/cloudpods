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
	"fmt"
	"strings"

	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type TagGetOptions struct {
		RESTYPE string   `help:"resource type, eg. instance"`
		ID      []string `help:"resource Id, eg. ins-123xxx"`
	}
	shellutils.R(&TagGetOptions{}, "tag-show", "show tag of a specific resource", func(cli *aliyun.SRegion, args *TagGetOptions) error {
		tags, err := cli.ListResourceTags(args.RESTYPE, args.ID)
		if err != nil {
			return err
		}
		for id, tag := range tags {
			fmt.Println(id, *tag)
		}
		return nil
	})

	type TagSetOptions struct {
		TagGetOptions
		Tag     []string `help:"tag to set, key:value"`
		Replace bool     `help:"replace all tags"`
	}
	shellutils.R(&TagSetOptions{}, "tag-set", "set tags of a specific resource", func(cli *aliyun.SRegion, args *TagSetOptions) error {
		tags := make(map[string]string)
		for _, t := range args.Tag {
			parts := strings.Split(t, ":")
			tags[parts[0]] = parts[1]
		}
		err := cli.SetResourceTags(args.RESTYPE, args.ID, tags, args.Replace)
		if err != nil {
			return err
		}
		return nil
	})
}
