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

type HuaweiCollect struct {
	SCollectByResourceIdDriver
}

func (self *HuaweiCollect) GetProvider() string {
	return api.CLOUD_PROVIDER_HUAWEI
}

func (self *HuaweiCollect) IsSupportMetrics() bool {
	return true
}

func init() {
	Register(&HuaweiCollect{})
}

func (self *HuaweiCollect) CollectAccountMetrics(ctx context.Context, account api.CloudaccountDetail) (influxdb.SMetricData, error) {
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

func (self *HuaweiCollect) CollectModelartsPoolMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.ModelartsPoolDetails, start, end time.Time) error {
	metrics := []influxdb.SMetricData{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i := range res {
		wg.Add(1)
		go func(pool api.ModelartsPoolDetails) {
			defer func() {
				wg.Done()
			}()
			opts := &cloudprovider.MetricListOptions{
				ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_MODELARTS_POOL,
				StartTime:    start,
				EndTime:      end,
			}
			opts.ResourceId = pool.ExternalId
			opts.RegionExtId = pool.RegionExtId

			tags := []influxdb.SKeyValue{}
			for k, v := range pool.GetMetricTags() {
				tags = append(tags, influxdb.SKeyValue{
					Key:   k,
					Value: v,
				})
			}

			data, err := provider.GetMetrics(opts)
			if err != nil {
				if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					log.Errorf("get modelarts_pool %s(%s) error: %v", pool.Name, pool.Id, err)
					return
				}
				return
			}
			for _, values := range data {
				for _, value := range values.Values {
					metric := influxdb.SMetricData{
						Name:      values.MetricType.Name(),
						Timestamp: value.Timestamp,
						Tags:      tags,
						Metrics: []influxdb.SKeyValue{
							{
								Key:   values.MetricType.Key(),
								Value: strconv.FormatFloat(value.Value, 'E', -1, 64),
							},
						},
					}
					for k, v := range value.Tags {
						metric.Tags = append(metric.Tags, influxdb.SKeyValue{
							Key:   k,
							Value: v,
						})
					}
					mu.Lock()
					metrics = append(metrics, metric)
					mu.Unlock()
				}
			}
		}(res[i])
	}
	wg.Wait()

	return self.sendMetrics(ctx, manager, "modelarts_pool", len(res), metrics)
}
