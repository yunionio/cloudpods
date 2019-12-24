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

package conditionparser

import (
	"regexp"
	"strings"

	"yunion.io/x/pkg/errors"
)

var (
	tempPattern = regexp.MustCompile(`\$\{[\w\d._-]*\}`)
)

func IsTemplate(template string) bool {
	return tempPattern.MatchString(template)
}

func EvalTemplate(template string, input interface{}) (string, error) {
	var output strings.Builder
	matches := tempPattern.FindAllStringSubmatchIndex(template, -1)
	offset := 0
	for _, match := range matches {
		output.WriteString(template[offset:match[0]])
		o, err := EvalString(template[match[0]+2:match[1]-1], input)
		if err != nil {
			return "", errors.Wrap(err, template[match[0]+2:match[1]])
		}
		output.WriteString(o)
		offset = match[1]
	}
	if offset < len(template) {
		output.WriteString(template[offset:])
	}
	return output.String(), nil
}
