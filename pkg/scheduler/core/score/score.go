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

package score

import (
	"fmt"
	"math"
	"strings"

	"yunion.io/x/pkg/tristate"
)

type TScore int

const (
	MinScore  TScore = -1
	ZeroScore TScore = 0
	MidScore  TScore = 1
	MaxScore  TScore = 2

	ZeroScoreName = "zero"
)

type SScore struct {
	Score TScore
	Name  string
}

func NewScore(score TScore, name string) SScore {
	return SScore{
		Score: score,
		Name:  name,
	}
}

func NewMinScore(name string) SScore {
	return NewScore(MinScore, name)
}

func NewZeroScore() SScore {
	return NewScore(ZeroScore, ZeroScoreName)
}

func NewMidScore(name string) SScore {
	return NewScore(MidScore, name)
}

func NewMaxScore(name string) SScore {
	return NewScore(MaxScore, name)
}

func (v SScore) GetScore() TScore {
	return v.Score
}

func (v SScore) String() string {
	return fmt.Sprintf("%s: %d", v.Name, v.Score)
}

type scores map[string]int

func (ss scores) Total() int {
	ret := 0
	for _, s := range ss {
		ret += s
	}
	return ret
}

type ScoreBucket struct {
	preferScore scores
	avoidScore  scores
	normalScore scores
}

func NewScoreBuckets() *ScoreBucket {
	return &ScoreBucket{
		preferScore: make(map[string]int),
		avoidScore:  make(map[string]int),
		normalScore: make(map[string]int),
	}
}

func (b *ScoreBucket) SetScore(score SScore, prefer tristate.TriState) *ScoreBucket {
	scoreToSet := b.normalScore
	if prefer.IsTrue() {
		scoreToSet = b.preferScore
	} else if prefer.IsFalse() {
		scoreToSet = b.avoidScore
	}
	scoreToSet[score.Name] = int(score.Score)
	return b
}

func (b *ScoreBucket) PreferScore() int {
	return b.preferScore.Total()
}

func (b *ScoreBucket) AvoidScore() int {
	return b.avoidScore.Total()
}

func (b *ScoreBucket) NormalScore() int {
	return b.normalScore.Total()
}

func (b *ScoreBucket) debugString(kind string, vals map[string]int) string {
	return fmt.Sprintf("%s: %v", kind, vals)
}

func (b *ScoreBucket) String() string {
	kinds := make([]string, 0)
	for kind, ss := range map[string]map[string]int{
		"prefer": b.preferScore,
		"avoid":  b.avoidScore,
		"normal": b.normalScore,
	} {
		kinds = append(kinds, b.debugString(kind, ss))
	}
	return strings.Join(kinds, "\n")
}

type Interval struct {
	start int64
	end   int64
}

func NewInterval(start, end int64) *Interval {
	return &Interval{start: start, end: end}
}

func (i Interval) IsContain(val int64) bool {
	return val >= i.start && val < i.end
}

type Intervals struct {
	MinInterval  *Interval
	ZeroInterval *Interval
	MidInterval  *Interval
	MaxInterval  *Interval
}

func NewIntervals(min, zero, mid int64) Intervals {
	return Intervals{
		MinInterval:  NewInterval(math.MinInt64, min),
		ZeroInterval: NewInterval(min, zero),
		MidInterval:  NewInterval(zero, mid),
		MaxInterval:  NewInterval(mid, math.MaxInt64),
	}
}

func (is Intervals) ToScore(val int64) TScore {
	for score, interval := range map[TScore]*Interval{
		MinScore:  is.MinInterval,
		ZeroScore: is.ZeroInterval,
		MidScore:  is.MidInterval,
		MaxScore:  is.MaxInterval,
	} {
		if interval != nil && interval.IsContain(val) {
			return score
		}
	}
	return ZeroScore
}
