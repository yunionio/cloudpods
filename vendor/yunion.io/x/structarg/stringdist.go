package structarg

import (
	"github.com/texttheater/golang-levenshtein/levenshtein"
	"sort"
	"strings"
	"fmt"
)

type stringDistance struct {
	str  string
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
		if len(stack[i]) + len(niddle) > 0 {
			rate = float64(len(stack[i]) + len(niddle) - dist)/float64(len(stack[i]) + len(niddle))
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