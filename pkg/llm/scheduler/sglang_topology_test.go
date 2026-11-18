package scheduler

import (
	"strings"
	"testing"
)

func TestValidateSGLangMultinodeTopologyDefaultsPPToNnodes(t *testing.T) {
	topo, err := ValidateSGLangMultinodeTopology([]int{2, 2, 2}, MultinodeUserParallelism{})
	if err != nil {
		t.Fatal(err)
	}
	if topo.NNodes != 3 || topo.TP != 2 || topo.PP != 3 {
		t.Fatalf("topology = %+v, want nnodes=3 tp=2 pp=3", topo)
	}
}

func TestValidateSGLangMultinodeTopologyRespectsPin(t *testing.T) {
	pp := 2
	topo, err := ValidateSGLangMultinodeTopology([]int{4, 4, 4, 4}, MultinodeUserParallelism{PP: &pp})
	if err != nil {
		t.Fatal(err)
	}
	if topo.PP != 2 {
		t.Fatalf("pp = %d, want 2", topo.PP)
	}
}

func TestValidateSGLangMultinodeTopologyRejectsPinAboveNnodes(t *testing.T) {
	pp := 4
	_, err := ValidateSGLangMultinodeTopology([]int{2, 2, 2}, MultinodeUserParallelism{PP: &pp})
	if err == nil {
		t.Fatal("expected pp > nnodes to be rejected")
	}
	if !strings.Contains(err.Error(), "--pp-size 4 exceeds dist_nnodes 3") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "world size 8 would exceed the 6 GPUs") {
		t.Fatalf("error should name the arithmetic: %v", err)
	}
}

func TestValidateSGLangMultinodeTopologyRejectsSingleNode(t *testing.T) {
	if _, err := ValidateSGLangMultinodeTopology([]int{8}, MultinodeUserParallelism{}); err == nil {
		t.Fatal("expected single-node groups to be rejected")
	}
}
