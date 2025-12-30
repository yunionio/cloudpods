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
	"time"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type AliyunCollect struct {
	SCollectByMetricTypeDriver
}

func (self *AliyunCollect) GetProvider() string {
	return api.CLOUD_PROVIDER_ALIYUN
}

func (self *AliyunCollect) IsSupportMetrics() bool {
	return true
}

func (self *AliyunCollect) GetDelayDuration() time.Duration {
	return time.Minute * 3
}

func init() {
	Register(&AliyunCollect{})
}
