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

package azure

import (
	"net/url"
)

type SAutoscaleSettingResource struct {
	AzureTags
	region *SRegion

	Properties SAutoscaleSettingProperties
	ID         string
	Location   string
	Name       string
	Type       string
}

type SAutoscaleSettingProperties struct {
	Enabled           bool
	Name              string
	TargetResourceUri string
	Profiles          []SAutoscaleProfile
}

type SAutoscaleProfile struct {
	Capacity   SScaleCapacity
	FixedDate  STimeWindow
	Name       string
	Recurrence SRecurrence
	Rule       []SScaleRule
}

type SRecurrence struct {
	Frequency string
	Schedule  SRecurrentSchedule
}

type SRecurrentSchedule struct {
	Days     []string
	Hours    []int
	Minutes  []int
	TimeZone string
}

type STimeWindow struct {
	// 采用 ISO 8601 格式的配置文件的结束时间。
	End string
	// 采用 ISO 8601 格式的配置文件的开始时间。
	Start    string
	TimeZone string
}

type SScaleAction struct {
	Cooldown  string
	Direction string
	Type      string
	Value     string
}

// 可以在此配置文件期间使用的实例数。
type SScaleCapacity struct {
	Default string
	Maximum string
	Minimum string
}

// Decrease	string
// Increase	string
// None	string

// 为缩放操作提供触发器和参数的规则。
type SScaleRule struct {
	MetricTrigger SMetricTrigger
	ScaleAction   SScaleAction
}

// 维度运算符。 仅支持"Equals"和"NotEquals"。 "Equals"等于任何值。 "NotEquals"不等于所有值
type SScaleRuleMetricDimensionOperationType struct {
	Equals    string
	NotEquals string
}

//触发缩放规则时应发生的操作类型。
// ChangeCount        string
// ExactCount         string
// PercentChangeCount string

type SScaleRuleMetricDimension struct {
	DimensionName string
	Operator      string
	Values        []string
}

type SMetricTrigger struct {
	// 维度条件的列表。 例如： [{"DimensionName"： "AppName"、"Operator"： "Equals"、"Values"： ["App1"]}、{"DimensionName"： "Deployment"、"Operator"： "Equals"、"Values"： ["default"]}]。
	Dimensions []SScaleRuleMetricDimension

	// 一个值，该值指示度量值是否应除以每个实例。
	DividePerInstance bool

	// 定义规则监视内容的指标的名称。
	MetricName string

	// 定义规则监视内容的指标的命名空间。
	MetricNamespace string

	// 规则监视的资源的资源标识符。
	MetricResourceUri string

	// 用于比较指标数据和阈值的运算符。
	// Equals GreaterThan GreaterThanOrEqual LessThan LessThanOrEqual NotEquals
	Operator string

	// 指标统计信息类型。 来自多个实例的指标进行组合的方式。
	// Average Max	Min	Sum
	Statistic string

	// 触发缩放操作的度量值的阈值。
	Threshold int

	// 时间聚合类型。 随着时间推移，收集的数据应如何组合。 默认值为 Average。
	// Average	Count	Last	Maximum	Minimum	Total
	TimeAggregation string

	//规则监视的度量值的粒度。 必须是从指标的指标定义返回的预定义值之一。 必须介于 12 小时和 1 分钟之间。
	TimeGrain string

	//收集实例数据的时间范围。 此值必须大于指标集合中的延迟，可能会因资源而异。 必须介于 12 小时和 5 分钟之间。
	TimeWindow string
}

func (r *SRegion) GetAutoscaleSettingResources() ([]SAutoscaleSettingResource, error) {
	result := []SAutoscaleSettingResource{}
	resource := "microsoft.insights/autoscalesettings"
	err := r.client.list(resource, url.Values{"api-version": []string{"2015-04-01"}}, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
