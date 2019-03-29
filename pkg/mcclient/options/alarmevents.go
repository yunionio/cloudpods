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

package options

type AlarmEventListOptions struct {
	BaseListOptions
	NodeLabels     string `help:"Service tree node labels"`
	MetricName     string `help:"Metric name"`
	HostName       string `help:"Host name"`
	HostIp         string `help:"Host IP address"`
	AlarmLevel     string `help:"Alarm level"`
	AlarmCondition string `help:"Concrete alarm rule"`
	Template       string `help:"Template number of the alarm condition"`
	AckStatus      string `help:"Alarm event ack status"`
}
