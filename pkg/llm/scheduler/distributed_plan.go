package scheduler

import (
	"fmt"
	"sort"
	"strings"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

const (
	PlacementSpread  = "spread"
	PlacementBinpack = "binpack"

	// DistMasterPort is the fixed PyTorch / SGLang dist-init port inside the pod network.
	DistMasterPort = 29500
	// DistDPRPCPort is the fixed vLLM DP RPC port inside the pod network.
	DistDPRPCPort = 29501
)

// HostGPUInventory is free GPU capacity of one host for a device model.
type HostGPUInventory struct {
	HostId   string
	HostName string
	// HostIp is the host access IP, used for placement diagnostics.
	HostIp string
	// Online is false only when the host is explicitly reported offline.
	Online bool
	// GpuModel is the model of the free devices counted by FreeGPUs.
	GpuModel string
	// MinVRAMMb is the smallest VRAM among the free devices. Zero means the
	// host has not reported device memory.
	MinVRAMMb int
	FreeGPUs  int
}

// PlanOptions controls PlanReplica.
type PlanOptions struct {
	TotalGPUs             int
	DeviceTemplate        api.Device
	PreferHosts           []string
	PlacementStrategy     string
	DistributedInference  bool
	ForceSingleGPUPerNode bool // Moore Threads HAMI: at most 1 GPU per pod
	// ForceNNodes pins the number of pods per logical replica: 0 = auto
	// (single host first), 1 = force a single host, >= 2 = force multi-host
	// with exactly this many nodes.
	ForceNNodes int
	// Backend is the inference backend type (vllm or sglang); empty means
	// vllm. It selects which topology rules validate candidate layouts.
	Backend string
	// BackendParameters are the SKU's raw backend parameters; pinned
	// parallelism (tp/pp/dp/pcp) is parsed from them.
	BackendParameters []string
	// PipelineParallelSize overrides any pipeline-parallel pin in
	// BackendParameters when > 0 (the deployment-level pin wins).
	PipelineParallelSize int
	// ParallelMode selects the multi-node layout: "pp" (default) or "dp".
	// It only affects multi-node vLLM groups; single-node and SGLang groups
	// ignore it.
	ParallelMode string
	Hosts        []HostGPUInventory
}

// ReplicaMemberPlan is one pod in a logical replica.
type ReplicaMemberPlan struct {
	HostId   string
	HostName string
	HostIp   string
	Rank     int
	Devices  api.Devices
}

// PlanParallelism is the derived parallel layout of a replica plan.
type PlanParallelism struct {
	TP int
	PP int
	DP int
}

// ReplicaPlan is a single-pod or multi-pod placement for one logical replica.
type ReplicaPlan struct {
	Distributed bool
	NNodes      int
	GpusPerNode int
	Members     []ReplicaMemberPlan
	// Parallelism is the derived TP/PP/DP layout of the plan, used for
	// schedulable-check output and OpsLog diagnostics.
	Parallelism *PlanParallelism
}

// userParallelismFromOptions merges the SKU backend parameters with the
// deployment-level pipeline pin; the deployment pin wins.
func userParallelismFromOptions(opts PlanOptions) MultinodeUserParallelism {
	user := ParseUserParallelism(opts.BackendParameters)
	if opts.PipelineParallelSize > 0 {
		pp := opts.PipelineParallelSize
		user.PP = &pp
	}
	return user
}

// ValidateSingleNodePipelineParallel checks a pinned PP against a single
// node's GPU count: TP derives as gpus/pp so the world fills the node exactly.
func ValidateSingleNodePipelineParallel(gpus int, pp *int) error {
	if pp == nil || *pp <= 1 {
		return nil
	}
	if *pp > gpus {
		return fmt.Errorf("pinned pipeline parallelism %d exceeds the %d GPUs on a single node", *pp, gpus)
	}
	if gpus%*pp != 0 {
		return fmt.Errorf("pinned pipeline parallelism %d does not divide %d GPUs (tensor parallel size would be fractional)", *pp, gpus)
	}
	return nil
}

// validatePlanTopology derives and validates the parallel layout for one
// candidate homogeneous node group. vLLM uses the MP topology rules; SGLang
// fixes tp to the per-node GPU count and only requires pp <= nnodes.
func validatePlanTopology(opts PlanOptions, gpuPerNode []int, user MultinodeUserParallelism) (*PlanParallelism, error) {
	switch opts.Backend {
	case api.LLM_MODEL_BACKEND_SGLANG:
		topo, err := ValidateSGLangMultinodeTopology(gpuPerNode, user)
		if err != nil {
			return nil, err
		}
		return &PlanParallelism{TP: topo.TP, PP: topo.PP, DP: 1}, nil
	default: // vllm
		// Apply deployment-level parallel_mode when the user pinned nothing.
		// PP (default): TP = GPUs per node, PP = node count, DP = 1.
		// DP: TP = GPUs per node, PP = 1, DP = node count.
		ApplyParallelMode(&user, opts.ParallelMode, gpuPerNode)
		ApplyDefaultMultinodeParallelism(&user, gpuPerNode)
		topo, err := ValidateMultinodeTopology(gpuPerNode, user)
		if err != nil {
			return nil, err
		}
		return &PlanParallelism{TP: topo.TP, PP: topo.PP, DP: topo.DP}, nil
	}
}

// ApplyParallelMode sets DP or PP from the deployment mode when the user did
// not explicitly pin the corresponding parallelism. Explicit pins always win.
func ApplyParallelMode(user *MultinodeUserParallelism, mode string, gpuPerNode []int) {
	if user == nil || len(gpuPerNode) < 2 {
		return
	}
	mode = api.NormalizeLLMDeploymentParallelMode(mode)
	switch mode {
	case api.LLM_DEPLOYMENT_PARALLEL_MODE_DP:
		if user.DP == nil {
			dp := len(gpuPerNode)
			user.DP = &dp
		}
	case api.LLM_DEPLOYMENT_PARALLEL_MODE_PP:
		// Default PP behavior is applied by ApplyDefaultMultinodeParallelism.
	}
}

// singleNodeParallelism reports the derived layout of a single-node plan.
func singleNodeParallelism(totalGPUs int, user MultinodeUserParallelism) *PlanParallelism {
	pp := 1
	tp := totalGPUs
	if user.PP != nil && *user.PP > 1 {
		pp = *user.PP
		tp = totalGPUs / pp
	}
	return &PlanParallelism{TP: tp, PP: pp, DP: 1}
}

// PlanReplica prefers a single host that can hold all GPUs. When that is
// impossible and DistributedInference is set, it builds a homogeneous multi-host plan.
// ForceNNodes (0 = auto) pins the layout: 1 only allows a single host, >= 2
// requires exactly that many homogeneous nodes.
func PlanReplica(opts PlanOptions) (*ReplicaPlan, error) {
	if opts.TotalGPUs <= 0 {
		return nil, fmt.Errorf("total GPUs must be positive")
	}
	if opts.ForceNNodes < 0 {
		return nil, fmt.Errorf("nnodes must be >= 0, got %d", opts.ForceNNodes)
	}
	hosts := filterPreferHosts(opts.Hosts, opts.PreferHosts)
	hosts = filterUsableHosts(hosts, opts.DeviceTemplate)
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no candidate hosts with free GPUs")
	}
	sortHosts(hosts, opts.PlacementStrategy)

	user := userParallelismFromOptions(opts)

	if opts.ForceNNodes >= 2 {
		return planForcedNodes(opts, user, hosts)
	}

	allowSinglePod := !opts.ForceSingleGPUPerNode || opts.TotalGPUs == 1
	if allowSinglePod {
		ppErr := ValidateSingleNodePipelineParallel(opts.TotalGPUs, user.PP)
		if ppErr != nil {
			if opts.ForceNNodes == 1 || !opts.DistributedInference {
				// Fail fast: an unrealizable PP would otherwise surface as a
				// cryptic engine startup error after the pod is created.
				return nil, ppErr
			}
			// A pinned PP that cannot fit one node may still fit a multi-host
			// layout; skip the single-host attempt.
		} else {
			for _, h := range hosts {
				if h.FreeGPUs >= opts.TotalGPUs {
					plan := &ReplicaPlan{
						Distributed: false,
						NNodes:      1,
						GpusPerNode: opts.TotalGPUs,
						Members: []ReplicaMemberPlan{{
							HostId:   h.HostId,
							HostName: h.HostName,
							HostIp:   h.HostIp,
							Rank:     0,
							Devices:  cloneDevices(opts.DeviceTemplate, opts.TotalGPUs),
						}},
						Parallelism: singleNodeParallelism(opts.TotalGPUs, user),
					}
					if err := verifyReplicaPlan(plan, opts); err != nil {
						return nil, err
					}
					return plan, nil
				}
			}
		}
	}

	// nnodes=1 pins a single host: never fall back to a multi-host layout.
	if !opts.DistributedInference || opts.ForceNNodes == 1 {
		return nil, fmt.Errorf(
			"no single host has %d free GPUs of the requested model; enable distributed_inference to span hosts",
			opts.TotalGPUs,
		)
	}

	maxPerNode := 0
	for _, h := range hosts {
		if h.FreeGPUs > maxPerNode {
			maxPerNode = h.FreeGPUs
		}
	}
	if opts.ForceSingleGPUPerNode {
		maxPerNode = 1
	}
	if maxPerNode <= 0 {
		return nil, fmt.Errorf("no free GPUs available for distributed placement")
	}

	// Prefer fewer nodes: try larger gpusPerNode first.
	var lastTopoErr error
	for gpusPerNode := maxPerNode; gpusPerNode >= 1; gpusPerNode-- {
		if opts.TotalGPUs%gpusPerNode != 0 {
			continue
		}
		nnodes := opts.TotalGPUs / gpusPerNode
		if nnodes < 2 {
			continue
		}
		eligible := make([]HostGPUInventory, 0, len(hosts))
		for _, h := range hosts {
			if h.FreeGPUs >= gpusPerNode {
				eligible = append(eligible, h)
			}
		}
		if len(eligible) < nnodes {
			continue
		}
		// A candidate layout must also realize the pinned parallelism;
		// otherwise the group would only fail once the pods are built.
		gpuPerNode := make([]int, nnodes)
		for i := range gpuPerNode {
			gpuPerNode[i] = gpusPerNode
		}
		parallel, topoErr := validatePlanTopology(opts, gpuPerNode, user)
		if topoErr != nil {
			lastTopoErr = topoErr
			continue
		}
		sortHosts(eligible, opts.PlacementStrategy)
		members := make([]ReplicaMemberPlan, 0, nnodes)
		for i := 0; i < nnodes; i++ {
			members = append(members, ReplicaMemberPlan{
				HostId:   eligible[i].HostId,
				HostName: eligible[i].HostName,
				HostIp:   eligible[i].HostIp,
				Rank:     i,
				Devices:  cloneDevices(opts.DeviceTemplate, gpusPerNode),
			})
		}
		plan := &ReplicaPlan{
			Distributed: true,
			NNodes:      nnodes,
			GpusPerNode: gpusPerNode,
			Members:     members,
			Parallelism: parallel,
		}
		if err := verifyReplicaPlan(plan, opts); err != nil {
			return nil, err
		}
		return plan, nil
	}

	if lastTopoErr != nil {
		return nil, fmt.Errorf("no host layout satisfies the pinned parallelism: %v", lastTopoErr)
	}

	return nil, fmt.Errorf(
		"cannot find a homogeneous host combination for %d GPUs with distributed inference (need equal GPUs per node)",
		opts.TotalGPUs,
	)
}

