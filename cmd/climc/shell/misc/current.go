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

package misc

import (
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	type CurrentUserOptions struct {
	}
	R(&CurrentUserOptions{}, "session-show", "show information of current account", func(s *mcclient.ClientSession, args *CurrentUserOptions) error {
		printObject(s.ToJson())
		return nil
	})
}
