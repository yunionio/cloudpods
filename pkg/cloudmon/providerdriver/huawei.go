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

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudmon/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
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
	s := auth.GetAdminSession(ctx, options.Options.Region)
	urls, err := s.GetServiceURLs(apis.SERVICE_TYPE_INFLUXDB, options.Options.SessionEndpointType, "")
	if err != nil {
		return errors.Wrap(err, "GetServiceURLs")
	}
	log.Infof("send %d modelarts_pool with %d metrics for %s(%s)", len(res), len(metrics), manager.Name, manager.Id)
	return influxdb.BatchSendMetrics(urls, options.Options.InfluxDatabase, metrics, false)
}
