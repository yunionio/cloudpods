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

package stringutils2

import (
	"strings"
)

func HasSuffixIgnoreCase(str string, suffix string) bool {
	if len(str) < len(suffix) {
		return false
	}
	return strings.EqualFold(str[len(str)-len(suffix):len(str)], suffix)
}

func HasPrefixIgnoreCase(str string, prefix string) bool {
	if len(str) < len(prefix) {
		return false
	}
	return strings.EqualFold(str[0:len(prefix)], prefix)
}
