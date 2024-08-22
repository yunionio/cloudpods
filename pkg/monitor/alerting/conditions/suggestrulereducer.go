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

package conditions

import (
	"time"

	"yunion.io/x/onecloud/pkg/apis/monitor"
)

type suggestRuleReducer struct {
	*queryReducer
	duration time.Duration
}

func NewSuggestRuleReducer(t string, duration time.Duration) Reducer {
	return &suggestRuleReducer{
		queryReducer: &queryReducer{Type: monitor.ReducerType(t)},
		duration:     duration,
	}
}

func (s *suggestRuleReducer) Reduce(series *monitor.TimeSeries) (*float64, []string) {
	/*if int(s.duration.Seconds()) > len(series.Points) {
		return nil, nil
	}*/
	return s.queryReducer.Reduce(series)
}
