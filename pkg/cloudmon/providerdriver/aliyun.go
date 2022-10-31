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

package providerdriver

import (
	"context"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

type AliyunCollect struct {
	SCollectByMetricTypeDriver
}

func (self *AliyunCollect) GetProvider() string {
	return api.CLOUD_PROVIDER_ALIYUN
}

func (self *AliyunCollect) IsSupportMetrics() bool {
	return true
}

func (self *AliyunCollect) CollectAccountMetrics(ctx context.Context, account api.CloudaccountDetail) (influxdb.SMetricData, error) {
	metric := influxdb.SMetricData{
		Name:      string(cloudprovider.METRIC_RESOURCE_TYPE_CLOUD_ACCOUNT),
		Timestamp: time.Now(),
		Tags:      []influxdb.SKeyValue{},
		Metrics:   []influxdb.SKeyValue{},
	}
	for k, v := range account.GetMetricTags() {
		metric.Tags = append([]influxdb.SKeyValue{
			{
				Key:   k,
				Value: v,
			},
		}, metric.Tags...)
	}
	for k, v := range account.GetMetricPairs() {
		metric.Metrics = append([]influxdb.SKeyValue{
			{
				Key:   k,
				Value: v,
			},
		}, metric.Metrics...)
	}
	return metric, nil
}

func (self *AliyunCollect) GetDelayDuration() time.Duration {
	return time.Minute * 3
}

func init() {
	Register(&AliyunCollect{})
}
