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

package regutils2

import "regexp"

/**
 * Parses val with the given regular expression and returns the
 * group values defined in the expression.
 */
func GetParams(compRegEx *regexp.Regexp, val string) map[string]string {
	match := compRegEx.FindStringSubmatch(val)

	paramsMap := make(map[string]string, 0)
	for i, name := range compRegEx.SubexpNames() {
		if i > 0 && i <= len(match) {
			paramsMap[name] = match[i]
		}
	}
	return paramsMap
}

func SubGroupMatch(pattern string, line string) map[string]string {
	regEx := regexp.MustCompile(pattern)
	return GetParams(regEx, line)
}
