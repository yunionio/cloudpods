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

	"yunion.io/x/onecloud/pkg/multicloud/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type TagAddOptions struct {
		ARN []string `help:"aws resource name"`
		Tag []string `help:"tag to set, key:value"`
	}
	shellutils.R(&TagAddOptions{}, "tag-add", "add tags", func(cli *aws.SRegion, args *TagAddOptions) error {
		tags := make(map[string]string)
		for _, t := range args.Tag {
			parts := strings.Split(t, ":")
			tags[parts[0]] = parts[1]
		}
		e := cli.TagResources(args.ARN, tags)
		if e != nil {
			return e
		}
		return nil
	})

	type UnTagOptions struct {
		ARN []string `help:"aws resource name"`
		Tag []string `help:"tag to set, key"`
	}
	shellutils.R(&UnTagOptions{}, "tag-del", "untag intances", func(cli *aws.SRegion, args *UnTagOptions) error {
		e := cli.UntagResources(args.ARN, args.Tag)
		if e != nil {
			return e
		}
		return nil
	})
}
