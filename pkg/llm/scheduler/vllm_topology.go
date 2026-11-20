package scheduler

import (
	"fmt"
	"strconv"
	"strings"
)

// MultinodeShape is the vLLM MP layout across nodes.
type MultinodeShape string

const (
	ShapeDPOnly MultinodeShape = "dp_only"
	ShapeMPOnly MultinodeShape = "mp_only"
	ShapeNested MultinodeShape = "nested"
)

// MultinodeUserParallelism captures optional user-pinned parallelism.
// Nil pointer fields mean the planner should derive them.
type MultinodeUserParallelism struct {
	TP  *int
	PP  *int
	DP  *int
	DPL *int
	PCP *int
}

// ValidatedTopology is the cluster-level layout after validation.
type ValidatedTopology struct {
	Shape          MultinodeShape
	TP             int
	PP             int
	PCP            int
	DP             int
	DPLPerNode     []int
	NNodesWithinDP int
}

// NodeTopology is ValidatedTopology plus the perspective of one node.
type NodeTopology struct {
	ValidatedTopology
	NNodes    int
	NodeRank  int
	StartRank int
	DPL       int
}

// EffectiveParallelism merges SKU backend parameters with the deployment
// pipeline pin (deployment wins) and the deployment parallel_mode, then applies
// the official default for any still-unpinned dimension.
// Any caller that validates a multi-node vLLM layout must use this so planned
// layouts and emitted flags agree.
func EffectiveParallelism(backendParameters []string, pipelineParallelSize int, parallelMode string, gpuPerNode []int) MultinodeUserParallelism {
	user := ParseUserParallelism(backendParameters)
	if pipelineParallelSize > 0 {
		pp := pipelineParallelSize
		user.PP = &pp
	}
	ApplyParallelMode(&user, parallelMode, gpuPerNode)
	ApplyDefaultMultinodeParallelism(&user, gpuPerNode)
	return user
}

// ApplyDefaultMultinodeParallelism applies the official vLLM multi-node
// default (TP = GPUs per node, PP = node count, DP = 1) when the user pinned
// nothing. Any user pin (TP/PP/DP/DPL/PCP) keeps the existing inference
// rules. Single-node groups (< 2) are left untouched (TP=gpus, PP=1).
//
// Both the planner (validatePlanTopology) and the container driver
// (buildVLLMDistributedFlags) must call this with the same inputs so the
// planned layout and the flags actually emitted stay consistent.
func ApplyDefaultMultinodeParallelism(user *MultinodeUserParallelism, gpuPerNode []int) {
	if user == nil || len(gpuPerNode) < 2 {
		return
	}
	if user.TP != nil || user.PP != nil || user.DP != nil || user.DPL != nil || user.PCP != nil {
		return
	}
	pp := len(gpuPerNode)
	user.PP = &pp
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	if a < 0 {
		return -a
	}
	return a
}

func gcdSlice(vals []int) int {
	if len(vals) == 0 {
		return 0
	}
	g := vals[0]
	for i := 1; i < len(vals); i++ {
		g = gcd(g, vals[i])
		if g == 1 {
			return 1
		}
	}
	return g
}

