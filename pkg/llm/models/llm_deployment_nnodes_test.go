package models

import "testing"

func TestValidateDeploymentNnodes(t *testing.T) {
	cases := []struct {
		name             string
		nnodes           int
		gpuCount         int
		singleGPUPerNode bool
		wantErr          bool
	}{
		{"auto", 0, 8, false, false},
		{"single", 1, 8, false, false},
		{"even split", 2, 8, false, false},
		{"indivisible", 3, 8, false, true},
		{"no gpus", 2, 0, false, true},
		{"hami must equal gpu count", 2, 8, true, true},
		{"hami one gpu per node", 8, 8, true, false},
	}
	for _, tc := range cases {
		err := validateDeploymentNnodes(tc.nnodes, tc.gpuCount, tc.singleGPUPerNode)
		if (err != nil) != tc.wantErr {
			t.Fatalf("%s: err = %v, wantErr %v", tc.name, err, tc.wantErr)
		}
	}
}
