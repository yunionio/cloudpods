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

package auth

import (
	"net/http"
	"testing"
	"time"
)

func TestCheckRequestFreshness(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name    string
		req     http.Request
		wantErr bool
	}{
		{
			name:    "v4 recent x-amz-date",
			req:     http.Request{Header: http.Header{"X-Amz-Date": []string{now.UTC().Format("20060102T150405Z")}}},
			wantErr: false,
		},
		{
			name:    "v4 expired x-amz-date",
			req:     http.Request{Header: http.Header{"X-Amz-Date": []string{now.Add(-time.Hour).UTC().Format("20060102T150405Z")}}},
			wantErr: true,
		},
		{
			name:    "v4 future x-amz-date",
			req:     http.Request{Header: http.Header{"X-Amz-Date": []string{now.Add(time.Hour).UTC().Format("20060102T150405Z")}}},
			wantErr: true,
		},
		{
			name:    "v4 malformed x-amz-date",
			req:     http.Request{Header: http.Header{"X-Amz-Date": []string{"not-a-date"}}},
			wantErr: true,
		},
		{
			name:    "v2 recent Date header",
			req:     http.Request{Header: http.Header{"Date": []string{now.UTC().Format(http.TimeFormat)}}},
			wantErr: false,
		},
		{
			name:    "v2 expired Date header",
			req:     http.Request{Header: http.Header{"Date": []string{now.Add(-time.Hour).UTC().Format(http.TimeFormat)}}},
			wantErr: true,
		},
		{
			name:    "missing date",
			req:     http.Request{Header: http.Header{}},
			wantErr: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := checkRequestFreshness(c.req)
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
