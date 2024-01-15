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

package models

import (
	"sort"
	"testing"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

func TestSortDnsResults(t *testing.T) {
	results := []api.SDnsResolveResult{
		{
			DnsName: "*.example.com",
		},
		{
			DnsName: "abc.example.com",
		},
		{
			DnsName: "efg.example.com",
		},
		{
			DnsName: "abc.example.com",
		},
	}
	sort.Sort(sDnsResolveResults(results))
	t.Logf("results: %s", jsonutils.Marshal(results))
}
