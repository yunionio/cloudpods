package models

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"yunion.io/x/onecloud/pkg/apis/monitor"
)

func TestBuildTagNameIdValueMap(t *testing.T) {
	cases := []struct {
		name     string
		series   monitor.TimeSeriesSlice
		nameTag  string
		idTag    string
		expected map[string]map[string]string
	}{
		{
			name: "basic vm mapping",
			series: monitor.TimeSeriesSlice{
				{Tags: map[string]string{"vm_name": "vm-a", "vm_id": "uuid-1"}},
				{Tags: map[string]string{"vm_name": "vm-b", "vm_id": "uuid-2"}},
			},
			nameTag: "vm_name",
			idTag:   "vm_id",
			expected: map[string]map[string]string{
				"vm_name": {
					"vm-a": "uuid-1",
					"vm-b": "uuid-2",
				},
			},
		},
		{
			name: "uuid id values are kept",
			series: monitor.TimeSeriesSlice{
				{Tags: map[string]string{"vm_name": "vm-a", "vm_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"}},
			},
			nameTag: "vm_name",
			idTag:   "vm_id",
			expected: map[string]map[string]string{
				"vm_name": {
					"vm-a": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
				},
			},
		},
		{
			name: "infer tags from series when tag keys empty",
			series: monitor.TimeSeriesSlice{
				{Tags: map[string]string{"host": "host-a", "host_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"}},
			},
			nameTag: "",
			idTag:   "",
			expected: map[string]map[string]string{
				"host": {
					"host-a": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
				},
			},
		},
		{
			name: "rename scenario same id different names",
			series: monitor.TimeSeriesSlice{
				{Tags: map[string]string{"vm_name": "old-name", "vm_id": "uuid-1"}},
				{Tags: map[string]string{"vm_name": "new-name", "vm_id": "uuid-1"}},
			},
			nameTag: "vm_name",
			idTag:   "vm_id",
			expected: map[string]map[string]string{
				"vm_name": {
					"old-name": "uuid-1",
					"new-name": "uuid-1",
				},
			},
		},
		{
			name: "conflicting id for same name keeps first",
			series: monitor.TimeSeriesSlice{
				{Tags: map[string]string{"vm_name": "vm-a", "vm_id": "uuid-1"}},
				{Tags: map[string]string{"vm_name": "vm-a", "vm_id": "uuid-2"}},
			},
			nameTag: "vm_name",
			idTag:   "vm_id",
			expected: map[string]map[string]string{
				"vm_name": {
					"vm-a": "uuid-1",
				},
			},
		},
		{
			name: "empty tags",
			series: monitor.TimeSeriesSlice{
				{Tags: map[string]string{"vm_name": "vm-a"}},
			},
			nameTag:  "vm_name",
			idTag:    "vm_id",
			expected: nil,
		},
		{
			name: "infer when only partial tag keys provided",
			series: monitor.TimeSeriesSlice{
				{Tags: map[string]string{"vm_name": "vm-a", "vm_id": "uuid-1"}},
			},
			nameTag: "",
			idTag:   "vm_id",
			expected: map[string]map[string]string{
				"vm_name": {
					"vm-a": "uuid-1",
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildTagNameIdValueMap(tc.series, tc.nameTag, tc.idTag)
			assert.Equal(t, tc.expected, got)
		})
	}
}
