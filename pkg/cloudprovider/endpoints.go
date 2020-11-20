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

package cloudprovider

type SApsaraEndpoints struct {
	EcsEndpoint             string `default:"$APSARA_ECS_ENDPOINT" metavar:"APSARA_ECS_ENDPOINT"`
	RdsEndpoint             string `default:"$APSARA_RDS_ENDPOINT"`
	VpcEndpoint             string `default:"$APSARA_VPC_ENDPOINT"`
	KvsEndpoint             string `default:"$APSARA_KVS_ENDPOINT"`
	SlbEndpoint             string `default:"$APSARA_SLB_ENDPOINT"`
	OssEndpoint             string `default:"$APSARA_OSS_ENDPOINT"`
	StsEndpoint             string `default:"$APSARA_STS_ENDPOINT"`
	ActionTrailEndpoint     string `default:"$APSARA_ACTION_TRAIL_ENDPOINT"`
	RamEndpoint             string `default:"$APSARA_RAM_ENDPOINT"`
	MetricsEndpoint         string `default:"$APSRRA_METRICS_ENDPOINT"`
	ResourcemanagerEndpoint string `default:"$APSARA_RESOURCEMANAGER_ENDPOINT"`
}
