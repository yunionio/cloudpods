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
	"context"
	"fmt"

	"yunion.io/x/onecloud/pkg/util/redfish"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ExploreInput struct {
		Element []string `help:"explore path" positional:"true" optional:"true"`
	}
	shellutils.R(&ExploreInput{}, "explore", "explore redfish API", func(cli redfish.IRedfishDriver, args *ExploreInput) error {
		path, resp, err := cli.GetResource(context.Background(), args.Element...)
		if err != nil {
			return err
		}
		fmt.Println(path)
		fmt.Println(resp.PrettyString())
		return nil
	})
}
