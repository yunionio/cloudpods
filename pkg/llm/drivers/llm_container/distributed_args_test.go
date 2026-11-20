package llm_container

import (
	"strings"
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/models"
)

func fourA100Devices() *api.Devices {
	return &api.Devices{
		{Model: "A100"}, {Model: "A100"}, {Model: "A100"}, {Model: "A100"},
	}
}

func TestBuildVLLMDistributedFlagsMPOnlyFollower(t *testing.T) {
	sku := &models.SLLMSku{
		SLLMSkuBase: models.SLLMSkuBase{
			Devices: fourA100Devices(),
		},
	}
	llm := &models.SLLM{
		DistRole:       api.LLM_DIST_ROLE_FOLLOWER,
		DistRank:       1,
		DistNnodes:     2,
		DistMasterAddr: "10.0.0.1",
	}
	llm.Devices = fourA100Devices()
	tp, flags, _, err := buildVLLMDistributedFlags(llm, sku, []string{"--tensor-parallel-size=8"})
	if err != nil {
		t.Fatalf("build flags: %v", err)
	}
	if tp != 8 {
		t.Fatalf("tp = %d, want 8", tp)
	}
	joined := strings.Join(flags, " ")
	if !strings.Contains(joined, "--headless") {
		t.Fatalf("follower flags missing --headless: %v", flags)
	}
	if !strings.Contains(joined, "--nnodes 2") || !strings.Contains(joined, "--node-rank 1") {
		t.Fatalf("flags = %v", flags)
	}
	if !strings.Contains(joined, "10.0.0.1") {
		t.Fatalf("missing master addr: %v", flags)
	}
}

func TestBuildVLLMDistributedFlagsLeaderNoHeadless(t *testing.T) {
	sku := &models.SLLMSku{
		SLLMSkuBase: models.SLLMSkuBase{
			Devices: fourA100Devices(),
		},
	}
	llm := &models.SLLM{
		DistRole:   api.LLM_DIST_ROLE_LEADER,
		DistRank:   0,
		DistNnodes: 2,
	}
	llm.Devices = fourA100Devices()
	_, flags, _, err := buildVLLMDistributedFlags(llm, sku, nil)
	if err != nil {
		t.Fatalf("build flags: %v", err)
	}
	joined := strings.Join(flags, " ")
	if strings.Contains(joined, "--headless") {
		t.Fatalf("leader must not be headless: %v", flags)
	}
	// Unpinned multi-node defaults to the official vLLM layout:
	// TP = GPUs per node, PP = node count, DP = 1 (mp_only).
	if !strings.Contains(joined, "--pipeline-parallel-size 2") || !strings.Contains(joined, "--nnodes 2") {
		t.Fatalf("expected default mp_only flags: %v", flags)
	}
	if strings.Contains(joined, "--data-parallel-size") {
		t.Fatalf("unpinned default must not emit DP flags: %v", flags)
	}
}

func TestBuildVLLMDistributedFlagsEmitsPCP(t *testing.T) {
	sku := &models.SLLMSku{SLLMSkuBase: models.SLLMSkuBase{Devices: fourA100Devices()}}
	llm := &models.SLLM{DistRole: api.LLM_DIST_ROLE_LEADER, DistRank: 0, DistNnodes: 2}
	llm.Devices = fourA100Devices()
	_, flags, _, err := buildVLLMDistributedFlags(llm, sku, []string{"--prefill-context-parallel-size=2"})
	if err != nil {
		t.Fatalf("build flags: %v", err)
	}
	joined := strings.Join(flags, " ")
	if !strings.Contains(joined, "--prefill-context-parallel-size 2") {
		t.Fatalf("flags missing PCP: %v", flags)
	}
}

func TestBuildVLLMDistributedFlagsNestedStartRank(t *testing.T) {
	// 4 nodes x 4 GPUs, TP=8 => each DP rank spans 2 nodes, node 2 starts DP rank 1.
	sku := &models.SLLMSku{SLLMSkuBase: models.SLLMSkuBase{Devices: fourA100Devices()}}
	llm := &models.SLLM{
		DistRole:   api.LLM_DIST_ROLE_FOLLOWER,
		DistRank:   2,
		DistNnodes: 4,
	}
	llm.Devices = fourA100Devices()
	_, flags, _, err := buildVLLMDistributedFlags(llm, sku, []string{"--tensor-parallel-size=8"})
	if err != nil {
		t.Fatalf("build flags: %v", err)
	}
	joined := strings.Join(flags, " ")
	if !strings.Contains(joined, "--data-parallel-start-rank 1") {
		t.Fatalf("expected start-rank 1 for node 2, got %v", flags)
	}
}

