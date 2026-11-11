// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package db

import (
	"testing"

	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/sqlchemy"
)

func TestShouldSkipOpsLog(t *testing.T) {
	cases := []struct {
		objType string
		action  string
		skip    bool
	}{
		{"dbinstance_sku", ACT_DELETE, true},
		{"dbinstance_sku", ACT_UPDATE, true},
		{"cloudimage", ACT_SET_METADATA, true},
		{"nas_sku", ACT_UPDATE, true},
		{"serversku", ACT_DELETE, true},
		{"serversku", ACT_SET_METADATA, true},
		{"serversku", ACT_CREATE, true},
		{"serversku", ACT_ENABLE, true},
		{"serversku", ACT_UPDATE, false},
		{"cachedimage", ACT_SYNC_UPDATE, true},
		{"cachedimage", ACT_DELETE, true},
		{"cachedimage", ACT_CREATE, false},
		{"cachedimage", ACT_SYNC_OWNER, false},
		{"cloudaccountsynctask", ACT_START, true},
		{"cloudprovider", ACT_SYNC_HOST_COMPLETE, false},
		{"cloudaccount", ACT_SYNC_OWNER, false},
		{"cloudaccount", ACT_SYNC_SHARE, false},
		{"server", ACT_CREATE, false},
		{"server", ACT_UPDATE_STATUS, false},
		{"cloudaccount", ACT_SYNC_UPDATE, false},
		{"disk", ACT_DELETE, false},
	}
	for _, c := range cases {
		got := shouldSkipOpsLog(c.objType, c.action)
		if got != c.skip {
			t.Errorf("shouldSkipOpsLog(%s, %s) = %v, want %v", c.objType, c.action, got, c.skip)
		}
	}
}

func TestFilterSyncUpdateDiffs(t *testing.T) {
	uds := sqlchemy.UpdateDiffs{
		"probe_at": {},
		"balance":  {},
		"name":     {},
	}
	filtered := filterSyncUpdateDiffs(uds)
	if len(filtered) != 1 {
		t.Fatalf("expect 1 remaining field, got %d: %v", len(filtered), filtered)
	}
	if _, ok := filtered["name"]; !ok {
		t.Fatalf("expect name kept, got %v", filtered)
	}

	onlyNoise := sqlchemy.UpdateDiffs{
		"probe_at": {},
		"balance":  {},
	}
	if len(filterSyncUpdateDiffs(onlyNoise)) != 0 {
		t.Fatalf("expect empty after filtering heartbeat-only diffs")
	}
}

func TestCurrentTimestamp(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{
			in:   "2021-12-09 11:01:02",
			want: 20211209110102000,
		},
	}
	for _, c := range cases {
		tm, err := timeutils.ParseTimeStr(c.in)
		if err != nil {
			t.Errorf("parseTime %s error %s", c.in, err)
		} else {
			got := CurrentTimestamp(tm)
			if got != c.want {
				t.Errorf("currentTimestamp %s want %d got %d", c.in, c.want, got)
			}
		}
	}
}
