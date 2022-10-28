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
	"fmt"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

var driverTable map[string]ICollectDriver = map[string]ICollectDriver{}

type ICollectDriver interface {
	GetProvider() string
	GetDelayDuration() time.Duration
	IsSupportMetrics() bool
	CollectAccountMetrics(ctx context.Context, account api.CloudaccountDetail) (influxdb.SMetricData, error)
	CollectDBInstanceMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.DBInstanceDetails, start, end time.Time) error
	CollectServerMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.ServerDetails, start, end time.Time) error
	CollectHostMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.HostDetails, start, end time.Time) error
	CollectStorageMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.StorageDetails, start, end time.Time) error
	CollectRedisMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.ElasticcacheDetails, start, end time.Time) error
	CollectLoadbalancerMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.LoadbalancerDetails, start, end time.Time) error
	CollectBucketMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.BucketDetails, start, end time.Time) error
	CollectK8sMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.KubeClusterDetails, start, end time.Time) error
	CollectModelartsPoolMetrics(ctx context.Context, manager api.CloudproviderDetails, provider cloudprovider.ICloudProvider, res map[string]api.ModelartsPoolDetails, start, end time.Time) error
}

func GetDriver(name string) (ICollectDriver, error) {
	driver, ok := driverTable[name]
	if !ok {
		return nil, fmt.Errorf("not found %s collect driver", name)
	}
	return driver, nil
}

func Register(driver ICollectDriver) {
	driverTable[driver.GetProvider()] = driver
}
