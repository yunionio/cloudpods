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

package influxdb

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type SKeyValue struct {
	Key   string
	Value string
}

func (kv SKeyValue) String() string {
	return fmt.Sprintf("%s=%s", strings.ReplaceAll(strings.Trim(kv.Key, " "), ",", ""), strings.ReplaceAll(strings.ReplaceAll(kv.Value, " ", "+"), ",", "+"))
}

type TKeyValuePairs []SKeyValue

func (a TKeyValuePairs) Len() int           { return len(a) }
func (a TKeyValuePairs) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a TKeyValuePairs) Less(i, j int) bool { return a[i].Key < a[j].Key }

type SMetricData struct {
	Name      string
	Tags      TKeyValuePairs
	Metrics   TKeyValuePairs
	Timestamp time.Time
}

func (m *SMetricData) Line() string {
	sort.Sort(m.Tags)
	sort.Sort(m.Metrics)

	line := strings.Builder{}
	line.WriteString(m.Name)
	keys := map[string]bool{}
	for i := range m.Tags {
		if _, ok := keys[m.Tags[i].Key]; !ok && len(m.Tags[i].Key) > 0 && len(m.Tags[i].Value) > 0 {
			line.WriteByte(',')
			line.WriteString(m.Tags[i].String())
			keys[m.Tags[i].Key] = true
		}
	}
	line.WriteByte(' ')
	for i := range m.Metrics {
		if i > 0 {
			line.WriteByte(',')
		}
		line.WriteString(m.Metrics[i].String())
	}
	line.WriteByte(' ')
	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now()
	}
	line.WriteString(strconv.FormatInt(m.Timestamp.UnixNano()/1000000, 10))
	return line.String()
}
