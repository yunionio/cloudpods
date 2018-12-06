package score

import (
	"container/list"
	"fmt"
	"math"
	//"yunion.io/x/log"
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

type Scores struct {
	scores *list.List
}

func newScores() *Scores {
	return &Scores{
		scores: list.New(),
	}
}

func (s *Scores) Append(scores ...SScore) *Scores {
	for _, score := range scores {
		s.scores.PushBack(score)
	}
	return s
}

func (s *Scores) AddToFirst(score SScore) *Scores {
	s.scores.PushFront(score)
	return s
}

func (s *Scores) Range(iterFunc func(ele *list.Element, score SScore) bool) {
	for ele := s.scores.Front(); ele != nil; ele = ele.Next() {
		cont := iterFunc(ele, ele.Value.(SScore))
		if !cont {
			break
		}
	}
}

func (s *Scores) SetScore(score SScore) *Scores {
	exists := false
	rf := func(ele *list.Element, oscore SScore) bool {
		if oscore.Name == score.Name {
			exists = true
			oscore.Score = score.Score
			ele.Value = oscore
			return false
		}
		return true
	}
	s.Range(rf)
	if !exists {
		s.Append(score)
	}
	return s
}

func (s *Scores) AddScore(score SScore) *Scores {
	exists := false
	rf := func(ele *list.Element, oscore SScore) bool {
		if oscore.Name == score.Name {
			exists = true
			oscore.Score += score.Score
			ele.Value = oscore
			return false
		}
		return true
	}
	s.Range(rf)
	if !exists {
		s.Append(score)
	}
	return s
}

func (s *Scores) Len() int {
	return s.scores.Len()
}

func (s *Scores) GetScores() []SScore {
	ret := make([]SScore, 0)
	rf := func(_ *list.Element, score SScore) bool {
		ret = append(ret, score)
		return true
	}
	s.Range(rf)
	return ret
}

type ScoreBucket struct {
	scores *Scores
}

func NewScoreBuckets() *ScoreBucket {
	return &ScoreBucket{
		scores: newScores(),
	}
}

func (b *ScoreBucket) AddToFirst(score SScore) *ScoreBucket {
	b.scores.AddToFirst(score)
	return b
}

func (b *ScoreBucket) Append(scores ...SScore) *ScoreBucket {
	b.scores.Append(scores...)
	return b
}

func (b *ScoreBucket) GetScores() []SScore {
	return b.scores.GetScores()
}

func (b *ScoreBucket) SetScore(score SScore) *ScoreBucket {
	b.scores.SetScore(score)
	return b
}

func (b *ScoreBucket) AddScore(score SScore) *ScoreBucket {
	b.scores.AddScore(score)
	return b
}

func (b *ScoreBucket) GetScore(scoreName string) (int, SScore) {
	for i, oscore := range b.scores.GetScores() {
		if oscore.Name == scoreName {
			return i, oscore
		}
	}
	return -1, SScore{}
}

func (b *ScoreBucket) Len() int {
	return b.scores.Len()
}

func (b *ScoreBucket) DigitString() string {
	s := ""
	rf := func(_ *list.Element, score SScore) bool {
		s = fmt.Sprintf("%s%d", s, score.Score)
		return true
	}
	b.scores.Range(rf)
	return s
}

func extend(scores []SScore, length int) []SScore {
	olen := len(scores)
	if olen >= length {
		return scores
	}
	ret := make([]SScore, 0)
	zeroDigits := length - olen
	for i := 0; i < zeroDigits; i++ {
		ret = append(ret, NewZeroScore())
	}
	ret = append(ret, scores...)
	return ret
}

func Equal(b1, b2 *ScoreBucket) bool {
	return compare(b1, b2, func(s1, s2 TScore) bool { return s1 == s2 })
}

func Less(b1, b2 *ScoreBucket) bool {
	return compare(b1, b2, func(s1, s2 TScore) bool { return s1 < s2 })
}

func compare(b1, b2 *ScoreBucket, cf func(s1, s2 TScore) bool) bool {
	maxLen := int(math.Max(float64(b1.Len()), float64(b2.Len())))
	s1 := b1.GetScores()
	s2 := b2.GetScores()
	s1 = extend(s1, maxLen)
	s2 = extend(s2, maxLen)
	for i := range s1 {
		v1 := s1[i].GetScore()
		v2 := s2[i].GetScore()
		ok := cf(v1, v2)
		if ok {
			return true
		} else if !ok {
			return false
		}
	}
	return false
}

func (b *ScoreBucket) debugString(vals []SScore, ret string) string {
	if len(vals) == 0 {
		return ret
	}
	restVal := vals[1:]
	if len(restVal) == 0 {
		return vals[0].String()
	}
	str := b.debugString(restVal, ret)
	str = fmt.Sprintf("%s, %s", vals[0].String(), str)
	return str
}

func (b *ScoreBucket) String() string {
	return b.debugString(b.GetScores(), "")
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
