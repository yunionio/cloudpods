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
	"yunion.io/x/onecloud/pkg/cloudcommon/tsdb"
	"yunion.io/x/onecloud/pkg/cloudmon/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

type SBaseCollectDriver struct {
}

func (self *SBaseCollectDriver) GetDelayDuration() time.Duration {
	return 6 * time.Minute
}

func (self *SBaseCollectDriver) IsSupportMetrics() bool {
	return false
}

func (self *SBaseCollectDriver) CollectAccountMetrics(ctx context.Context, account api.CloudaccountDetail) (influxdb.SMetricData, error) {
	return influxdb.SMetricData{}, errors.Wrapf(cloudprovider.ErrNotImplemented, "CollectAccountMetrics")
}

func (self *SBaseCollectDriver) CollectDBInstanceMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.DBInstanceDetails, start, end time.Time) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SBaseCollectDriver) CollectServerMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.ServerDetails, start, end time.Time) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SBaseCollectDriver) CollectHostMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.HostDetails, start, end time.Time) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SBaseCollectDriver) CollectWireMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.WireDetails, start, end time.Time) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SBaseCollectDriver) CollectEipMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.ElasticipDetails, start, end time.Time) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SBaseCollectDriver) CollectStorageMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.StorageDetails, start, end time.Time) error {
	metrics := []influxdb.SMetricData{}
	for _, storage := range res {
		metric := influxdb.SMetricData{
			Name:      string(cloudprovider.METRIC_RESOURCE_TYPE_STORAGE),
			Timestamp: time.Now(),
			Tags:      []influxdb.SKeyValue{},
			Metrics:   []influxdb.SKeyValue{},
		}
		for k, v := range storage.GetMetricTags() {
			metric.Tags = append(metric.Tags, influxdb.SKeyValue{
				Key:   k,
				Value: v,
			})
		}
		for k, v := range storage.GetMetricPairs() {
			metric.Metrics = append(metric.Metrics, influxdb.SKeyValue{
				Key:   k,
				Value: v,
			})
		}
		metrics = append(metrics, metric)
	}
	return self.sendMetrics(ctx, manager, "storage", len(res), metrics)
}

func (self *SBaseCollectDriver) sendMetrics(ctx context.Context, manager api.CloudproviderDetails, resName string, resCnt int, metrics []influxdb.SMetricData) error {
	s := auth.GetAdminSession(ctx, options.Options.Region)
	urls, err := tsdb.GetDefaultServiceSourceURLs(s, options.Options.SessionEndpointType)
	if err != nil {
		return errors.Wrap(err, "GetServiceURLs")
	}
	log.Infof("send %d %s with %d metrics for %s(%s)", resCnt, resName, len(metrics), manager.Name, manager.Id)
	return influxdb.BatchSendMetrics(urls, options.Options.InfluxDatabase, metrics, false)
}

func (self *SBaseCollectDriver) CollectRedisMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.ElasticcacheDetails, start, end time.Time) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SBaseCollectDriver) CollectLoadbalancerMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.LoadbalancerDetails, start, end time.Time) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SBaseCollectDriver) CollectBucketMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.BucketDetails, start, end time.Time) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SBaseCollectDriver) CollectK8sMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.KubeClusterDetails, start, end time.Time) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SBaseCollectDriver) CollectModelartsPoolMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.ModelartsPoolDetails, start, end time.Time) error {
	return cloudprovider.ErrNotImplemented
}

type SCollectByResourceIdDriver struct {
	SBaseCollectDriver
}

