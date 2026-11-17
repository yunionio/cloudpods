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

type QcloudCollect struct {
	SBaseCollectDriver
}

func (self *QcloudCollect) GetProvider() string {
	return api.CLOUD_PROVIDER_QCLOUD
}

func (self *QcloudCollect) IsSupportMetrics() bool {
	return true
}

func init() {
	Register(&QcloudCollect{})
}

func (self *QcloudCollect) CollectServerMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.ServerDetails, start, end time.Time) error {
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
				for j := range servers[i*10 : last] {
					opts.ResourceIds = append(opts.ResourceIds, servers[i*10+j].ExternalId)
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

func (self *QcloudCollect) CollectDBInstanceMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.DBInstanceDetails, start, end time.Time) error {
	metrics := []influxdb.SMetricData{}
	regionDBInstances := map[string][]api.DBInstanceDetails{}
	for i := range res {
		regionId := res[i].RegionExtId
		_, ok := regionDBInstances[regionId]
		if !ok {
			regionDBInstances[regionId] = []api.DBInstanceDetails{}
		}
		regionDBInstances[regionId] = append(regionDBInstances[regionId], res[i])
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for regionId, servers := range regionDBInstances {
		wg.Add(1)
		go func(regionId string, servers []api.DBInstanceDetails) {
			defer func() {
				wg.Done()
			}()
			data := []cloudprovider.MetricValues{}
			for i := 0; i < (len(servers)+9)/10; i++ {
				opts := &cloudprovider.MetricListOptions{
					ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_RDS,
					RegionExtId:  regionId,
					StartTime:    start,
					EndTime:      end,
				}
				last := (i + 1) * 10
				if last > len(servers) {
					last = len(servers)
				}
				for j := range servers[i*10 : last] {
					opts.ResourceIds = append(opts.ResourceIds, servers[i*10+j].ExternalId)
				}

				part, err := provider.GetMetrics(opts)
				if err != nil {
					if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
						log.Errorf("get rds %s(%s) error: %v", strings.Join(opts.ResourceIds, ","), regionId, err)
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

	return self.sendMetrics(ctx, manager, "rds", len(res), metrics)
}

func (self *QcloudCollect) CollectRedisMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.ElasticcacheDetails, start, end time.Time) error {
	metrics := []influxdb.SMetricData{}
	regionRedis := map[string][]api.ElasticcacheDetails{}
	for i := range res {
		regionId := res[i].RegionExtId
		_, ok := regionRedis[regionId]
		if !ok {
			regionRedis[regionId] = []api.ElasticcacheDetails{}
		}
		regionRedis[regionId] = append(regionRedis[regionId], res[i])
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for regionId, servers := range regionRedis {
		wg.Add(1)
		go func(regionId string, servers []api.ElasticcacheDetails) {
			defer func() {
				wg.Done()
			}()
			data := []cloudprovider.MetricValues{}
			for i := 0; i < (len(servers)+9)/10; i++ {
				opts := &cloudprovider.MetricListOptions{
					ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_REDIS,
					RegionExtId:  regionId,
					StartTime:    start,
					EndTime:      end,
				}
				last := (i + 1) * 10
				if last > len(servers) {
					last = len(servers)
				}
				for j := range servers[i*10 : last] {
					opts.ResourceIds = append(opts.ResourceIds, servers[i*10+j].ExternalId)
				}

				part, err := provider.GetMetrics(opts)
				if err != nil {
					if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
						log.Errorf("get redis %s(%s) error: %v", strings.Join(opts.ResourceIds, ","), regionId, err)
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

	return self.sendMetrics(ctx, manager, "redis", len(res), metrics)
}

func (self *QcloudCollect) CollectK8sMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.KubeClusterDetails, start, end time.Time) error {
	base := &SCollectByResourceIdDriver{}
	return base.CollectK8sMetrics(ctx, manager, provider, res, start, end)
}

func (self *QcloudCollect) CollectNatMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.NatgatewayDetails, start, end time.Time) error {
	metrics := []influxdb.SMetricData{}
	regionNats := map[string][]api.NatgatewayDetails{}
	for i := range res {
		regionId := res[i].RegionExtId
		regionNats[regionId] = append(regionNats[regionId], res[i])
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for regionId, nats := range regionNats {
		wg.Add(1)
		go func(regionId string, nats []api.NatgatewayDetails) {
			defer wg.Done()
			data := []cloudprovider.MetricValues{}
			for i := 0; i < (len(nats)+9)/10; i++ {
				opts := &cloudprovider.MetricListOptions{
					ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_NAT,
					RegionExtId:  regionId,
					StartTime:    start,
					EndTime:      end,
				}
				last := (i + 1) * 10
				if last > len(nats) {
					last = len(nats)
				}
				for j := range nats[i*10 : last] {
					opts.ResourceIds = append(opts.ResourceIds, nats[i*10+j].ExternalId)
				}
				part, err := provider.GetMetrics(opts)
				if err != nil {
					if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
						log.Errorf("get nat %s(%s) error: %v", strings.Join(opts.ResourceIds, ","), regionId, err)
					}
					continue
				}
				data = append(data, part...)
			}
			for _, value := range data {
				nat, ok := res[value.Id]
				if !ok {
					continue
				}
				tags := []influxdb.SKeyValue{}
				for k, v := range nat.GetMetricTags() {
					tags = append(tags, influxdb.SKeyValue{Key: k, Value: v})
				}
				for _, v := range value.Values {
					metric := influxdb.SMetricData{
						Name:      value.MetricType.Name(),
						Timestamp: v.Timestamp,
						Tags:      tags,
						Metrics: []influxdb.SKeyValue{
							{Key: value.MetricType.Key(), Value: strconv.FormatFloat(v.Value, 'E', -1, 64)},
						},
					}
					mu.Lock()
					metrics = append(metrics, metric)
					mu.Unlock()
				}
			}
		}(regionId, nats)
	}
	wg.Wait()
	return self.sendMetrics(ctx, manager, "nat", len(res), metrics)
}

func (self *QcloudCollect) CollectEipMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.ElasticipDetails, start, end time.Time) error {
	metrics := []influxdb.SMetricData{}
	regionEips := map[string][]api.ElasticipDetails{}
	ipMap := map[string]api.ElasticipDetails{}
	for i := range res {
		regionId := res[i].RegionExtId
		regionEips[regionId] = append(regionEips[regionId], res[i])
		if len(res[i].IpAddr) > 0 {
			ipMap[res[i].IpAddr] = res[i]
		}
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for regionId, eips := range regionEips {
		wg.Add(1)
		go func(regionId string, eips []api.ElasticipDetails) {
			defer wg.Done()
			data := []cloudprovider.MetricValues{}
			for i := 0; i < (len(eips)+9)/10; i++ {
				opts := &cloudprovider.MetricListOptions{
					ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_EIP,
					RegionExtId:  regionId,
					StartTime:    start,
					EndTime:      end,
				}
				last := (i + 1) * 10
				if last > len(eips) {
					last = len(eips)
				}
				for j := range eips[i*10 : last] {
					if len(eips[i*10+j].IpAddr) > 0 {
						opts.ResourceIds = append(opts.ResourceIds, eips[i*10+j].IpAddr)
					}
				}
				if len(opts.ResourceIds) == 0 {
					continue
				}
				part, err := provider.GetMetrics(opts)
				if err != nil {
					if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
						log.Errorf("get eip %s(%s) error: %v", strings.Join(opts.ResourceIds, ","), regionId, err)
					}
					continue
				}
				data = append(data, part...)
			}
			for _, value := range data {
				eip, ok := ipMap[value.Id]
				if !ok {
					continue
				}
				tags := []influxdb.SKeyValue{}
				for k, v := range eip.GetMetricTags() {
					tags = append(tags, influxdb.SKeyValue{Key: k, Value: v})
				}
				for _, v := range value.Values {
					metric := influxdb.SMetricData{
						Name:      value.MetricType.Name(),
						Timestamp: v.Timestamp,
						Tags:      tags,
						Metrics: []influxdb.SKeyValue{
							{Key: value.MetricType.Key(), Value: strconv.FormatFloat(v.Value, 'E', -1, 64)},
						},
					}
					mu.Lock()
					metrics = append(metrics, metric)
					mu.Unlock()
				}
			}
		}(regionId, eips)
	}
	wg.Wait()
	return self.sendMetrics(ctx, manager, "elasticip", len(res), metrics)
}
