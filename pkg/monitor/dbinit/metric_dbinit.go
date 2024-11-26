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

package dbinit

import (
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/monitor/dbinit/measurements"
)

var MetricNeedDeleteDescriptions = []string{}
var metricInitInputMap map[string]monitor.MetricCreateInput

func RegistryMetricCreateInput(name, displayName, resType, database string, score int,
	fields []monitor.MetricFieldCreateInput) {
	if metricInitInputMap == nil {
		metricInitInputMap = make(map[string]monitor.MetricCreateInput)
	}
	if _, ok := metricInitInputMap[name]; ok {
		log.Fatalf("inputMeasurementName: %q has already existed.", name)
		return
	}
	metricInitInputMap[name] = monitor.MetricCreateInput{
		Measurement: monitor.MetricMeasurementCreateInput{
			StandaloneResourceCreateInput: apis.StandaloneResourceCreateInput{Name: name},
			ResType:                       resType,
			DisplayName:                   displayName,
			Database:                      database,
			Score:                         score,
		},
		MetricFields: fields,
	}
}

func GetRegistryMetricInput() (metricInitInputs []monitor.MetricCreateInput) {
	if metricInitInputMap == nil {
		metricInitInputMap = make(map[string]monitor.MetricCreateInput)
	}
	for name := range metricInitInputMap {
		metricInitInputs = append(metricInitInputs, metricInitInputMap[name])
	}
	return
}

func newMetricFieldCreateInput(name, displayName, unit string, score int) monitor.MetricFieldCreateInput {
	return monitor.MetricFieldCreateInput{
		StandaloneResourceCreateInput: apis.StandaloneResourceCreateInput{Name: name},
		DisplayName:                   displayName,
		Unit:                          unit,
		ValueType:                     "",
		Score:                         score,
	}
}

// order by score asc
// score default:99
func init() {
	var measurements = measurements.All

	metricCount := 0
	scoreIdx := 0
	for mIdx := range measurements {
		measurement := measurements[mIdx]
		for ctxIdx := range measurement.Context {
			ctx := measurement.Context[ctxIdx]
			inputs := []monitor.MetricFieldCreateInput{}
			for metricIdx := range measurement.Metrics {
				metric := measurement.Metrics[metricIdx]
				inputs = append(inputs, newMetricFieldCreateInput(metric.Name, metric.DisplayName, metric.Unit, metricIdx+1))
				metricCount++
			}
			RegistryMetricCreateInput(ctx.Name, ctx.DisplayName, ctx.ResourceType, ctx.Database, scoreIdx+1, inputs)
			scoreIdx++
		}
	}

	log.Infof("[monitor.dbinit] Register %d metrics", metricCount)
}
