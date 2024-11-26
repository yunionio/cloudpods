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

package measurements

import (
	"fmt"
	"os"
	"testing"
)

func TestOutputMetrics(t *testing.T) {
	f, err := os.Create("metrics.csv")
	if err != nil {
		t.Fatalf("create metrics.csv fail %s", err)
	}
	defer f.Close()
	qCSV := func(str string) string {
		return fmt.Sprintf("%q", str)
	}
	f.WriteString("Measurement,MeasurementNote,ResourceType,Database,Metric,MetricNote,MetricUnit\n")
	for _, measurement := range All {
		for _, ctx := range measurement.Context {
			for _, metric := range measurement.Metrics {
				f.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s\n", qCSV(ctx.Name), qCSV(ctx.DisplayName), qCSV(ctx.ResourceType), qCSV(ctx.Database), qCSV(metric.Name), qCSV(metric.DisplayName), qCSV(metric.Unit)))
			}
		}
	}
}
