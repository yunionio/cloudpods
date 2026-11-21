package scheduler

import (
	"strings"
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestPlanReplicaSingleHostPreferred(t *testing.T) {
	plan, err := PlanReplica(PlanOptions{
		TotalGPUs:            4,
		DeviceTemplate:       api.Device{Model: "A100"},
		DistributedInference: true,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 8},
			{HostId: "h2", Online: true, FreeGPUs: 8},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Distributed || len(plan.Members) != 1 || plan.Members[0].HostId != "h1" {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	if len(plan.Members[0].Devices) != 4 {
		t.Fatalf("devices = %d", len(plan.Members[0].Devices))
	}
}

func TestPlanReplicaDistributedSplit(t *testing.T) {
	plan, err := PlanReplica(PlanOptions{
		TotalGPUs:            8,
		DeviceTemplate:       api.Device{Model: "A100"},
		DistributedInference: true,
		PlacementStrategy:    PlacementSpread,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 4},
			{HostId: "h2", Online: true, FreeGPUs: 4},
			{HostId: "h3", Online: true, FreeGPUs: 2},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Distributed || plan.NNodes != 2 || plan.GpusPerNode != 4 {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	if plan.Members[0].HostId != "h1" || plan.Members[1].HostId != "h2" {
		t.Fatalf("members = %+v", plan.Members)
	}
	// Unpinned multi-node defaults to the official vLLM layout:
	// TP = GPUs per node, PP = node count, DP = 1.
	if got, want := plan.Parallelism, (&PlanParallelism{TP: 4, PP: 2, DP: 1}); got == nil || *got != *want {
		t.Fatalf("parallelism = %+v, want %+v", got, want)
	}
}

func TestPlanReplicaDefaultMultinodeLayoutIsMPOnly(t *testing.T) {
	// 2 nodes x 8 GPUs, nothing pinned: official default TP=8, PP=2, DP=1.
	plan, err := PlanReplica(PlanOptions{
		TotalGPUs:            16,
		DeviceTemplate:       api.Device{Model: "A100"},
		DistributedInference: true,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 8},
			{HostId: "h2", Online: true, FreeGPUs: 8},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.NNodes != 2 || plan.GpusPerNode != 8 {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	if got, want := plan.Parallelism, (&PlanParallelism{TP: 8, PP: 2, DP: 1}); got == nil || *got != *want {
		t.Fatalf("parallelism = %+v, want %+v", got, want)
	}
}

func TestPlanReplicaPinnedDPKeepsDPOnly(t *testing.T) {
	// A user-pinned --data-parallel-size keeps the DP-only derivation.
	plan, err := PlanReplica(PlanOptions{
		TotalGPUs:            8,
		DeviceTemplate:       api.Device{Model: "A100"},
		DistributedInference: true,
		BackendParameters:    []string{"--data-parallel-size 2"},
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 4},
			{HostId: "h2", Online: true, FreeGPUs: 4},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := plan.Parallelism, (&PlanParallelism{TP: 4, PP: 1, DP: 2}); got == nil || *got != *want {
		t.Fatalf("parallelism = %+v, want %+v", got, want)
	}
}

func TestPlanReplicaRequiresDistributedFlag(t *testing.T) {
	_, err := PlanReplica(PlanOptions{
		TotalGPUs: 8,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 4},
			{HostId: "h2", Online: true, FreeGPUs: 4},
		},
	})
	if err == nil {
		t.Fatal("expected error without distributed_inference")
	}
}

func TestPlanReplicaNoHomogeneousCombination(t *testing.T) {
	_, err := PlanReplica(PlanOptions{
		TotalGPUs:            5,
		DistributedInference: true,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 3},
			{HostId: "h2", Online: true, FreeGPUs: 2},
		},
	})
	if err == nil {
		t.Fatal("expected failure for non-divisible layout")
	}
}

func TestPlanReplicaForceSingleGPUPerNode(t *testing.T) {
	plan, err := PlanReplica(PlanOptions{
		TotalGPUs:             2,
		DistributedInference:  true,
		ForceSingleGPUPerNode: true,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 4},
			{HostId: "h2", Online: true, FreeGPUs: 4},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Distributed || plan.GpusPerNode != 1 || plan.NNodes != 2 {
		t.Fatalf("unexpected plan: %+v", plan)
	}
}

func TestPlanReplicaSkipsOfflineHosts(t *testing.T) {
	_, err := PlanReplica(PlanOptions{
		TotalGPUs:            4,
		DeviceTemplate:       api.Device{Model: "A100"},
		DistributedInference: true,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: false, FreeGPUs: 8},
		},
	})
	if err == nil {
		t.Fatal("expected offline-only hosts to be rejected")
	}
}

func TestPlanReplicaSkipsHostsWithInsufficientVram(t *testing.T) {
	_, err := PlanReplica(PlanOptions{
		TotalGPUs:            2,
		DeviceTemplate:       api.Device{Model: "A100", MemoryMb: 40000},
		DistributedInference: true,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 4, MinVRAMMb: 24000},
			{HostId: "h2", Online: true, FreeGPUs: 4, MinVRAMMb: 32000},
		},
	})
	if err == nil {
		t.Fatal("expected hosts below the requested VRAM to be filtered out")
	}
}

func TestPlanReplicaSelectsVramQualifiedHost(t *testing.T) {
	plan, err := PlanReplica(PlanOptions{
		TotalGPUs:            2,
		DeviceTemplate:       api.Device{Model: "A100", MemoryMb: 40000},
		DistributedInference: true,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 4, MinVRAMMb: 24000},
			{HostId: "h2", Online: true, FreeGPUs: 4, MinVRAMMb: 80000},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Members) != 1 || plan.Members[0].HostId != "h2" {
		t.Fatalf("expected h2 (80GB) to be selected, plan=%+v", plan)
	}
}

func TestPlanReplicaRejectsHeterogeneousModel(t *testing.T) {
	_, err := PlanReplica(PlanOptions{
		TotalGPUs:            4,
		DeviceTemplate:       api.Device{Model: "A100"},
		DistributedInference: true,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 2, GpuModel: "A100"},
			{HostId: "h2", Online: true, FreeGPUs: 2, GpuModel: "H100"},
		},
	})
	if err == nil {
		t.Fatal("expected heterogeneous model hosts to be rejected")
	}
}

