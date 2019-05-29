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
	"reflect"
	"testing"
)

func TestNewResourcePendingUsage(t *testing.T) {
	tests := []struct {
		name string
		args map[string]int
		want map[string]int
	}{
		{
			name: "map init",
			args: map[string]int{"s1": 1, "s2": 2},
			want: map[string]int{"s1": 1, "s2": 2},
		},
		{
			name: "map nil init",
			args: nil,
			want: map[string]int{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewResourcePendingUsage(tt.args); !reflect.DeepEqual(got.ToMap(), tt.want) {
				t.Errorf("NewResourcePendingUsage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSResourcePendingUsage_Add(t *testing.T) {
	tests := []struct {
		name string
		ou   *SResourcePendingUsage
		su   *SResourcePendingUsage
		want map[string]int
	}{
		{
			name: "same add",
			ou:   NewResourcePendingUsage(map[string]int{"k1": 1, "k2": 2}),
			su:   NewResourcePendingUsage(map[string]int{"k1": 3, "k2": 1}),
			want: map[string]int{"k1": 4, "k2": 3},
		},
		{
			name: "diff add",
			ou:   NewResourcePendingUsage(map[string]int{"k1": 1, "k2": 2}),
			su:   NewResourcePendingUsage(map[string]int{"k1": 3, "k2": 1, "k3": 4}),
			want: map[string]int{"k1": 4, "k2": 3, "k3": 4},
		},
		{
			name: "add nil",
			ou:   NewResourcePendingUsage(map[string]int{"k1": 1, "k2": 2}),
			su:   NewResourcePendingUsage(nil),
			want: map[string]int{"k1": 1, "k2": 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ou.Add(tt.su)
			if got := tt.ou.ToMap(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewResourcePendingUsage_add() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSResourcePendingUsage_Sub(t *testing.T) {
	tests := []struct {
		name string
		ou   *SResourcePendingUsage
		su   *SResourcePendingUsage
		want map[string]int
	}{
		{
			name: "same sub",
			ou:   NewResourcePendingUsage(map[string]int{"k1": 1, "k2": 2}),
			su:   NewResourcePendingUsage(map[string]int{"k1": 3, "k2": 1}),
			want: map[string]int{"k1": 0, "k2": 1},
		},
		{
			name: "sub nil",
			ou:   NewResourcePendingUsage(map[string]int{"k1": 1, "k2": 2}),
			su:   NewResourcePendingUsage(nil),
			want: map[string]int{"k1": 1, "k2": 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ou.Sub(tt.su)
			if got := tt.ou.ToMap(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewResourcePendingUsage_add() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSResourcePendingUsage_IsEmpty(t *testing.T) {
	tests := []struct {
		name string
		u    *SResourcePendingUsage
		want bool
	}{
		{
			name: "nil is empty",
			u:    NewResourcePendingUsage(nil),
			want: true,
		},
		{
			name: "item zeros is empty",
			u:    NewResourcePendingUsage(map[string]int{"k1": 0, "k2": 0}),
			want: true,
		},
		{
			name: "item no zeros not empty",
			u:    NewResourcePendingUsage(map[string]int{"k1": 1, "k2": 0}),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.u.IsEmpty(); got != tt.want {
				t.Errorf("SResourcePendingUsage.IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}