// planForcedNodes builds a homogeneous plan with exactly opts.ForceNNodes
// pods. hosts are already filtered and sorted.
func planForcedNodes(opts PlanOptions, user MultinodeUserParallelism, hosts []HostGPUInventory) (*ReplicaPlan, error) {
	nnodes := opts.ForceNNodes
	if opts.TotalGPUs%nnodes != 0 {
		return nil, fmt.Errorf(
			"cannot split %d GPUs evenly into %d nodes; pick a node count that divides the GPU total",
			opts.TotalGPUs, nnodes,
		)
	}
	gpusPerNode := opts.TotalGPUs / nnodes
	if opts.ForceSingleGPUPerNode && gpusPerNode != 1 {
		return nil, fmt.Errorf(
			"this device type allows at most 1 GPU per pod; %d GPUs need %d nodes, cannot use %d",
			opts.TotalGPUs, opts.TotalGPUs, nnodes,
		)
	}
	gpuPerNode := make([]int, nnodes)
	for i := range gpuPerNode {
		gpuPerNode[i] = gpusPerNode
	}
	parallel, err := validatePlanTopology(opts, gpuPerNode, user)
	if err != nil {
		return nil, fmt.Errorf("cannot realize the pinned parallelism with nnodes=%d: %v", nnodes, err)
	}
	eligible := make([]HostGPUInventory, 0, len(hosts))
	for _, h := range hosts {
		if h.FreeGPUs >= gpusPerNode {
			eligible = append(eligible, h)
		}
	}
	if len(eligible) < nnodes {
		maxFree := 0
		for _, h := range hosts {
			if h.FreeGPUs > maxFree {
				maxFree = h.FreeGPUs
			}
		}
		return nil, fmt.Errorf(
			"need %d hosts with at least %d free GPUs each, only %d qualify (largest host has %d free GPUs); pass more prefer_hosts or lower nnodes",
			nnodes, gpusPerNode, len(eligible), maxFree,
		)
	}
	sortHosts(eligible, opts.PlacementStrategy)
	members := make([]ReplicaMemberPlan, 0, nnodes)
	for i := 0; i < nnodes; i++ {
		members = append(members, ReplicaMemberPlan{
			HostId:   eligible[i].HostId,
			HostName: eligible[i].HostName,
			HostIp:   eligible[i].HostIp,
			Rank:     i,
			Devices:  cloneDevices(opts.DeviceTemplate, gpusPerNode),
		})
	}
	plan := &ReplicaPlan{
		Distributed: true,
		NNodes:      nnodes,
		GpusPerNode: gpusPerNode,
		Members:     members,
		Parallelism: parallel,
	}
	if err := verifyReplicaPlan(plan, opts); err != nil {
		return nil, err
	}
	return plan, nil
}