// ValidateMultinodeTopology mirrors GPUStack validate_multinode_topology.
func ValidateMultinodeTopology(gpuPerNode []int, user MultinodeUserParallelism) (*ValidatedTopology, error) {
	if len(gpuPerNode) == 0 {
		return nil, fmt.Errorf("vLLM multi-node: empty gpu_per_node")
	}
	for i, g := range gpuPerNode {
		if g <= 0 {
			return nil, fmt.Errorf("vLLM multi-node: gpu_per_node[%d]=%d must be positive", i, g)
		}
	}
	nnodes := len(gpuPerNode)

	for name, value := range map[string]*int{
		"pipeline-parallel-size":        user.PP,
		"prefill-context-parallel-size": user.PCP,
		"data-parallel-size":            user.DP,
		"data-parallel-size-local":      user.DPL,
	} {
		if value != nil && *value <= 0 {
			return nil, fmt.Errorf("vLLM multi-node: --%s %d must be positive", name, *value)
		}
	}

	pp := 1
	if user.PP != nil {
		pp = *user.PP
	}
	pcp := 1
	if user.PCP != nil {
		pcp = *user.PCP
	}

	var tp int
	switch {
	case user.TP != nil:
		tp = *user.TP
	case user.DP != nil:
		totalGPUs := 0
		for _, g := range gpuPerNode {
			totalGPUs += g
		}
		denom := (*user.DP) * pp * pcp
		if denom == 0 || totalGPUs%denom != 0 {
			return nil, fmt.Errorf(
				"vLLM multi-node: cannot infer --tensor-parallel-size from --data-parallel-size %d, --pipeline-parallel-size %d and --prefill-context-parallel-size %d: total GPUs %d is not a multiple of %d",
				*user.DP, pp, pcp, totalGPUs, denom,
			)
		}
		tp = totalGPUs / denom
	default:
		tp = gcdSlice(gpuPerNode)
	}
	if tp <= 0 {
		return nil, fmt.Errorf("vLLM multi-node: --tensor-parallel-size %d must be positive", tp)
	}

	workersPerDP := tp * pp * pcp
	minNode := gpuPerNode[0]
	for _, g := range gpuPerNode[1:] {
		if g < minNode {
			minNode = g
		}
	}

	if workersPerDP <= minNode {
		dplPerNode := make([]int, len(gpuPerNode))
		dpTotal := 0
		for i, g := range gpuPerNode {
			dplPerNode[i] = g / workersPerDP
			dpTotal += dplPerNode[i]
		}
		if user.DP != nil && *user.DP != dpTotal {
			return nil, fmt.Errorf(
				"vLLM multi-node: --data-parallel-size %d does not match derived total %d (per-node DP-Local = %v)",
				*user.DP, dpTotal, dplPerNode,
			)
		}
		if user.DPL != nil {
			for _, d := range dplPerNode {
				if d != *user.DPL {
					return nil, fmt.Errorf(
						"vLLM multi-node: --data-parallel-size-local %d cannot be applied uniformly to a heterogeneous worker group (per-node GPUs = %v, derived per-node DP-Local = %v)",
						*user.DPL, gpuPerNode, dplPerNode,
					)
				}
			}
		}
		return &ValidatedTopology{
			Shape:          ShapeDPOnly,
			TP:             tp,
			PP:             pp,
			PCP:            pcp,
			DP:             dpTotal,
			DPLPerNode:     dplPerNode,
			NNodesWithinDP: 1,
		}, nil
	}

	homogeneous := true
	for i := 1; i < len(gpuPerNode); i++ {
		if gpuPerNode[i] != gpuPerNode[0] {
			homogeneous = false
			break
		}
	}
	if !homogeneous {
		return nil, fmt.Errorf(
			"vLLM multi-node: cross-node TP/PP (workers_per_dp=%d > min node capacity %d) requires a homogeneous cluster, but workers have differing GPU counts %v",
			workersPerDP, minNode, gpuPerNode,
		)
	}
	gNode := gpuPerNode[0]
	if workersPerDP%gNode != 0 {
		return nil, fmt.Errorf(
			"vLLM multi-node: workers_per_dp=%d cannot evenly distribute across nodes with %d GPUs each",
			workersPerDP, gNode,
		)
	}
	nnodesWithinDP := workersPerDP / gNode
	if nnodes%nnodesWithinDP != 0 {
		return nil, fmt.Errorf(
			"vLLM multi-node: cluster size %d cannot evenly distribute DP ranks that each span %d nodes",
			nnodes, nnodesWithinDP,
		)
	}
	inferredDP := nnodes / nnodesWithinDP
	if user.DP != nil && *user.DP != inferredDP {
		return nil, fmt.Errorf(
			"vLLM multi-node: --data-parallel-size %d does not match derived total %d (cluster fits %d DP ranks of %d nodes each)",
			*user.DP, inferredDP, inferredDP, nnodesWithinDP,
		)
	}
	if user.DPL != nil && *user.DPL != 1 {
		return nil, fmt.Errorf(
			"vLLM multi-node: --data-parallel-size-local %d is invalid in cross-node TP/PP layouts (must be 1)",
			*user.DPL,
		)
	}
	shape := ShapeMPOnly
	if inferredDP > 1 {
		shape = ShapeNested
	}
	dpl := make([]int, nnodes)
	for i := range dpl {
		dpl[i] = 1
	}
	return &ValidatedTopology{
		Shape:          shape,
		TP:             tp,
		PP:             pp,
		PCP:            pcp,
		DP:             inferredDP,
		DPLPerNode:     dpl,
		NNodesWithinDP: nnodesWithinDP,
	}, nil
}

