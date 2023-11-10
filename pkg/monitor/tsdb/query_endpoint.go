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

package tsdb

import (
	"context"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
)

type TsdbQueryEndpoint interface {
	Query(ctx context.Context, ds *DataSource, query *TsdbQuery) (*Response, error)
	FilterMeasurement(ctx context.Context, ds *DataSource, from, to string, ms *monitor.InfluxMeasurement, tagFilter *monitor.MetricQueryTag) (*monitor.InfluxMeasurement, error)
}

var registry map[string]GetTsdbQueryEndpointFn

type GetTsdbQueryEndpointFn func(dsInfo *DataSource) (TsdbQueryEndpoint, error)

func init() {
	registry = make(map[string]GetTsdbQueryEndpointFn)
}

var (
	ErrorNotFoundExecutorDataSource error = errors.Error("Not find executor for data source")
)

func getDataSourceFunc(dsType string) (GetTsdbQueryEndpointFn, error) {
	fn, exists := registry[dsType]
	if !exists {
		return nil, errors.Wrapf(ErrorNotFoundExecutorDataSource, "type: %s", dsType)
	}
	return fn, nil
}

func GetTsdbQueryEndpointFor(dsInfo *DataSource) (TsdbQueryEndpoint, error) {
	fn, err := getDataSourceFunc(dsInfo.Type)
	if err != nil {
		return nil, errors.Wrap(err, "getDataSourceFunc")
	}
	executor, err := fn(dsInfo)
	if err != nil {
		return nil, errors.Wrap(err, "construct datasource query endpoint")
	}
	return executor, nil
}

func IsValidDataSource(dsType string) error {
	_, err := getDataSourceFunc(dsType)
	return err
}

func RegisterTsdbQueryEndpoint(dataSourceType string, fn GetTsdbQueryEndpointFn) {
	registry[dataSourceType] = fn
}