func TestPlanReplicaHonorsPreferHosts(t *testing.T) {
	plan, err := PlanReplica(PlanOptions{
		TotalGPUs:            4,
		DeviceTemplate:       api.Device{Model: "A100"},
		DistributedInference: true,
		PreferHosts:          []string{"h2"},
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 8},
			{HostId: "h2", Online: true, FreeGPUs: 8},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Members) != 1 || plan.Members[0].HostId != "h2" {
		t.Fatalf("prefer_hosts ignored: %+v", plan)
	}
}

func TestSortHostsPlacementStrategies(t *testing.T) {
	hosts := []HostGPUInventory{
		{HostId: "small", Online: true, FreeGPUs: 2},
		{HostId: "big", Online: true, FreeGPUs: 8},
	}
	spread := append([]HostGPUInventory(nil), hosts...)
	sortHosts(spread, PlacementSpread)
	if spread[0].HostId != "big" {
		t.Fatalf("spread should prefer most free GPUs: %+v", spread)
	}
	binpack := append([]HostGPUInventory(nil), hosts...)
	sortHosts(binpack, PlacementBinpack)
	if binpack[0].HostId != "small" {
		t.Fatalf("binpack should prefer least free GPUs: %+v", binpack)
	}
}

func TestPlanReplicaForceNNodesOverridesSingleHostFit(t *testing.T) {
	plan, err := PlanReplica(PlanOptions{
		TotalGPUs:            8,
		DeviceTemplate:       api.Device{Model: "A100"},
		DistributedInference: true,
		ForceNNodes:          2,
		PlacementStrategy:    PlacementSpread,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 8},
			{HostId: "h2", Online: true, FreeGPUs: 8},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Distributed || plan.NNodes != 2 || plan.GpusPerNode != 4 {
		t.Fatalf("unexpected plan: %+v", plan)
	}
}

func TestPlanReplicaForceNNodesRejectsIndivisibleTotal(t *testing.T) {
	_, err := PlanReplica(PlanOptions{
		TotalGPUs:            8,
		DistributedInference: true,
		ForceNNodes:          3,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 8},
			{HostId: "h2", Online: true, FreeGPUs: 8},
			{HostId: "h3", Online: true, FreeGPUs: 8},
		},
	})
	if err == nil {
		t.Fatal("expected indivisible GPU total to be rejected")
	}
}

func TestPlanReplicaForceNNodesRequiresEnoughHosts(t *testing.T) {
	_, err := PlanReplica(PlanOptions{
		TotalGPUs:            8,
		DistributedInference: true,
		ForceNNodes:          2,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 8},
			{HostId: "h2", Online: true, FreeGPUs: 2},
		},
	})
	if err == nil {
		t.Fatal("expected insufficient hosts to be rejected")
	}
}

func TestPlanReplicaForceSingleNodeNeverSpansHosts(t *testing.T) {
	_, err := PlanReplica(PlanOptions{
		TotalGPUs:            8,
		DistributedInference: true,
		ForceNNodes:          1,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 4},
			{HostId: "h2", Online: true, FreeGPUs: 4},
		},
	})
	if err == nil {
		t.Fatal("expected nnodes=1 to refuse a multi-host fallback")
	}
}