// filterUsableHosts keeps online hosts that can hold the requested device:
// the free devices must match the template model and, when the template
// requests VRAM (HAMi), every free device must have at least that much memory.
func filterUsableHosts(hosts []HostGPUInventory, tmpl api.Device) []HostGPUInventory {
	model := strings.TrimSpace(tmpl.Model)
	reqVram := tmpl.MemoryMb
	out := make([]HostGPUInventory, 0, len(hosts))
	for _, h := range hosts {
		if !h.Online || h.FreeGPUs <= 0 {
			continue
		}
		if model != "" && h.GpuModel != "" && !strings.EqualFold(h.GpuModel, model) {
			continue
		}
		if reqVram > 0 && h.MinVRAMMb > 0 && h.MinVRAMMb < reqVram {
			continue
		}
		out = append(out, h)
	}
	return out
}

// verifyReplicaPlan re-checks the placement invariants that both the single-
// host and multi-host paths rely on: unique hosts, positive per-node GPU
// counts, equal GPUs per node for distributed plans, and an exact GPU total.
func verifyReplicaPlan(plan *ReplicaPlan, opts PlanOptions) error {
	if plan == nil || len(plan.Members) == 0 {
		return fmt.Errorf("empty replica plan")
	}
	seen := make(map[string]struct{}, len(plan.Members))
	total := 0
	for _, m := range plan.Members {
		if strings.TrimSpace(m.HostId) == "" {
			return fmt.Errorf("replica plan member %d has no host", m.Rank)
		}
		if _, dup := seen[m.HostId]; dup {
			return fmt.Errorf("host %s selected more than once in replica plan", m.HostId)
		}
		seen[m.HostId] = struct{}{}
		if len(m.Devices) == 0 {
			return fmt.Errorf("replica plan member %d has no GPUs", m.Rank)
		}
		total += len(m.Devices)
	}
	if total != opts.TotalGPUs {
		return fmt.Errorf("replica plan allocates %d GPUs, want %d", total, opts.TotalGPUs)
	}
	if plan.Distributed {
		if plan.NNodes != len(plan.Members) || plan.NNodes < 2 {
			return fmt.Errorf("distributed plan has %d members, want nnodes %d >= 2", len(plan.Members), plan.NNodes)
		}
		for _, m := range plan.Members {
			if len(m.Devices) != plan.GpusPerNode {
				return fmt.Errorf("replica plan member %d has %d GPUs, want homogeneous %d",
					m.Rank, len(m.Devices), plan.GpusPerNode)
			}
		}
	}
	return nil
}

