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

const (
	BATCH_SEND_SIZE = 10000
)

func SendMetrics(urls []string, dbName string, metrics []SMetricData, debug bool) error {
	lines := make([]string, len(metrics))
	for i := range metrics {
		lines[i] = metrics[i].Line()
	}
	if len(lines) == 0 {
		return nil
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

func BatchSendMetrics(urls []string, dbName string, metrics []SMetricData, debug bool) error {
	lines := make([]string, len(metrics))
	for i := range metrics {
		lines[i] = metrics[i].Line()
	}
	if len(lines) == 0 {
		return nil
	}
	for _, url := range urls {
		db := NewInfluxdbWithDebug(url, debug)
		err := db.SetDatabase(dbName)
		if err != nil {
			return errors.Wrap(err, "SetDatabase")
		}
		errs := []error{}
		for i := 0; i < (len(lines)+BATCH_SEND_SIZE-1)/BATCH_SEND_SIZE; i++ {
			last := (i + 1) * BATCH_SEND_SIZE
			if last > len(lines) {
				last = len(lines)
			}
			err = db.BatchWrite(lines[i*BATCH_SEND_SIZE:last], "ms")
			if err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			return errors.Wrapf(err, "db.BatchWrite")
		}
	}
	return nil
}