func TestBuildVLLMDistributedFlagsInvalidTopologyFails(t *testing.T) {
	sku := &models.SLLMSku{SLLMSkuBase: models.SLLMSkuBase{Devices: fourA100Devices()}}
	llm := &models.SLLM{DistRole: api.LLM_DIST_ROLE_LEADER, DistRank: 0, DistNnodes: 2}
	llm.Devices = fourA100Devices()
	if _, _, _, err := buildVLLMDistributedFlags(llm, sku, []string{"--data-parallel-size=3"}); err == nil {
		t.Fatal("expected topology error")
	}
}

func TestBuildVLLMDistributedFlagsDPOnlyStartRank(t *testing.T) {
	// 2 nodes x 4 GPUs, TP=4 => one DP rank per node; node 1 starts at rank 1.
	sku := &models.SLLMSku{SLLMSkuBase: models.SLLMSkuBase{Devices: fourA100Devices()}}
	llm := &models.SLLM{
		DistRole:   api.LLM_DIST_ROLE_FOLLOWER,
		DistRank:   1,
		DistNnodes: 2,
	}
	llm.Devices = fourA100Devices()
	_, flags, _, err := buildVLLMDistributedFlags(llm, sku, []string{"--tensor-parallel-size=4"})
	if err != nil {
		t.Fatalf("build flags: %v", err)
	}
	joined := strings.Join(flags, " ")
	if !strings.Contains(joined, "--data-parallel-start-rank 1") {
		t.Fatalf("expected dp_only start-rank 1, got %v", flags)
	}
	if !strings.Contains(joined, "--data-parallel-size-local 1") {
		t.Fatalf("expected one local DP rank per node, got %v", flags)
	}
}

func TestSingleNodeVLLMAllowsDPAndPPArgs(t *testing.T) {
	// Distributed-only keys must not be globally protected: single-node
	// deployments may still set DP/PP themselves.
	for _, key := range []string{"data-parallel-size", "pipeline-parallel-size", "nnodes"} {
		if err := validateVLLMArgKey(key); err != nil {
			t.Fatalf("single-node key %q should be allowed: %v", key, err)
		}
	}
	if _, ok := protectedVLLMArgKeys["data-parallel-size"]; ok {
		t.Fatal("data-parallel-size must not be in the global protected set")
	}
}

func TestDistributedBackendParameterFilterDropsManagedFlags(t *testing.T) {
	items := []string{
		"--nnodes=2",
		"--data-parallel-size=2",
		"--gpu-memory-utilization=0.9",
	}
	out := filterDistributedBackendParameters(items, protectedDistributedVLLMKeys())
	joined := strings.Join(out, " ")
	if strings.Contains(joined, "nnodes") || strings.Contains(joined, "data-parallel-size") {
		t.Fatalf("managed flags were not filtered: %v", out)
	}
	if !strings.Contains(joined, "gpu-memory-utilization") {
		t.Fatalf("unrelated flag was dropped: %v", out)
	}
}

func TestBuildSGLangDistributedFlags(t *testing.T) {
	sku := &models.SLLMSku{
		SLLMSkuBase: models.SLLMSkuBase{
			Devices: &api.Devices{{Model: "A100"}, {Model: "A100"}},
		},
	}
	llm := &models.SLLM{
		DistRole:       api.LLM_DIST_ROLE_FOLLOWER,
		DistRank:       1,
		DistNnodes:     2,
		DistMasterAddr: "10.0.0.2",
	}
	llm.Devices = &api.Devices{{Model: "A100"}, {Model: "A100"}}
	tp, flags, err := buildSGLangDistributedFlags(llm, sku, nil)
	if err != nil {
		t.Fatalf("build flags: %v", err)
	}
	if tp != 2 {
		t.Fatalf("tp = %d", tp)
	}
	joined := strings.Join(flags, " ")
	if !strings.Contains(joined, "--nnodes 2") || !strings.Contains(joined, "--node-rank 1") {
		t.Fatalf("flags = %v", flags)
	}
	if !strings.Contains(joined, "--pp-size 2") {
		t.Fatalf("expected pp-size: %v", flags)
	}
	if !strings.Contains(joined, "10.0.0.2:29500") {
		t.Fatalf("expected dist-init-addr: %v", flags)
	}
}

