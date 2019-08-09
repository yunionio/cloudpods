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

package structarg

import (
	"fmt"
	"sort"
	"strings"

	"github.com/texttheater/golang-levenshtein/levenshtein"
)

type stringDistance struct {
	str string
	/* hanming distance */
	dist int
	/* similarity rate, 0~1: totally different ~ identical */
	rate float64
}

type LevenshteinStrings struct {
	target     string
	candidates []stringDistance
}

func (strs LevenshteinStrings) Len() int {
	return len(strs.candidates)
}

func (strs LevenshteinStrings) Swap(i, j int) {
	strs.candidates[i], strs.candidates[j] = strs.candidates[j], strs.candidates[i]
}

func (strs LevenshteinStrings) Less(i, j int) bool {
	if strs.candidates[i].dist != strs.candidates[j].dist {
		if strs.candidates[i].dist < strs.candidates[j].dist {
			return true
		} else {
			return false
		}
	}
	if strs.candidates[i].rate != strs.candidates[j].rate {
		if strs.candidates[i].rate > strs.candidates[j].rate {
			return true
		} else {
			return false
		}
	}
	if strs.candidates[i].str < strs.candidates[j].str {
		return true
	}
	return false
}

/**
 *
 * minRate: minimal similarity ratio, between 0.0~1.0, 0.0: totally different, 1.0: exactly identitical
 */
func FindSimilar(niddle string, stack []string, maxDist int, minRate float64) []string {
	cands := make([]stringDistance, 0)
	for i := 0; i < len(stack); i += 1 {
		cand := stringDistance{}
		dist := levenshtein.DistanceForStrings([]rune(stack[i]), []rune(niddle), levenshtein.DefaultOptions)
		rate := 1.0
		if len(stack[i])+len(niddle) > 0 {
			rate = float64(len(stack[i])+len(niddle)-dist) / float64(len(stack[i])+len(niddle))
		}
		if (maxDist < 0 || dist <= maxDist) && (minRate < 0.0 || minRate > 1.0 || rate >= minRate) {
			cand.str = stack[i]
			cand.dist = dist
			cand.rate = rate
			cands = append(cands, cand)
		}
	}
	lstrs := LevenshteinStrings{target: niddle, candidates: cands}
	sort.Sort(lstrs)
	result := make([]string, len(cands))
	for i := 0; i < len(result); i += 1 {
		result[i] = lstrs.candidates[i].str
	}
	return result
}

func ChoicesString(choices []string) string {
	if len(choices) == 0 {
		return ""
	}
	if len(choices) == 1 {
		return choices[0]
	}
	if len(choices) == 2 {
		return strings.Join(choices, " or ")
	}
	return fmt.Sprintf("%s or %s", strings.Join(choices[:len(choices)-1], ", "), choices[len(choices)-1])
}

func quotedChoicesString(choices []string) string {
	quoted := make([]string, len(choices))
	for i, c := range choices {
		quoted[i] = fmt.Sprintf("%q", c)
	}
	return ChoicesString(quoted)
}