func (self *SCollectByResourceIdDriver) CollectDBInstanceMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.DBInstanceDetails, start, end time.Time) error {
	ch := make(chan struct{}, options.Options.CloudResourceCollectMetricsBatchCount)
	defer close(ch)
	metrics := []influxdb.SMetricData{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i := range res {
		ch <- struct{}{}
		wg.Add(1)
		go func(rds api.DBInstanceDetails) {
			defer func() {
				wg.Done()
				<-ch
			}()
			opts := &cloudprovider.MetricListOptions{
				ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_RDS,
				StartTime:    start,
				EndTime:      end,
			}
			opts.ResourceId = rds.ExternalId
			opts.RegionExtId = rds.RegionExtId
			opts.Engine = rds.Engine

			tags := []influxdb.SKeyValue{}
			for k, v := range rds.GetMetricTags() {
				tags = append(tags, influxdb.SKeyValue{
					Key:   k,
					Value: v,
				})
			}
			data, err := provider.GetMetrics(opts)
			if err != nil {
				if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					log.Errorf("get rds %s(%s) error: %v", rds.Name, rds.Id, err)
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

	return self.sendMetrics(ctx, manager, "rds", len(res), metrics)
}

func (self *SCollectByResourceIdDriver) CollectServerMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.ServerDetails, start, end time.Time) error {
	cnt := options.Options.CloudResourceCollectMetricsBatchCount
	if manager.Provider == api.CLOUD_PROVIDER_ORACLE { // oracle 限速
		cnt = options.Options.OracleCloudResourceCollectMetricsBatchCount
	}
	ch := make(chan struct{}, cnt)
	defer close(ch)
	metrics := []influxdb.SMetricData{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i := range res {
		ch <- struct{}{}
		wg.Add(1)
		go func(vm api.ServerDetails) {
			defer func() {
				wg.Done()
				<-ch
			}()
			opts := &cloudprovider.MetricListOptions{
				ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_SERVER,
				RegionExtId:  vm.RegionExtId,
				StartTime:    start,
				EndTime:      end,
				OsType:       vm.OsType,

				IsSupportAzureTableStorageMetric: options.Options.SupportAzureTableStorageMetric,
			}
			opts.ResourceId = vm.ExternalId

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
			data, err := provider.GetMetrics(opts)
			if err != nil {
				if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					log.Errorf("get server %s(%s) error: %v", vm.Name, vm.Id, err)
					return
				}
				return
			}
			for _, values := range data {
				metricKey := values.MetricType.Key()
				for _, value := range values.Values {
					metric := influxdb.SMetricData{
						Name:      values.MetricType.Name(),
						Timestamp: value.Timestamp,
						Tags:      []influxdb.SKeyValue{},
						Metrics: []influxdb.SKeyValue{
							{
								Key:   metricKey,
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
					metric.Tags = append(metric.Tags, tags...)
					metric.Metrics = append(metric.Metrics, pairs...)
					mu.Lock()
					metrics = append(metrics, metric)
					mu.Unlock()
				}
			}
		}(res[i])
	}
	wg.Wait()

	return self.sendMetrics(ctx, manager, "server", len(res), metrics)
}

func (self *SCollectByResourceIdDriver) CollectHostMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.HostDetails, start, end time.Time) error {
	ch := make(chan struct{}, options.Options.CloudResourceCollectMetricsBatchCount)
	defer close(ch)
	metrics := []influxdb.SMetricData{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i := range res {
		ch <- struct{}{}
		wg.Add(1)
		go func(host api.HostDetails) {
			defer func() {
				wg.Done()
				<-ch
			}()
			opts := &cloudprovider.MetricListOptions{
				ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_HOST,
				ResourceId:   host.ExternalId,
				RegionExtId:  host.RegionExtId,
				StartTime:    start,
				EndTime:      end,
			}

			tags := []influxdb.SKeyValue{}
			for k, v := range host.GetMetricTags() {
				tags = append(tags, influxdb.SKeyValue{
					Key:   k,
					Value: v,
				})
			}
			pairs := []influxdb.SKeyValue{}
			for k, v := range host.GetMetricPairs() {
				pairs = append(pairs, influxdb.SKeyValue{
					Key:   k,
					Value: v,
				})
			}
			data, err := provider.GetMetrics(opts)
			if err != nil {
				if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					log.Errorf("get host %s(%s) error: %v", host.Name, host.Id, err)
					return
				}
				return
			}
			for _, values := range data {
				for _, value := range values.Values {
					metric := influxdb.SMetricData{
						Name:      values.MetricType.Name(),
						Timestamp: value.Timestamp,
						Tags:      []influxdb.SKeyValue{},
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
					metric.Tags = append(metric.Tags, tags...)
					metric.Metrics = append(metric.Metrics, pairs...)
					mu.Lock()
					metrics = append(metrics, metric)
					mu.Unlock()
				}
			}
		}(res[i])
	}
	wg.Wait()

	return self.sendMetrics(ctx, manager, "host", len(res), metrics)
}

func (self *SCollectByResourceIdDriver) CollectRedisMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.ElasticcacheDetails, start, end time.Time) error {
	ch := make(chan struct{}, options.Options.CloudResourceCollectMetricsBatchCount)
	defer close(ch)
	metrics := []influxdb.SMetricData{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i := range res {
		ch <- struct{}{}
		wg.Add(1)
		go func(vm api.ElasticcacheDetails) {
			defer func() {
				wg.Done()
				<-ch
			}()
			opts := &cloudprovider.MetricListOptions{
				ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_REDIS,
				RegionExtId:  vm.RegionExtId,
				StartTime:    start,
				EndTime:      end,
			}
			opts.ResourceId = vm.ExternalId

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
			data, err := provider.GetMetrics(opts)
			if err != nil {
				if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					log.Errorf("get %s %s(%s) error: %v", opts.ResourceType, vm.Name, vm.Id, err)
					return
				}
				return
			}
			for _, values := range data {
				for _, value := range values.Values {
					metric := influxdb.SMetricData{
						Name:      values.MetricType.Name(),
						Timestamp: value.Timestamp,
						Tags:      []influxdb.SKeyValue{},
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
					metric.Tags = append(metric.Tags, tags...)
					metric.Metrics = append(metric.Metrics, pairs...)
					mu.Lock()
					metrics = append(metrics, metric)
					mu.Unlock()
				}
			}
		}(res[i])
	}
	wg.Wait()

	return self.sendMetrics(ctx, manager, "redis", len(res), metrics)
}

func (self *SCollectByResourceIdDriver) CollectBucketMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.BucketDetails, start, end time.Time) error {
	ch := make(chan struct{}, options.Options.CloudResourceCollectMetricsBatchCount)
	defer close(ch)
	metrics := []influxdb.SMetricData{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i := range res {
		ch <- struct{}{}
		wg.Add(1)
		go func(vm api.BucketDetails) {
			defer func() {
				wg.Done()
				<-ch
			}()
			opts := &cloudprovider.MetricListOptions{
				ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_BUCKET,
				RegionExtId:  vm.RegionExtId,
				StartTime:    start,
				EndTime:      end,
			}
			opts.ResourceId = vm.ExternalId

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
			data, err := provider.GetMetrics(opts)
			if err != nil {
				if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					log.Errorf("get %s %s(%s) error: %v", opts.ResourceType, vm.Name, vm.Id, err)
					return
				}
				return
			}
			for _, values := range data {
				for _, value := range values.Values {
					metric := influxdb.SMetricData{
						Name:      values.MetricType.Name(),
						Timestamp: value.Timestamp,
						Tags:      []influxdb.SKeyValue{},
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
					metric.Metrics = append(metric.Metrics, pairs...)
					mu.Lock()
					metrics = append(metrics, metric)
					mu.Unlock()
				}
			}
		}(res[i])
	}
	wg.Wait()

	return self.sendMetrics(ctx, manager, "bucket", len(res), metrics)
}

func (self *SCollectByResourceIdDriver) CollectK8sMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.KubeClusterDetails, start, end time.Time) error {
	ch := make(chan struct{}, options.Options.CloudResourceCollectMetricsBatchCount)
	defer close(ch)
	metrics := []influxdb.SMetricData{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i := range res {
		ch <- struct{}{}
		wg.Add(1)
		go func(vm api.KubeClusterDetails) {
			defer func() {
				wg.Done()
				<-ch
			}()
			// 未同步到本地k8s集群
			if len(vm.ExternalClusterId) == 0 {
				log.Infof("skip collect %s %s(%s) metric, because not with local kubeserver", vm.Name, manager.Name, manager.Id)
				return
			}
			opts := &cloudprovider.MetricListOptions{
				ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_K8S,
				ResourceId:   vm.ExternalId,
				RegionExtId:  vm.RegionExtId,
				StartTime:    start,
				EndTime:      end,
			}
			data, err := provider.GetMetrics(opts)
			if err != nil {
				if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					log.Errorf("get %s %s(%s) error: %v", opts.ResourceType, vm.Name, vm.Id, err)
					return
				}
				return
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
			for _, values := range data {
				for _, value := range values.Values {
					metric := influxdb.SMetricData{
						Name:      values.MetricType.Name(),
						Timestamp: value.Timestamp,
						Tags:      []influxdb.SKeyValue{},
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
					metric.Metrics = append(metric.Metrics, pairs...)
					mu.Lock()
					metrics = append(metrics, metric)
					mu.Unlock()
				}
			}
		}(res[i])
	}
	wg.Wait()

	return self.sendMetrics(ctx, manager, "k8s", len(res), metrics)
}

func (self *SCollectByResourceIdDriver) CollectLoadbalancerMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.LoadbalancerDetails, start, end time.Time) error {
	metrics := []influxdb.SMetricData{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i := range res {
		wg.Add(1)
		go func(lb api.LoadbalancerDetails) {
			defer func() {
				wg.Done()
			}()
			opts := &cloudprovider.MetricListOptions{
				ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_LB,
				StartTime:    start,
				EndTime:      end,
			}
			opts.ResourceId = lb.ExternalId
			opts.RegionExtId = lb.RegionExtId

			tags := []influxdb.SKeyValue{}
			for k, v := range lb.GetMetricTags() {
				tags = append(tags, influxdb.SKeyValue{
					Key:   k,
					Value: v,
				})
			}

			data, err := provider.GetMetrics(opts)
			if err != nil {
				if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					log.Errorf("get loadbalancers %s(%s) error: %v", lb.Name, lb.Id, err)
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

	return self.sendMetrics(ctx, manager, "slb", len(res), metrics)
}

type SCollectByMetricTypeDriver struct {
	SBaseCollectDriver
}

func (self *SCollectByMetricTypeDriver) CollectDBInstanceMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.DBInstanceDetails, start, end time.Time) error {
	metrics := []influxdb.SMetricData{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, _metricType := range cloudprovider.ALL_RDS_METRIC_TYPES {
		wg.Add(1)
		go func(metricType cloudprovider.TMetricType) {
			defer func() {
				wg.Done()
			}()
			opts := &cloudprovider.MetricListOptions{
				ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_RDS,
				MetricType:   metricType,
				StartTime:    start,
				EndTime:      end,
			}
			data, err := provider.GetMetrics(opts)
			if err != nil {
				if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					log.Errorf("get rds %s(%s) %s error: %v", manager.Name, manager.Id, metricType, err)
					return
				}
				return
			}
			for _, value := range data {
				rds, ok := res[value.Id]
				if !ok {
					continue
				}
				tags := []influxdb.SKeyValue{}
				for k, v := range rds.GetMetricTags() {
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

	return self.sendMetrics(ctx, manager, "rds", len(res), metrics)
}

func (self *SCollectByMetricTypeDriver) CollectServerMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.ServerDetails, start, end time.Time) error {
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

func (self *SCollectByMetricTypeDriver) CollectHostMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.HostDetails, start, end time.Time) error {
	metrics := []influxdb.SMetricData{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, _metricType := range cloudprovider.ALL_HOST_METRIC_TYPES {
		wg.Add(1)
		go func(metricType cloudprovider.TMetricType) {
			defer func() {
				wg.Done()
			}()
			opts := &cloudprovider.MetricListOptions{
				ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_HOST,
				MetricType:   metricType,

				StartTime: start,
				EndTime:   end,
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

	return self.sendMetrics(ctx, manager, "host", len(res), metrics)
}

func (self *SCollectByMetricTypeDriver) CollectRedisMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.ElasticcacheDetails, start, end time.Time) error {
	metrics := []influxdb.SMetricData{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, _metricType := range cloudprovider.ALL_REDIS_METRIC_TYPES {
		wg.Add(1)
		go func(metricType cloudprovider.TMetricType) {
			defer func() {
				wg.Done()
			}()
			opts := &cloudprovider.MetricListOptions{
				ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_REDIS,
				MetricType:   metricType,
				StartTime:    start,
				EndTime:      end,
			}

			data, err := provider.GetMetrics(opts)
			if err != nil {
				if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					log.Errorf("get metric %s for %s(%s) error: %v", opts.MetricType, manager.Name, manager.Id, err)
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

	return self.sendMetrics(ctx, manager, "redis", len(res), metrics)
}

func (self *SCollectByMetricTypeDriver) CollectBucketMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.BucketDetails, start, end time.Time) error {
	metrics := []influxdb.SMetricData{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, _metricType := range cloudprovider.ALL_BUCKET_TYPES {
		wg.Add(1)
		go func(metricType cloudprovider.TMetricType) {
			defer func() {
				wg.Done()
			}()
			opts := &cloudprovider.MetricListOptions{
				ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_BUCKET,
				MetricType:   metricType,
				StartTime:    start,
				EndTime:      end,
			}

			data, err := provider.GetMetrics(opts)
			if err != nil {
				if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					log.Errorf("get metric %s for %s(%s) error: %v", opts.MetricType, manager.Name, manager.Id, err)
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

	return self.sendMetrics(ctx, manager, "bucket", len(res), metrics)
}

func (self *SCollectByMetricTypeDriver) CollectK8sMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.KubeClusterDetails, start, end time.Time) error {
	metrics := []influxdb.SMetricData{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, _metricType := range cloudprovider.ALL_K8S_NODE_TYPES {
		wg.Add(1)
		go func(metricType cloudprovider.TMetricType) {
			defer func() {
				wg.Done()
			}()
			opts := &cloudprovider.MetricListOptions{
				ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_K8S,
				MetricType:   metricType,
				StartTime:    start,
				EndTime:      end,
			}
			data, err := provider.GetMetrics(opts)
			if err != nil {
				if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					log.Errorf("get rds %s(%s) %s error: %v", manager.Name, manager.Id, metricType, err)
					return
				}
				return
			}
			for _, value := range data {
				k8s, ok := res[value.Id]
				if !ok {
					continue
				}
				tags := []influxdb.SKeyValue{}
				for k, v := range k8s.GetMetricTags() {
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
						metric.Tags = append([]influxdb.SKeyValue{
							{
								Key:   k,
								Value: v,
							},
						}, tags...)
					}
					mu.Lock()
					metrics = append(metrics, metric)
					mu.Unlock()
				}
			}
		}(_metricType)
	}
	wg.Wait()

	return self.sendMetrics(ctx, manager, "k8s", len(res), metrics)
}

func (driver *SCollectByMetricTypeDriver) CollectLoadbalancerMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.LoadbalancerDetails, start, end time.Time) error {
	metrics := []influxdb.SMetricData{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, _metricType := range cloudprovider.ALL_LB_METRIC_TYPES {
		wg.Add(1)
		go func(metricType cloudprovider.TMetricType) {
			defer func() {
				wg.Done()
			}()
			opts := &cloudprovider.MetricListOptions{
				ResourceType: cloudprovider.METRIC_RESOURCE_TYPE_LB,
				MetricType:   metricType,
				StartTime:    start,
				EndTime:      end,
			}
			data, err := provider.GetMetrics(opts)
			if err != nil {
				if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					log.Errorf("get slb %s(%s) %s error: %v", manager.Name, manager.Id, metricType, err)
					return
				}
				return
			}
			for _, value := range data {
				slb, ok := res[value.Id]
				if !ok {
					continue
				}
				tags := []influxdb.SKeyValue{}
				for k, v := range slb.GetMetricTags() {
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

	return driver.sendMetrics(ctx, manager, "slb", len(res), metrics)
}
