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

package sortutils

import "strings"

type CompareResult int

const (
	Equal = CompareResult(0)
	Less  = CompareResult(-1)
	More  = CompareResult(1)
)

func CompareString(str1, str2 string) CompareResult {
	return CompareResult(strings.Compare(str1, str2))
}
