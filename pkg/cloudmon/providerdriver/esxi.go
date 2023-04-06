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

type EsxiCollect struct {
	SCollectByMetricTypeDriver
}

func (self *EsxiCollect) GetProvider() string {
	return api.CLOUD_PROVIDER_VMWARE
}

func (self *EsxiCollect) IsSupportMetrics() bool {
	return true
}

func (self *EsxiCollect) GetDelayDuration() time.Duration {
	return time.Minute * 4
}

func init() {
	Register(&EsxiCollect{})
}

func (self *EsxiCollect) CollectServerMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.ServerDetails, start, end time.Time) error {
	metrics := []influxdb.SMetricData{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, _metricType := range cloudprovider.ALL_VM_METRIC_TYPES {
		wg.Add(1)
		go func(metricType cloudprovider.TMetricType) {
			defer func() {
				wg.Done()
			}()
			opts := &cloudprovider.MetricListOptions{
				ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_SERVER,
				MetricType:   metricType,

				StartTime: start,
				EndTime:   end,
			}

			// 磁盘使用率esxi每半小时才有一次数据, 采集时间得提前半小时，否则采集不到数据
			if metricType == cloudprovider.VM_METRIC_TYPE_DISK_USAGE {
				opts.StartTime = opts.StartTime.Add(time.Minute * -30)
				opts.EndTime = opts.EndTime.Add(time.Minute * -30)
			}

			data, err := provider.GetMetrics(opts)
			if err != nil {
				if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					log.Errorf("get server metric %s for %s(%s) error: %v", metricType, manager.Name, manager.Id, err)
					return
				}
				return
			}
			for _, value := range data {
				vm, ok := res[value.Id]
				if !ok {
					continue
				}
				tags := []influxdb.SKeyValue{}
				for k, v := range vm.GetMetricTags() {
					tags = append(tags, influxdb.SKeyValue{
						Key:   k,
						Value: v,
					})
				}
				pairs := []influxdb.SKeyValue{}
				for k, v := range vm.GetMetricPairs() {
					pairs = append(pairs, influxdb.SKeyValue{
						Key:   k,
						Value: v,
					})
				}
				for _, v := range value.Values {
					if value.MetricType == cloudprovider.VM_METRIC_TYPE_DISK_USAGE {
						if vm.DiskSizeMb == 0 { // avoid div zero
							vm.DiskSizeMb = 1
							v.Value = 0
						}
						v.Value = v.Value / 1024 / 1024 / float64(vm.DiskSizeMb) * 100
					}
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
		}(_metricType)

	}
	wg.Wait()

	return self.sendMetrics(ctx, manager, "server", len(res), metrics)
}
