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
	"strconv"
	"sync"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

type ApsaraCollect struct {
	SCollectByMetricTypeDriver
}

func (self *ApsaraCollect) GetProvider() string {
	return api.CLOUD_PROVIDER_APSARA
}

func (self *ApsaraCollect) IsSupportMetrics() bool {
	return true
}

func init() {
	Register(&ApsaraCollect{})
}

func (self *ApsaraCollect) CollectEipMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.ElasticipDetails, start, end time.Time) error {
	metrics := []influxdb.SMetricData{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, _metricType := range cloudprovider.ALL_EIP_TYPES {
		wg.Add(1)
		go func(metricType cloudprovider.TMetricType) {
			defer func() {
				wg.Done()
			}()
			opts := &cloudprovider.MetricListOptions{
				ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_EIP,
				MetricType:   metricType,
				StartTime:    start,
				EndTime:      end,
			}
			data, err := provider.GetMetrics(opts)
			if err != nil {
				if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					log.Errorf("get eip %s(%s) %s error: %v", manager.Name, manager.Id, metricType, err)
					return
				}
				return
			}
			for _, value := range data {
				eip, ok := res[value.Id]
				if !ok {
					continue
				}
				tags := []influxdb.SKeyValue{}
				for k, v := range eip.GetMetricTags() {
					tags = append(tags, influxdb.SKeyValue{
						Key:   k,
						Value: v,
					})
				}
				for _, v := range value.Values {
					metric := influxdb.SMetricData{
						Name:      value.MetricType.Name(),
						Timestamp: v.Timestamp,
						Tags:      []influxdb.SKeyValue{},
						Metrics: []influxdb.SKeyValue{
							{
								Key:   value.MetricType.Key(),
								Value: strconv.FormatFloat(v.Value, 'E', -1, 64),
							},
						},
					}
					for k, v := range v.Tags {
						metric.Tags = append(metric.Tags, influxdb.SKeyValue{
							Key:   k,
							Value: v,
						})
					}
					metric.Tags = append(metric.Tags, tags...)
					mu.Lock()
					metrics = append(metrics, metric)
					mu.Unlock()
				}
			}
		}(_metricType)
	}
	wg.Wait()

	return self.sendMetrics(ctx, manager, "elasticip", len(res), metrics)
}