func TestPlanReplicaSingleHostPinnedPP(t *testing.T) {
	plan, err := PlanReplica(PlanOptions{
		TotalGPUs:            8,
		DeviceTemplate:       api.Device{Model: "A100"},
		DistributedInference: true,
		PipelineParallelSize: 4,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 8},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Distributed || plan.Parallelism == nil {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	if got, want := plan.Parallelism, (&PlanParallelism{TP: 2, PP: 4, DP: 1}); *got != *want {
		t.Fatalf("parallelism = %+v, want %+v", got, want)
	}
}

func TestPlanReplicaSingleHostPinnedPPRejected(t *testing.T) {
	_, err := PlanReplica(PlanOptions{
		TotalGPUs:            8,
		DeviceTemplate:       api.Device{Model: "A100"},
		DistributedInference: false,
		PipelineParallelSize: 3,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 8},
		},
	})
	if err == nil {
		t.Fatal("expected non-divisible single-node PP to be rejected")
	}
	if !strings.Contains(err.Error(), "does not divide 8 GPUs") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlanReplicaPinnedPPForcesCrossNodeLayout(t *testing.T) {
	// 16 GPUs, pp=4: gpusPerNode=8 (2 nodes) fails nnodes % nnodesWithinDP,
	// gpusPerNode=4 (4 nodes) gives tp=4, workers=16 spanning 4 nodes (mp_only).
	plan, err := PlanReplica(PlanOptions{
		TotalGPUs:            16,
		DeviceTemplate:       api.Device{Model: "A100"},
		DistributedInference: true,
		PipelineParallelSize: 4,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 8},
			{HostId: "h2", Online: true, FreeGPUs: 8},
			{HostId: "h3", Online: true, FreeGPUs: 4},
			{HostId: "h4", Online: true, FreeGPUs: 4},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Distributed || plan.NNodes != 4 || plan.GpusPerNode != 4 {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	if got, want := plan.Parallelism, (&PlanParallelism{TP: 4, PP: 4, DP: 1}); got == nil || *got != *want {
		t.Fatalf("parallelism = %+v, want %+v", got, want)
	}
}

func TestPlanReplicaSGLangPinnedPPExceedsNnodes(t *testing.T) {
	// 8 GPUs, pp=4: the only candidate with enough hosts (4x2) fails
	// pp <= nnodes, so no layout exists.
	_, err := PlanReplica(PlanOptions{
		TotalGPUs:            8,
		DeviceTemplate:       api.Device{Model: "A100"},
		DistributedInference: true,
		Backend:              api.LLM_MODEL_BACKEND_SGLANG,
		PipelineParallelSize: 4,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 4},
			{HostId: "h2", Online: true, FreeGPUs: 4},
		},
	})
	if err == nil {
		t.Fatal("expected pp > nnodes to be rejected")
	}
	if !strings.Contains(err.Error(), "--pp-size 4 exceeds dist_nnodes 2") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlanReplicaSGLangPinnedPPFitsMultinode(t *testing.T) {
	// 8 GPUs, pp=3: 4 nodes x 2 GPUs, tp=2, world=6 <= 8.
	plan, err := PlanReplica(PlanOptions{
		TotalGPUs:            8,
		DeviceTemplate:       api.Device{Model: "A100"},
		DistributedInference: true,
		Backend:              api.LLM_MODEL_BACKEND_SGLANG,
		PipelineParallelSize: 3,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 2},
			{HostId: "h2", Online: true, FreeGPUs: 2},
			{HostId: "h3", Online: true, FreeGPUs: 2},
			{HostId: "h4", Online: true, FreeGPUs: 2},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Distributed || plan.NNodes != 4 || plan.GpusPerNode != 2 {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	if got, want := plan.Parallelism, (&PlanParallelism{TP: 2, PP: 3, DP: 1}); got == nil || *got != *want {
		t.Fatalf("parallelism = %+v, want %+v", got, want)
	}
}

func TestPlanReplicaDeploymentPPOverridesSkuParam(t *testing.T) {
	// SKU pins pp=4 (invalid for 2x4) but the deployment pin pp=2 wins.
	plan, err := PlanReplica(PlanOptions{
		TotalGPUs:            8,
		DeviceTemplate:       api.Device{Model: "A100"},
		DistributedInference: true,
		BackendParameters:    []string{"--pipeline-parallel-size=4"},
		PipelineParallelSize: 2,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 4},
			{HostId: "h2", Online: true, FreeGPUs: 4},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Parallelism == nil || plan.Parallelism.PP != 2 {
		t.Fatalf("deployment pin must win over SKU params: %+v", plan)
	}
}

func TestPlanReplicaForcedNnodesWithPPConflict(t *testing.T) {
	_, err := PlanReplica(PlanOptions{
		TotalGPUs:            8,
		DeviceTemplate:       api.Device{Model: "A100"},
		DistributedInference: true,
		ForceNNodes:          2,
		PipelineParallelSize: 3,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 4},
			{HostId: "h2", Online: true, FreeGPUs: 4},
		},
	})
	if err == nil {
		t.Fatal("expected forced nnodes + PP conflict to be rejected")
	}
	if !strings.Contains(err.Error(), "nnodes=2") {
		t.Fatalf("error should name the pinned nnodes: %v", err)
	}
}

func TestValidateSingleNodePipelineParallel(t *testing.T) {
	pp := func(v int) *int { return &v }
	cases := []struct {
		name    string
		gpus    int
		pp      *int
		wantErr bool
	}{
		{name: "unset", gpus: 8, pp: nil, wantErr: false},
		{name: "one-is-default", gpus: 8, pp: pp(1), wantErr: false},
		{name: "divides", gpus: 8, pp: pp(4), wantErr: false},
		{name: "not-divisible", gpus: 8, pp: pp(3), wantErr: true},
		{name: "exceeds", gpus: 8, pp: pp(9), wantErr: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateSingleNodePipelineParallel(c.gpus, c.pp)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, c.wantErr)
			}
		})
	}
}

