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
)

type TsdbQueryEndpoint interface {
	Query(ctx context.Context, ds *DataSource, query *TsdbQuery) (*Response, error)
}

var registry map[string]GetTsdbQueryEndpointFn

type GetTsdbQueryEndpointFn func(dsInfo *DataSource) (TsdbQueryEndpoint, error)

func init() {
	registry = make(map[string]GetTsdbQueryEndpointFn)
}

const (
	ErrorNotFoundExecutorDataSource = errors.Error("Not find executor for data source")
)

func getTsdbQueryEndpointFor(dsInfo *DataSource) (TsdbQueryEndpoint, error) {
	if fn, exists := registry[dsInfo.Type]; exists {
		executor, err := fn(dsInfo)
		if err != nil {
			return nil, err
		}
		return executor, nil
	}
	return nil, errors.Wrapf(ErrorNotFoundExecutorDataSource, "type: %s", dsInfo.Type)
}

func RegisterTsdbQueryEndpoint(dataSourceType string, fn GetTsdbQueryEndpointFn) {
	registry[dataSourceType] = fn
}
