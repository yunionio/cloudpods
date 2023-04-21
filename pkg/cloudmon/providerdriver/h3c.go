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
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

type H3CCollect struct {
	SBaseCollectDriver
}

func (self *H3CCollect) GetProvider() string {
	return api.CLOUD_PROVIDER_H3C
}

func init() {
	Register(&H3CCollect{})
}

func (self *H3CCollect) IsSupportMetrics() bool {
	return true
}

func (self *H3CCollect) CollectServerMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.ServerDetails, start, end time.Time) error {
	metrics := []influxdb.SMetricData{}
	var mu sync.Mutex
	opts := &cloudprovider.MetricListOptions{
		ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_SERVER,
	}
	data := []cloudprovider.MetricValues{}
	for start.Before(end) {
		part, err := provider.GetMetrics(opts)
		if err != nil {
			continue
		}
		start = start.Add(time.Minute)
		time.Sleep(time.Minute)
		data = append(data, part...)
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

	return self.sendMetrics(ctx, manager, "server", len(res), metrics)
}

func (self *H3CCollect) CollectDBInstanceMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.DBInstanceDetails, start, end time.Time) error {
	metrics := []influxdb.SMetricData{}
	var mu sync.Mutex
	opts := &cloudprovider.MetricListOptions{
		ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_RDS,
	}
	data, err := provider.GetMetrics(opts)
	if err != nil {
		return errors.Wrapf(err, "GetMetrics")
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

	return self.sendMetrics(ctx, manager, "rds", len(res), metrics)
}
