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

package fileutils2

import (
	"os"

	"yunion.io/x/pkg/util/fileutils"
)

func Exists(filepath string) bool {
	_, err := os.Lstat(filepath)
	if err != nil {
		return false
	}
	return true
}

func IsFile(filepath string) bool {
	return fileutils.IsFile(filepath)
}

func IsDir(filepath string) bool {
	return fileutils.IsDir(filepath)
}
