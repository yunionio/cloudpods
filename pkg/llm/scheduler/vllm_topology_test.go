package scheduler

import "testing"

func TestValidateMultinodeTopologyDPOnly(t *testing.T) {
	topo, err := ValidateMultinodeTopology([]int{4, 4}, MultinodeUserParallelism{})
	if err != nil {
		t.Fatal(err)
	}
	if topo.Shape != ShapeDPOnly {
		t.Fatalf("shape = %s, want dp_only", topo.Shape)
	}
	if topo.TP != 4 || topo.DP != 2 || topo.PP != 1 {
		t.Fatalf("tp/pp/dp = %d/%d/%d, want 4/1/2", topo.TP, topo.PP, topo.DP)
	}
}

func TestValidateMultinodeTopologyMPOnly(t *testing.T) {
	tp := 8
	topo, err := ValidateMultinodeTopology([]int{4, 4}, MultinodeUserParallelism{TP: &tp})
	if err != nil {
		t.Fatal(err)
	}
	if topo.Shape != ShapeMPOnly {
		t.Fatalf("shape = %s, want mp_only", topo.Shape)
	}
	if topo.TP != 8 || topo.DP != 1 || topo.NNodesWithinDP != 2 {
		t.Fatalf("tp=%d dp=%d nnodes_within_dp=%d", topo.TP, topo.DP, topo.NNodesWithinDP)
	}
}

func TestValidateMultinodeTopologyRejectHeterogeneousCrossNode(t *testing.T) {
	tp := 6
	_, err := ValidateMultinodeTopology([]int{4, 2}, MultinodeUserParallelism{TP: &tp})
	if err == nil {
		t.Fatal("expected error for heterogeneous cross-node TP")
	}
}

func TestCalMultinodeTopologyStartRank(t *testing.T) {
	node, err := CalMultinodeTopology([]int{4, 4}, 1, MultinodeUserParallelism{})
	if err != nil {
		t.Fatal(err)
	}
	if node.StartRank != 1 {
		t.Fatalf("start_rank = %d, want 1", node.StartRank)
	}
	if node.DPL != 1 {
		t.Fatalf("dpl = %d, want 1", node.DPL)
	}
}

func TestParseUserParallelism(t *testing.T) {
	user := ParseUserParallelism([]string{"--tensor-parallel-size=8", "--pipeline-parallel-size 2"})
	if user.TP == nil || *user.TP != 8 {
		t.Fatalf("tp = %v", user.TP)
	}
	if user.PP == nil || *user.PP != 2 {
		t.Fatalf("pp = %v", user.PP)
	}
}

func TestApplyDefaultMultinodeParallelism(t *testing.T) {
	pp := func(v int) *int { return &v }
	cases := []struct {
		name       string
		user       MultinodeUserParallelism
		gpuPerNode []int
		wantPP     int // 0 = nil
	}{
		{name: "unpinned multinode", user: MultinodeUserParallelism{}, gpuPerNode: []int{4, 4}, wantPP: 2},
		{name: "unpinned three nodes", user: MultinodeUserParallelism{}, gpuPerNode: []int{8, 8, 8}, wantPP: 3},
		{name: "pinned pp", user: MultinodeUserParallelism{PP: pp(4)}, gpuPerNode: []int{8, 8}, wantPP: 4},
		{name: "pinned tp", user: MultinodeUserParallelism{TP: pp(8)}, gpuPerNode: []int{8, 8}, wantPP: 0},
		{name: "pinned dp", user: MultinodeUserParallelism{DP: pp(2)}, gpuPerNode: []int{8, 8}, wantPP: 0},
		{name: "pinned pcp", user: MultinodeUserParallelism{PCP: pp(2)}, gpuPerNode: []int{8, 8}, wantPP: 0},
		{name: "single node untouched", user: MultinodeUserParallelism{}, gpuPerNode: []int{8}, wantPP: 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			user := c.user
			ApplyDefaultMultinodeParallelism(&user, c.gpuPerNode)
			if c.wantPP == 0 {
				if user.PP != nil {
					t.Fatalf("pp = %v, want nil", user.PP)
				}
				return
			}
			if user.PP == nil || *user.PP != c.wantPP {
				t.Fatalf("pp = %v, want %d", user.PP, c.wantPP)
			}
		})
	}
}

func TestEffectiveParallelismDeploymentPinWins(t *testing.T) {
	user := EffectiveParallelism([]string{"--pipeline-parallel-size=4"}, 2, "", []int{4, 4})
	if user.PP == nil || *user.PP != 2 {
		t.Fatalf("deployment pin must win over sku params: %+v", user)
	}
}

func TestEffectiveParallelismDPMode(t *testing.T) {
	user := EffectiveParallelism([]string{}, 0, "dp", []int{4, 4})
	if user.DP == nil || *user.DP != 2 {
		t.Fatalf("parallel_mode=dp must set DP to node count: %+v", user)
	}
	if user.PP != nil {
		t.Fatalf("parallel_mode=dp must not set PP: %+v", user)
	}
}
