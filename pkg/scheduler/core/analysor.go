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

package core

import (
	"fmt"
	"sort"
	"time"

	"yunion.io/x/log"
)

type predicateAnalysor struct {
	hint    string
	starts  map[string]time.Time
	elpased map[string]time.Duration
}

func newPredicateAnalysor(hint string) *predicateAnalysor {
	return &predicateAnalysor{
		hint:    hint,
		starts:  make(map[string]time.Time),
		elpased: make(map[string]time.Duration),
	}
}

func (p *predicateAnalysor) Start(pName string) *predicateAnalysor {
	p.starts[pName] = time.Now()
	return p
}

func (p *predicateAnalysor) End(pName string, end time.Time) *predicateAnalysor {
	start, ok := p.starts[pName]
	if !ok {
		panic(fmt.Sprintf("Not found start time of %q", pName))
	}
	p.elpased[pName] = end.Sub(start)
	return p
}

type predicateDuration struct {
	name     string
	duration time.Duration
}

type predicateDurations []*predicateDuration

func (p predicateDurations) Len() int {
	return len(p)
}

func (p predicateDurations) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p predicateDurations) Less(i, j int) bool {
	return p[i].duration > p[j].duration
}

func (p *predicateAnalysor) ShowResult() {
	lists := make([]*predicateDuration, 0)
	for name, d := range p.elpased {
		lists = append(lists, &predicateDuration{
			name:     name,
			duration: d,
		})
	}
	l := predicateDurations(lists)
	sort.Sort(l)
	log.Infof("=================Start %s Result=================", p.hint)
	for _, p := range l {
		log.Infof("%s: %s", p.name, p.duration)
	}
	log.Infof("=================End %s Result=======================", p.hint)
}
