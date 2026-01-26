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
	"os"
	"path/filepath"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/shellutils"

	"yunion.io/x/onecloud/pkg/util/isoutils"
)

func init() {
	type DetectOSOptions struct {
		ISO string `help:"ISO file"`
	}
	shellutils.R(&DetectOSOptions{}, "detect", "Detect", func(args *DetectOSOptions) error {
		stat, err := os.Stat(args.ISO)
		if err != nil {
			return err
		}

		if stat.IsDir() {
			// 如果是目录，仅遍历一层目录下的 .iso 文件
			entries, err := os.ReadDir(args.ISO)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue // 跳过子目录
				}
				if strings.HasSuffix(strings.ToLower(entry.Name()), ".iso") {
					path := filepath.Join(args.ISO, entry.Name())
					fmt.Printf("Processing: %s\n", path)
					f, err := os.Open(path)
					if err != nil {
						fmt.Printf("Error opening %s: %v\n", path, err)
						continue // 继续处理其他文件
					}
					isoInfo, err := isoutils.DetectOSFromISO(f)
					f.Close()
					if err != nil {
						fmt.Printf("Error detecting OS from %s: %v\n", path, err)
						continue
					}
					fmt.Printf("%s: %s\n", path, jsonutils.Marshal(isoInfo))
				}
			}
			return nil
		}

		// 如果是文件，按原逻辑处理
		f, err := os.Open(args.ISO)
		if err != nil {
			return err
		}
		defer f.Close()
		isoInfo, err := isoutils.DetectOSFromISO(f)
		if err != nil {
			return err
		}
		fmt.Printf("%s: %s\n", args.ISO, jsonutils.Marshal(isoInfo))
		return nil
	})
}