func TestBuildSGLangDistributedFlagsRespectsUserPP(t *testing.T) {
	sku := &models.SLLMSku{SLLMSkuBase: models.SLLMSkuBase{Devices: &api.Devices{{Model: "A100"}, {Model: "A100"}}}}
	llm := &models.SLLM{DistRole: api.LLM_DIST_ROLE_FOLLOWER, DistRank: 1, DistNnodes: 4}
	llm.Devices = &api.Devices{{Model: "A100"}, {Model: "A100"}}
	_, flags, err := buildSGLangDistributedFlags(llm, sku, []string{"--pp-size=3"})
	if err != nil {
		t.Fatalf("build flags: %v", err)
	}
	joined := strings.Join(flags, " ")
	if !strings.Contains(joined, "--pp-size 3") {
		t.Fatalf("expected user pp-size 3, got %v", flags)
	}

	// A pp larger than the node count makes the world size exceed the GPUs in
	// the group and must be rejected at pod create.
	small := &models.SLLM{DistRole: api.LLM_DIST_ROLE_FOLLOWER, DistRank: 1, DistNnodes: 3}
	small.Devices = &api.Devices{{Model: "A100"}, {Model: "A100"}}
	_, _, err = buildSGLangDistributedFlags(small, sku, []string{"--pp-size=4"})
	if err == nil {
		t.Fatalf("expected pp-size 4 on dist_nnodes 3 to be rejected")
	}
	if !strings.Contains(err.Error(), "--pp-size 4 exceeds dist_nnodes 3") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestSplitDistributedBackendParametersKeepsUserTopology(t *testing.T) {
	// pp-size 2 on a 3-node group: smaller than the node count (so a valid
	// world size) but different from the default, proving the user value wins.
	raw := []string{"--pp-size=2", "--gpu-memory-utilization=0.9"}
	distLLM := &models.SLLM{DistRole: api.LLM_DIST_ROLE_LEADER, DistRank: 0, DistNnodes: 3}

	// Single-node replicas keep every parameter untouched.
	topo, cmd := splitDistributedBackendParameters(raw, &models.SLLM{}, protectedDistributedSGLangKeys())
	if !strings.Contains(strings.Join(topo, " "), "--pp-size=2") ||
		!strings.Contains(strings.Join(cmd, " "), "--pp-size=2") {
		t.Fatalf("single-node params were filtered: topo=%v cmd=%v", topo, cmd)
	}

	// Distributed members derive topology from raw params but drop managed flags
	// from the final command.
	topo, cmd = splitDistributedBackendParameters(raw, distLLM, protectedDistributedSGLangKeys())
	if !strings.Contains(strings.Join(topo, " "), "--pp-size=2") {
		t.Fatalf("topology lost user pp-size: %v", topo)
	}
	if strings.Contains(strings.Join(cmd, " "), "--pp-size") {
		t.Fatalf("command kept managed pp-size: %v", cmd)
	}
	if !strings.Contains(strings.Join(cmd, " "), "--gpu-memory-utilization=0.9") {
		t.Fatalf("command dropped unrelated parameter: %v", cmd)
	}

	sku := &models.SLLMSku{SLLMSkuBase: models.SLLMSkuBase{Devices: fourA100Devices()}}
	_, flags, err := buildSGLangDistributedFlags(distLLM, sku, topo)
	if err != nil {
		t.Fatalf("build flags: %v", err)
	}
	if !strings.Contains(strings.Join(flags, " "), "--pp-size 2") {
		t.Fatalf("user pp-size not applied: %v", flags)
	}
}

func TestSplitDistributedBackendParametersKeepsVLLMPipelineParallel(t *testing.T) {
	raw := []string{"--pipeline-parallel-size=2"}
	llm := &models.SLLM{DistRole: api.LLM_DIST_ROLE_LEADER, DistRank: 0, DistNnodes: 2}
	llm.Devices = fourA100Devices()
	sku := &models.SLLMSku{SLLMSkuBase: models.SLLMSkuBase{Devices: fourA100Devices()}}

	topo, cmd := splitDistributedBackendParameters(raw, llm, protectedDistributedVLLMKeys())
	if strings.Contains(strings.Join(cmd, " "), "pipeline-parallel-size") {
		t.Fatalf("command kept managed pipeline parallel: %v", cmd)
	}
	_, flags, _, err := buildVLLMDistributedFlags(llm, sku, topo)
	if err != nil {
		t.Fatalf("build flags: %v", err)
	}
	joined := strings.Join(flags, " ")
	if !strings.Contains(joined, "--pipeline-parallel-size 2") {
		t.Fatalf("user pipeline parallel not applied: %v", flags)
	}
	if !strings.Contains(joined, "--nnodes 2") {
		t.Fatalf("expected mp_only layout, got %v", flags)
	}
}

func TestDistPortsOffsetByGroup(t *testing.T) {
	llm := &models.SLLM{DistRole: api.LLM_DIST_ROLE_LEADER, DistRank: 0, DistNnodes: 2, DistGroup: 3}
	if got, want := distMasterPort(llm), api.LLM_DIST_MASTER_PORT+3*api.LLM_DIST_PORT_STRIDE; got != want {
		t.Fatalf("master port = %d, want %d", got, want)
	}
	if got, want := distDPRPCPort(llm), api.LLM_DIST_DP_RPC_PORT+3*api.LLM_DIST_PORT_STRIDE; got != want {
		t.Fatalf("dp rpc port = %d, want %d", got, want)
	}
	if got := distMasterPort(nil); got != api.LLM_DIST_MASTER_PORT {
		t.Fatalf("nil master port = %d", got)
	}
	if got := distDPRPCPort(&models.SLLM{}); got != api.LLM_DIST_DP_RPC_PORT {
		t.Fatalf("group 0 dp rpc port = %d", got)
	}
}

// On a hostlocal network the follower reaches the leader through the leader
// host's mapped ports, persisted on the member; they override the in-pod math.
func TestDistPortsPreferHostMappedPorts(t *testing.T) {
	llm := &models.SLLM{
		DistRole:       api.LLM_DIST_ROLE_FOLLOWER,
		DistRank:       1,
		DistNnodes:     2,
		DistGroup:      0,
		DistMasterPort: 23456,
		DistDPRPCPort:  23457,
	}
	if got := distMasterPort(llm); got != 23456 {
		t.Fatalf("master port = %d, want host-mapped 23456", got)
	}
	if got := distDPRPCPort(llm); got != 23457 {
		t.Fatalf("dp rpc port = %d, want host-mapped 23457", got)
	}

	sku := &models.SLLMSku{SLLMSkuBase: models.SLLMSkuBase{Devices: &api.Devices{{Model: "A100"}, {Model: "A100"}}}}
	llm.Devices = &api.Devices{{Model: "A100"}, {Model: "A100"}}
	// Pin DP explicitly: unpinned multi-node now defaults to the official
	// TP x PP layout, which has no DP RPC port.
	_, flags, _, err := buildVLLMDistributedFlags(llm, sku, []string{"--data-parallel-size=2"})
	if err != nil {
		t.Fatalf("build flags: %v", err)
	}
	joined := strings.Join(flags, " ")
	if !strings.Contains(joined, "--data-parallel-rpc-port 23457") {
		t.Fatalf("expected host-mapped dp rpc port, got %v", flags)
	}
}

func TestDistributedStartScriptPrefixFallsBackToDefaultRouteInterface(t *testing.T) {
	prefix := distributedStartScriptPrefix()
	if !strings.Contains(prefix, "/sys/class/net/") {
		t.Fatalf("expected interface existence check, got %q", prefix)
	}
	if !strings.Contains(prefix, "/proc/net/route") {
		t.Fatalf("expected default-route interface fallback, got %q", prefix)
	}
	if !strings.Contains(prefix, "GLOO_SOCKET_IFNAME") || !strings.Contains(prefix, "NCCL_SOCKET_IFNAME") {
		t.Fatalf("expected NCCL/GLOO exports, got %q", prefix)
	}
	if !strings.Contains(prefix, "VLLM_HOST_IP") {
		t.Fatalf("expected VLLM_HOST_IP export, got %q", prefix)
	}
	if !strings.Contains(prefix, "ip -4 addr show dev") {
		t.Fatalf("expected socket-interface IP derivation, got %q", prefix)
	}
	// The interface must be resolved before the host IP is derived from it.
	ifaceIdx := strings.Index(prefix, "/sys/class/net/")
	ipIdx := strings.Index(prefix, "VLLM_HOST_IP")
	if ifaceIdx < 0 || ipIdx < 0 || ifaceIdx > ipIdx {
		t.Fatalf("expected interface resolution to precede VLLM_HOST_IP derivation, got %q", prefix)
	}
}

func TestVLLMDPEnvForLeaderAndFollower(t *testing.T) {
	leader := &models.SLLM{DistRole: api.LLM_DIST_ROLE_LEADER, DistRank: 0, DistNnodes: 2, DistGroup: 0}
	envs := appendVLLMDPEnv(nil, leader)
	values := map[string]string{}
	for _, e := range envs {
		values[e.Key] = e.Value
	}
	if _, ok := values["VLLM_DP_MASTER_IP"]; ok {
		t.Fatalf("leader must resolve its own IP in the entrypoint, got %v", values)
	}
	if values["VLLM_DP_MASTER_PORT"] != "29500" {
		t.Fatalf("leader dp master port = %q", values["VLLM_DP_MASTER_PORT"])
	}
	if !strings.Contains(vllmDPEnvScriptPrefix(leader), "VLLM_DP_MASTER_IP:-$(hostname -I") {
		t.Fatalf("leader script must resolve its own IP: %q", vllmDPEnvScriptPrefix(leader))
	}

	follower := &models.SLLM{
		DistRole:       api.LLM_DIST_ROLE_FOLLOWER,
		DistRank:       1,
		DistNnodes:     2,
		DistGroup:      0,
		DistMasterAddr: "10.0.0.9",
		DistMasterPort: 23456,
	}
	envs = appendVLLMDPEnv(nil, follower)
	values = map[string]string{}
	for _, e := range envs {
		values[e.Key] = e.Value
	}
	if values["VLLM_DP_MASTER_IP"] != "10.0.0.9" {
		t.Fatalf("follower dp master ip = %q", values["VLLM_DP_MASTER_IP"])
	}
	if values["VLLM_DP_MASTER_PORT"] != "23456" {
		t.Fatalf("follower dp master port = %q", values["VLLM_DP_MASTER_PORT"])
	}
	if !strings.Contains(vllmDPEnvScriptPrefix(follower), "-23456}") {
		t.Fatalf("follower script default must keep the mapped port: %q", vllmDPEnvScriptPrefix(follower))
	}
}

func eightA100Devices() *api.Devices {
	return &api.Devices{
		{Model: "A100"}, {Model: "A100"}, {Model: "A100"}, {Model: "A100"},
		{Model: "A100"}, {Model: "A100"}, {Model: "A100"}, {Model: "A100"},
	}
}

func TestInjectPipelineParallelParameterOverridesSkuPin(t *testing.T) {
	out := injectPipelineParallelParameter([]string{
		"--pipeline-parallel-size=2",
		"--pp-size 3",
		"--pp=4",
		"--gpu-memory-utilization=0.9",
	}, "pipeline-parallel-size", 4)
	joined := strings.Join(out, " ")
	if strings.Contains(joined, "=2") || strings.Contains(joined, "pp-size 3") || strings.Contains(joined, "--pp=4") {
		t.Fatalf("sku pins were not replaced: %v", out)
	}
	if strings.Count(joined, "pipeline-parallel-size") != 1 || !strings.Contains(joined, "--pipeline-parallel-size 4") {
		t.Fatalf("exactly one deployment pin expected: %v", out)
	}
	if !strings.Contains(joined, "gpu-memory-utilization=0.9") {
		t.Fatalf("unrelated parameter dropped: %v", out)
	}
}

func TestSingleNodeVLLMDeploymentPPDerivesTP(t *testing.T) {
	sku := &models.SLLMSku{SLLMSkuBase: models.SLLMSkuBase{Devices: eightA100Devices()}}
	llm := &models.SLLM{}
	tp, _, _, err := buildVLLMDistributedFlags(llm, sku, []string{"--pipeline-parallel-size=4"})
	if err != nil {
		t.Fatalf("build flags: %v", err)
	}
	if tp != 2 {
		t.Fatalf("tp = %d, want 2 (gpus/pp)", tp)
	}

	_, _, _, err = buildVLLMDistributedFlags(llm, sku, []string{"--pipeline-parallel-size=3"})
	if err == nil {
		t.Fatal("expected non-divisible single-node PP to be rejected")
	}
}

func TestSingleNodeSGLangDeploymentPPDerivesTP(t *testing.T) {
	sku := &models.SLLMSku{SLLMSkuBase: models.SLLMSkuBase{Devices: fourA100Devices()}}
	llm := &models.SLLM{}
	tp, _, err := buildSGLangDistributedFlags(llm, sku, []string{"--pp-size=2"})
	if err != nil {
		t.Fatalf("build flags: %v", err)
	}
	if tp != 2 {
		t.Fatalf("tp = %d, want 2 (gpus/pp)", tp)
	}

	_, _, err = buildSGLangDistributedFlags(llm, sku, []string{"--pp-size=5"})
	if err == nil {
		t.Fatal("expected pp above the GPU count to be rejected")
	}
}

func TestSingleNodeVLLMWithoutPPKeepsAllGPUs(t *testing.T) {
	sku := &models.SLLMSku{SLLMSkuBase: models.SLLMSkuBase{Devices: eightA100Devices()}}
	tp, _, _, err := buildVLLMDistributedFlags(&models.SLLM{}, sku, nil)
	if err != nil {
		t.Fatalf("build flags: %v", err)
	}
	if tp != 8 {
		t.Fatalf("tp = %d, want 8 (regression: no-PP single node must keep all GPUs)", tp)
	}
}
