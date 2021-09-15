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
	"fmt"
	"net/url"
	"time"
)

type ResponseMetirc struct {
	// Cost - The integer value representing the cost of the query, for data case.
	Cost *float64 `json:"cost,omitempty"`
	// Timespan - The timespan for which the data was retrieved. Its value consists of two datetimes concatenated, separated by '/'.  This may be adjusted in the future and returned back from what was originally requested.
	Timespan *string `json:"timespan,omitempty"`
	// Interval - The interval (window size) for which the metric data was returned in.  This may be adjusted in the future and returned back from what was originally requested.  This is not present if a metadata request was made.
	Interval *string `json:"interval,omitempty"`
	// Namespace - The namespace of the metrics been queried
	Namespace *string `json:"namespace,omitempty"`
	// Resourceregion - The region of the resource been queried for metrics.
	Resourceregion *string `json:"resourceregion,omitempty"`
	// Value - the value of the collection.
	Value *[]Metric `json:"value,omitempty"`
}

// Metric the result data of a query.
type Metric struct {
	// ID - the metric Id.
	ID *string `json:"id,omitempty"`
	// Type - the resource type of the metric resource.
	Type *string `json:"type,omitempty"`
	// Name - the name and the display name of the metric, i.e. it is localizable string.
	Name *LocalizableString `json:"name,omitempty"`
	// Unit - the unit of the metric. Possible values include: 'UnitCount', 'UnitBytes', 'UnitSeconds', 'UnitCountPerSecond', 'UnitBytesPerSecond', 'UnitPercent', 'UnitMilliSeconds', 'UnitByteSeconds', 'UnitUnspecified', 'UnitCores', 'UnitMilliCores', 'UnitNanoCores', 'UnitBitsPerSecond'
	Unit string `json:"unit,omitempty"`
	// Timeseries - the time series returned when a data query is performed.
	Timeseries *[]TimeSeriesElement `json:"timeseries,omitempty"`
}

type TimeSeriesElement struct {
	// Metadatavalues - the metadata values returned if $filter was specified in the call.
	Metadatavalues *[]MetadataValue `json:"metadatavalues,omitempty"`
	// Data - An array of data points representing the metric values.  This is only returned if a result type of data is specified.
	Data *[]MetricValue `json:"data,omitempty"`
}

type MetadataValue struct {
	// Name - the name of the metadata.
	Name *LocalizableString `json:"name,omitempty"`
	// Value - the value of the metadata.
	Value *string `json:"value,omitempty"`
}

type LocalizableString struct {
	// Value - the invariant value.
	Value *string `json:"value,omitempty"`
	// LocalizedValue - the locale specific value.
	LocalizedValue *string `json:"localizedValue,omitempty"`
}

type MetricValue struct {
	// TimeStamp - the timestamp for the metric value in ISO 8601 format.
	TimeStamp *time.Time `json:"timeStamp,omitempty"`
	// Average - the average value in the time range.
	Average *float64 `json:"average,omitempty"`
	// Minimum - the least value in the time range.
	Minimum *float64 `json:"minimum,omitempty"`
	// Maximum - the greatest value in the time range.
	Maximum *float64 `json:"maximum,omitempty"`
	// Total - the sum of all of the values in the time range.
	Total *float64 `json:"total,omitempty"`
	// Count - the number of samples in the time range. Can be used to determine the number of values that contributed to the average value.
	Count *float64 `json:"count,omitempty"`
}

func (self *SRegion) GetMonitorData(name string, ns string, resourceId string, since time.Time, until time.Time, interval string) (*ResponseMetirc, error) {
	params := url.Values{}
	params.Set("metricnamespace", ns)
	params.Set("metricnames", name)
	params.Set("interval", "PT1M")
	if len(interval) != 0 {
		params.Set("interval", interval)
	}
	params.Set("aggregation", "Average")
	params.Set("api-version", "2018-01-01")
	if !since.IsZero() && !until.IsZero() {
		params.Set("timespan", since.UTC().Format(time.RFC3339)+"/"+until.UTC().Format(time.RFC3339))
	}
	resource := fmt.Sprintf("%s/providers/microsoft.insights/metrics", resourceId)
	elements := ResponseMetirc{}
	err := self.get(resource, params, &elements)
	if err != nil {
		return nil, err
	}
	return &elements, nil
}
