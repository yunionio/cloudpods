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

package handlers

import (
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestUsageRangePreset24h(t *testing.T) {
	now := time.Date(2026, 8, 13, 16, 0, 0, 0, time.UTC)
	start, end, err := usageRange("24h", "", "", now)
	if err != nil {
		t.Fatal(err)
	}
	if !end.Equal(now) {
		t.Fatalf("end = %v, want %v", end, now)
	}
	if !start.Equal(now.Add(-24 * time.Hour)) {
		t.Fatalf("start = %v, want %v", start, now.Add(-24*time.Hour))
	}
}

func TestUsageRangeStartEndWithoutRange(t *testing.T) {
	now := time.Date(2026, 8, 13, 16, 0, 0, 0, time.UTC)
	start, end, err := usageRange("", "2026-08-12T00:00:00Z", "2026-08-13T00:00:00Z", now)
	if err != nil {
		t.Fatal(err)
	}
	wantStart := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC)
	if !start.Equal(wantStart) || !end.Equal(wantEnd) {
		t.Fatalf("got %v %v, want %v %v", start, end, wantStart, wantEnd)
	}
}

func TestUsageRangeCustomRFC3339(t *testing.T) {
	now := time.Date(2026, 8, 13, 16, 0, 0, 0, time.UTC)
	start, end, err := usageRange("custom", "2026-08-12T08:00:00Z", "2026-08-12T10:00:00Z", now)
	if err != nil {
		t.Fatal(err)
	}
	wantStart := time.Date(2026, 8, 12, 8, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	if !start.Equal(wantStart) || !end.Equal(wantEnd) {
		t.Fatalf("got %v %v, want %v %v", start, end, wantStart, wantEnd)
	}
}

func TestUsageRangeCustomMySQLTime(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 8, 13, 16, 0, 0, 0, loc)
	start, end, err := usageRange("custom", "2026-08-12 00:00:00", "2026-08-12 12:00:00", now)
	if err != nil {
		t.Fatal(err)
	}
	wantStart := time.Date(2026, 8, 12, 0, 0, 0, 0, loc)
	wantEnd := time.Date(2026, 8, 12, 12, 0, 0, 0, loc)
	if !start.Equal(wantStart) || !end.Equal(wantEnd) {
		t.Fatalf("got %v %v, want %v %v", start.In(loc), end.In(loc), wantStart, wantEnd)
	}
}

func TestUsageRangeUnixSeconds(t *testing.T) {
	now := time.Date(2026, 8, 13, 16, 0, 0, 0, time.UTC)
	from := now.Add(-time.Hour)
	start, end, err := usageRange("", unixSeconds(from), unixSeconds(now), now)
	if err != nil {
		t.Fatal(err)
	}
	if start.Unix() != from.Unix() || end.Unix() != now.Unix() {
		t.Fatalf("got %v %v, want %v %v", start, end, from, now)
	}
}

func TestUsageRangeMissingEnd(t *testing.T) {
	now := time.Date(2026, 8, 13, 16, 0, 0, 0, time.UTC)
	_, _, err := usageRange("custom", "2026-08-12T00:00:00Z", "", now)
	if err == nil {
		t.Fatal("expected error for missing end")
	}
}

func TestUsageRangeEndNotAfterStart(t *testing.T) {
	now := time.Date(2026, 8, 13, 16, 0, 0, 0, time.UTC)
	_, _, err := usageRange("custom", "2026-08-13T00:00:00Z", "2026-08-13T00:00:00Z", now)
	if err == nil {
		t.Fatal("expected error when end is not after start")
	}
}

func TestUsageRangeLongWindowAllowed(t *testing.T) {
	now := time.Date(2026, 8, 13, 16, 0, 0, 0, time.UTC)
	start, end, err := usageRange("custom", "2026-01-01T00:00:00Z", "2026-02-02T00:00:00Z", now)
	if err != nil {
		t.Fatalf("long window should be allowed: %v", err)
	}
	wantStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC)
	if !start.Equal(wantStart) || !end.Equal(wantEnd) {
		t.Fatalf("got %v %v, want %v %v", start, end, wantStart, wantEnd)
	}
}

func TestParseUsageFilterStartEndWithoutRange(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/?start=2026-08-12T00:00:00Z&end=2026-08-13T00:00:00Z", nil)
	if err != nil {
		t.Fatal(err)
	}
	filter, err := parseUsageFilter(req)
	if err != nil {
		t.Fatal(err)
	}
	if filter.Range != "custom" {
		t.Fatalf("range = %q, want custom", filter.Range)
	}
	wantStart := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC)
	if !filter.Start.Equal(wantStart) || !filter.End.Equal(wantEnd) {
		t.Fatalf("got %v %v, want %v %v", filter.Start, filter.End, wantStart, wantEnd)
	}
}

func unixSeconds(ts time.Time) string {
	return strconv.FormatInt(ts.Unix(), 10)
}
