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
