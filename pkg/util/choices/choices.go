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

package choices

import (
	"strings"
)

type Empty struct{}
type Choices map[string]Empty

func NewChoices(choices ...string) Choices {
	cs := Choices{}
	for _, choice := range choices {
		cs[choice] = Empty{}
	}
	return cs
}

func (cs Choices) Has(choice string) bool {
	_, ok := cs[choice]
	return ok
}

func (cs Choices) String() string {
	choices := make([]string, len(cs))
	i := 0
	for choice := range cs {
		choices[i] = choice
		i++
	}
	return strings.Join(choices, "|")
}