// CalMultinodeTopology returns the layout for one node rank.
func CalMultinodeTopology(gpuPerNode []int, nodeRank int, user MultinodeUserParallelism) (*NodeTopology, error) {
	validated, err := ValidateMultinodeTopology(gpuPerNode, user)
	if err != nil {
		return nil, err
	}
	if nodeRank < 0 || nodeRank >= len(gpuPerNode) {
		return nil, fmt.Errorf("vLLM multi-node: node_rank %d out of range [0,%d)", nodeRank, len(gpuPerNode))
	}
	startRank := 0
	if validated.Shape == ShapeDPOnly {
		for i := 0; i < nodeRank; i++ {
			startRank += validated.DPLPerNode[i]
		}
	} else {
		// mp_only / nested: vLLM derives the DP rank from
		// node_rank / nnodes_within_dp, not from the per-node DP-local sum.
		startRank = nodeRank / validated.NNodesWithinDP
	}
	return &NodeTopology{
		ValidatedTopology: *validated,
		NNodes:            len(gpuPerNode),
		NodeRank:          nodeRank,
		StartRank:         startRank,
		DPL:               validated.DPLPerNode[nodeRank],
	}, nil
}

func intPtr(v int) *int { return &v }

// ParseUserParallelism extracts TP/PP/DP from backend parameter strings.
func ParseUserParallelism(backendParameters []string) MultinodeUserParallelism {
	user := MultinodeUserParallelism{}
	for _, item := range backendParameters {
		key, value, ok := splitBackendParam(item)
		if !ok || value == "" {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			continue
		}
		switch key {
		case "tensor-parallel-size", "tp", "tp-size":
			user.TP = intPtr(n)
		case "pipeline-parallel-size", "pp", "pp-size":
			user.PP = intPtr(n)
		case "data-parallel-size", "dp", "dp-size":
			user.DP = intPtr(n)
		case "data-parallel-size-local", "dpl":
			user.DPL = intPtr(n)
		case "prefill-context-parallel-size", "pcp":
			user.PCP = intPtr(n)
		}
	}
	return user
}

func splitBackendParam(item string) (key, value string, ok bool) {
	item = strings.TrimSpace(item)
	if item == "" {
		return "", "", false
	}
	if strings.HasPrefix(item, "--") {
		item = strings.TrimSpace(strings.TrimPrefix(item, "--"))
	} else if strings.HasPrefix(item, "-") {
		return "", "", false
	}
	if item == "" {
		return "", "", false
	}
	if idx := strings.Index(item, "="); idx >= 0 {
		return strings.TrimSpace(item[:idx]), item[idx+1:], true
	}
	fields := strings.Fields(item)
	if len(fields) >= 2 {
		return fields[0], strings.TrimSpace(item[len(fields[0]):]), true
	}
	return item, "", true
}
