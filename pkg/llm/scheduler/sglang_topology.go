package scheduler

import "fmt"

// SGLangMultinodeTopology is the validated SGLang multi-node layout.
// SGLang fixes tp to the per-node GPU count; pp defaults to the node count,
// and a user pin must not exceed it (tp*pp is the world size).
type SGLangMultinodeTopology struct {
	NNodes int
	TP     int
	PP     int
}

// ValidateSGLangMultinodeTopology validates a homogeneous SGLang group. It
// mirrors the pp rule previously embedded in buildSGLangDistributedFlags so
// the planner and the container driver reject the same layouts with the same
// diagnostics.
func ValidateSGLangMultinodeTopology(gpuPerNode []int, user MultinodeUserParallelism) (*SGLangMultinodeTopology, error) {
	if len(gpuPerNode) == 0 {
		return nil, fmt.Errorf("SGLang multi-node: empty gpu_per_node")
	}
	for i, g := range gpuPerNode {
		if g <= 0 {
			return nil, fmt.Errorf("SGLang multi-node: gpu_per_node[%d]=%d must be positive", i, g)
		}
	}
	nnodes := len(gpuPerNode)
	if nnodes < 2 {
		return nil, fmt.Errorf("SGLang multi-node: need at least 2 nodes, got %d", nnodes)
	}
	tp := gpuPerNode[0]
	pp := nnodes
	if user.PP != nil && *user.PP > 0 {
		if *user.PP > nnodes {
			return nil, fmt.Errorf(
				"SGLang multi-node: --pp-size %d exceeds dist_nnodes %d (world size %d would exceed the %d GPUs in the group)",
				*user.PP, nnodes, tp*(*user.PP), tp*nnodes)
		}
		pp = *user.PP
	}
	return &SGLangMultinodeTopology{NNodes: nnodes, TP: tp, PP: pp}, nil
}
