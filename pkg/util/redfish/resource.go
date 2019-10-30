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

package redfish

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/onecloud/pkg/util/influxdb"
)

type SCdromInfo struct {
	Image         string `json:"Image"`
	SupportAction bool   `json:"SupportAction"`
}

type SSystemInfo struct {
	Manufacturer string   `json:"Manufacturer"`
	Model        string   `json:"Model"`
	SKU          string   `json:"SKU"`
	SerialNumber string   `json:"SerialNumber"`
	UUID         string   `json:"UUID"`
	EthernetNICs []string `json:"EthernetNICs"`
	MemoryGB     int      `json:"MemoryGB"`
	NodeCount    int      `json:"NodeCount"`
	CpuDesc      string   `json:"CpuDesc"`
	PowerState   string   `json:"PowerState"`
	NextBootDev  string   `json:"NextBootDev"`

	NextBootDevSupported []string `json:"NextBootDevSupported"`
	ResetTypeSupported   []string `json:"ResetTypeSupported"`
}

const (
	EVENT_TYPE_SYSTEM  = "system"
	EVENT_TYPE_MANAGER = "manager"
)

type SEvent struct {
	Created  time.Time `json:"Created"`
	EventId  string    `json:"Id"`
	Message  string    `json:"Message"`
	Severity string    `json:"Severity"`
	Type     string    `json:"type"`
}

type SEventList []SEvent

func (el SEventList) Len() int           { return len(el) }
func (el SEventList) Swap(i, j int)      { el[i], el[j] = el[j], el[i] }
func (el SEventList) Less(i, j int) bool { return el[i].Created.Before(el[j].Created) }

func SortEvents(el []SEvent) {
	sort.Sort(SEventList(el))
}

type SBiosInfo struct {
}

type SPower struct {
	PowerCapacityWatts int `json:"PowerCapacityWatts"`
	PowerConsumedWatts int `json:"PowerConsumedWatts"`
	PowerMetrics       struct {
		AverageConsumedWatts int `json:"AverageConsumedWatts"`
		IntervalInMin        int `json:"IntervalInMin"`
		MaxConsumedWatts     int `json:"MaxConsumedWatts"`
		MinConsumedWatts     int `json:"MinConsumedWatts"`
	} `json:"PowerMetrics"`
}

func (p SPower) ToMetrics() []influxdb.SKeyValue {
	return []influxdb.SKeyValue{
		{
			Key:   "PowerCapacityWatts",
			Value: strconv.FormatInt(int64(p.PowerCapacityWatts), 10),
		},
		{
			Key:   "PowerConsumedWatts",
			Value: strconv.FormatInt(int64(p.PowerConsumedWatts), 10),
		},
		{
			Key:   "AverageConsumedWatts",
			Value: strconv.FormatInt(int64(p.PowerMetrics.AverageConsumedWatts), 10),
		},
		{
			Key:   "IntervalInMin",
			Value: strconv.FormatInt(int64(p.PowerMetrics.IntervalInMin), 10),
		},
		{
			Key:   "MaxConsumedWatts",
			Value: strconv.FormatInt(int64(p.PowerMetrics.MaxConsumedWatts), 10),
		},
		{
			Key:   "MinConsumedWatts",
			Value: strconv.FormatInt(int64(p.PowerMetrics.MinConsumedWatts), 10),
		},
	}
}

type STemperature struct {
	Name            string `json:"Name"`
	PhysicalContext string `json:"PhysicalContext"`
	ReadingCelsius  int    `json:"ReadingCelsius"`
}

func (t STemperature) ToMetric() influxdb.SKeyValue {
	return influxdb.SKeyValue{
		Key:   fmt.Sprintf("%s/%s", t.PhysicalContext, strings.ReplaceAll(t.Name, " ", "")),
		Value: strconv.FormatInt(int64(t.ReadingCelsius), 10),
	}
}

type SNTPConf struct {
	NTPServers      []string `json:"NTPServers,allowempty"`
	ProtocolEnabled bool     `json:"ProtocolEnabled,allowfalse"`
	TimeZone        string   `json:"TimeZone"`
}
