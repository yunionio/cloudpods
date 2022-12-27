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
	"yunion.io/x/pkg/util/shellutils"

	"yunion.io/x/onecloud/pkg/util/zeroclean"
)

func init() {
	type ZeroCleanOptions struct {
		Recursive bool   `help:"recursive" optional:"true"`
		FILE      string `help:"file to clean"`
	}
	shellutils.R(&ZeroCleanOptions{}, "zero-clean", "Zero clean file", func(args *ZeroCleanOptions) error {
		var err error
		if args.Recursive {
			err = zeroclean.ZeroDir(args.FILE)
		} else {
			err = zeroclean.ZeroFile(args.FILE)
		}
		if err != nil {
			return err
		}
		return nil
	})
}