func filterPreferHosts(hosts []HostGPUInventory, prefer []string) []HostGPUInventory {
	if len(prefer) == 0 {
		return append([]HostGPUInventory(nil), hosts...)
	}
	allowed := make(map[string]struct{}, len(prefer))
	for _, id := range prefer {
		if id == "" {
			continue
		}
		allowed[id] = struct{}{}
	}
	out := make([]HostGPUInventory, 0, len(hosts))
	for _, h := range hosts {
		if _, ok := allowed[h.HostId]; ok {
			out = append(out, h)
		}
	}
	return out
}

func sortHosts(hosts []HostGPUInventory, strategy string) {
	switch strategy {
	case PlacementBinpack:
		sort.SliceStable(hosts, func(i, j int) bool {
			if hosts[i].FreeGPUs != hosts[j].FreeGPUs {
				return hosts[i].FreeGPUs < hosts[j].FreeGPUs
			}
			return hosts[i].HostId < hosts[j].HostId
		})
	default: // spread
		sort.SliceStable(hosts, func(i, j int) bool {
			if hosts[i].FreeGPUs != hosts[j].FreeGPUs {
				return hosts[i].FreeGPUs > hosts[j].FreeGPUs
			}
			return hosts[i].HostId < hosts[j].HostId
		})
	}
}

func cloneDevices(tmpl api.Device, n int) api.Devices {
	out := make(api.Devices, n)
	for i := 0; i < n; i++ {
		out[i] = tmpl
	}
	return out
}

// GpuPerNodeFromPlan returns the per-node GPU counts for topology validation.
func GpuPerNodeFromPlan(plan *ReplicaPlan) []int {
	if plan == nil {
		return nil
	}
	out := make([]int, len(plan.Members))
	for i, m := range plan.Members {
		out[i] = len(m.Devices)
	}
	return out
}
