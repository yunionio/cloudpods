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

package db

import (
	"reflect"
	"testing"
)

func TestISharableMergeChangeOwnerCandidateDomainIds(t *testing.T) {
	cases := []struct {
		candidates [][]string
		want       []string
	}{
		{
			candidates: [][]string{
				nil,
				[]string{"abc"},
				[]string{},
				[]string{"abc", "bcd"},
				[]string{"bcd"},
			},
			want: []string{"123"},
		},
		{
			candidates: [][]string{
				nil,
				[]string{},
			},
			want: nil,
		},
	}
	model := &SInfrasResourceBase{}
	model.DomainId = "123"
	for _, c := range cases {
		got := ISharableMergeChangeOwnerCandidateDomainIds(model, c.candidates...)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("want: %s got %s", c.want, got)
		}
	}
}

func TestISharableMergeShareRequireDomainIds(t *testing.T) {
	cases := []struct {
		requires [][]string
		want     []string
	}{
		{
			requires: [][]string{
				nil,
				[]string{"abc"},
			},
			want: nil,
		},
		{
			requires: [][]string{
				{"abc"},
				{"def"},
				{"abc", "def"},
			},
			want: []string{"abc", "def"},
		},
	}
	for _, c := range cases {
		got := ISharableMergeShareRequireDomainIds(c.requires...)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("want: %s got %s", c.want, got)
		}
	}
}
