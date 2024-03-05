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
	"strings"
	"sync"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

type CtyunCollect struct {
	SBaseCollectDriver
}

func (self *CtyunCollect) GetProvider() string {
	return api.CLOUD_PROVIDER_CTYUN
}

func (self *CtyunCollect) IsSupportMetrics() bool {
	return true
}

func init() {
	Register(&CtyunCollect{})
}

func (self *CtyunCollect) CollectServerMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.ServerDetails, start, end time.Time) error {
	metrics := []influxdb.SMetricData{}
	regionServers := map[string][]api.ServerDetails{}
	for i := range res {
		regionId := res[i].RegionExtId
		_, ok := regionServers[regionId]
		if !ok {
			regionServers[regionId] = []api.ServerDetails{}
		}
		regionServers[regionId] = append(regionServers[regionId], res[i])
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for regionId, servers := range regionServers {
		wg.Add(1)
		go func(regionId string, servers []api.ServerDetails) {
			defer func() {
				wg.Done()
			}()
			data := []cloudprovider.MetricValues{}
			for i := 0; i < (len(servers)+9)/10; i++ {
				opts := &cloudprovider.MetricListOptions{
					ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_SERVER,
					RegionExtId:  regionId,
					StartTime:    start,
					EndTime:      end,
				}
				last := (i + 1) * 10
				if last > len(servers) {
					last = len(servers)
				}
				for i := range servers[i*10 : last] {
					opts.ResourceIds = append(opts.ResourceIds, servers[i].ExternalId)
				}
				part, err := provider.GetMetrics(opts)
				if err != nil {
					if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
						log.Errorf("get server %s(%s) error: %v", strings.Join(opts.ResourceIds, ","), regionId, err)
						continue
					}
					continue
				}
				data = append(data, part...)
			}
			for _, value := range data {
				server, ok := res[value.Id]
				if !ok {
					continue
				}
				tags := []influxdb.SKeyValue{}
				for k, v := range server.GetMetricTags() {
					tags = append(tags, influxdb.SKeyValue{
						Key:   k,
						Value: v,
					})
				}
				pairs := []influxdb.SKeyValue{}
				for k, v := range server.GetMetricPairs() {
					pairs = append(pairs, influxdb.SKeyValue{
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
					metric.Metrics = append(metric.Metrics, pairs...)
					metric.Tags = append(metric.Tags, tags...)
					mu.Lock()
					metrics = append(metrics, metric)
					mu.Unlock()
				}
			}
		}(regionId, servers)
	}
	wg.Wait()

	return self.sendMetrics(ctx, manager, "server", len(res), metrics)
}
