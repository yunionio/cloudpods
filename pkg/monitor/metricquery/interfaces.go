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

package metricquery

import (
	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type MetricQuery interface {
	ExecuteQuery(userCred mcclient.TokenCredential, scope string, forceCheckSeries bool) (*monitor.MetricsQueryResult, error)
}

type QueryFactory func(model []*monitor.AlertCondition) (MetricQuery, error)

var queryFactories = make(map[string]QueryFactory)

func RegisterMetricQuery(typeName string, factory QueryFactory) {
	queryFactories[typeName] = factory
}

func GetQueryFactories() map[string]QueryFactory {
	return queryFactories
}
