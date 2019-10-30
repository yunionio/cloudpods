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

package influxdb

import (
	"yunion.io/x/pkg/errors"
)

func SendMetrics(urls []string, dbName string, metrics []SMetricData, debug bool) error {
	lines := make([]string, len(metrics))
	for i := range metrics {
		lines[i] = metrics[i].Line()
	}
	for _, url := range urls {
		db := NewInfluxdbWithDebug(url, debug)
		err := db.SetDatabase(dbName)
		if err != nil {
			return errors.Wrap(err, "SetDatabase")
		}
		for _, line := range lines {
			err = db.Write(line, "ms")
			if err != nil {
				return errors.Wrap(err, "db.Write")
			}
		}
	}
	return nil
}