func TestPlanReplicaParallelModeDP(t *testing.T) {
	// 2 nodes x 4 GPUs, parallel_mode=dp: TP=4, PP=1, DP=2.
	plan, err := PlanReplica(PlanOptions{
		TotalGPUs:            8,
		DeviceTemplate:       api.Device{Model: "A100"},
		DistributedInference: true,
		ParallelMode:         api.LLM_DEPLOYMENT_PARALLEL_MODE_DP,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 4},
			{HostId: "h2", Online: true, FreeGPUs: 4},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := plan.Parallelism, (&PlanParallelism{TP: 4, PP: 1, DP: 2}); got == nil || *got != *want {
		t.Fatalf("parallelism = %+v, want %+v", got, want)
	}
}

func TestPlanReplicaParallelModeDPIgnoresDefaultPP(t *testing.T) {
	// DP mode must not emit PP flags even though the default layout would.
	plan, err := PlanReplica(PlanOptions{
		TotalGPUs:            8,
		DeviceTemplate:       api.Device{Model: "A100"},
		DistributedInference: true,
		ParallelMode:         api.LLM_DEPLOYMENT_PARALLEL_MODE_DP,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 4},
			{HostId: "h2", Online: true, FreeGPUs: 4},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Parallelism == nil || plan.Parallelism.PP != 1 {
		t.Fatalf("dp mode must not produce PP: %+v", plan.Parallelism)
	}
}

func TestPlanReplicaParallelModePPRespectsDeploymentPin(t *testing.T) {
	// PP mode + deployment pin pp=2 on 2 nodes x 4 GPUs.
	plan, err := PlanReplica(PlanOptions{
		TotalGPUs:            8,
		DeviceTemplate:       api.Device{Model: "A100"},
		DistributedInference: true,
		ParallelMode:         api.LLM_DEPLOYMENT_PARALLEL_MODE_PP,
		PipelineParallelSize: 2,
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 4},
			{HostId: "h2", Online: true, FreeGPUs: 4},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Parallelism == nil || plan.Parallelism.TP != 4 || plan.Parallelism.PP != 2 || plan.Parallelism.DP != 1 {
		t.Fatalf("expected TP=4/PP=2/DP=1, got %+v", plan.Parallelism)
	}
}

func TestPlanReplicaParallelModeExplicitPinsWin(t *testing.T) {
	// DP mode with explicit --data-parallel-size 1 should stay DP=1 (the pin wins).
	plan, err := PlanReplica(PlanOptions{
		TotalGPUs:            8,
		DeviceTemplate:       api.Device{Model: "A100"},
		DistributedInference: true,
		ParallelMode:         api.LLM_DEPLOYMENT_PARALLEL_MODE_DP,
		BackendParameters:    []string{"--data-parallel-size=1"},
		Hosts: []HostGPUInventory{
			{HostId: "h1", Online: true, FreeGPUs: 4},
			{HostId: "h2", Online: true, FreeGPUs: 4},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Parallelism == nil || plan.Parallelism.DP != 1 {
		t.Fatalf("explicit DP=1 pin must win: %+v", plan.Parallelism)
	}
}
