package llm_container

import (
	"fmt"
	"strings"

	"yunion.io/x/log"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/llm/scheduler"
)

const distSocketIfName = "eth0"

func llmLocalGPUCount(llm *models.SLLM, sku *models.SLLMSku) int {
	devs := models.GetEffectiveDevices(llm, sku)
	if devs != nil && len(*devs) > 0 {
		return len(*devs)
	}
	return 1
}

func llmDistributedGPUPerNode(llm *models.SLLM, sku *models.SLLMSku) []int {
	if llm == nil || !llm.IsDistributedMember() || llm.DistNnodes <= 0 {
		return []int{llmLocalGPUCount(llm, sku)}
	}
	perNode := llmLocalGPUCount(llm, sku)
	out := make([]int, llm.DistNnodes)
	for i := range out {
		out[i] = perNode
	}
	return out
}

// deploymentPinnedPipelineParallel returns the deployment-level pipeline pin
// for a member's deployment (0 = unset/unavailable). Deployment pins win over
// SKU backend parameters. A fetch failure falls back to the SKU parameters so
// pod builds never fail on a transient DB read.
func deploymentPinnedPipelineParallel(llm *models.SLLM) int {
	if deploy := deploymentForLLM(llm); deploy != nil {
		return deploy.PipelineParallelSize
	}
	return 0
}

// deploymentParallelMode returns the deployment-level parallel mode (default
// "pp"). A fetch failure returns "" so the caller falls back to SKU parameters.
func deploymentParallelMode(llm *models.SLLM) string {
	if deploy := deploymentForLLM(llm); deploy != nil {
		return api.NormalizeLLMDeploymentParallelMode(deploy.ParallelMode)
	}
	return ""
}

func deploymentForLLM(llm *models.SLLM) *models.SLLMDeployment {
	if llm == nil || llm.LLMDeploymentId == "" {
		return nil
	}
	obj, err := models.GetLLMDeploymentManager().FetchById(llm.LLMDeploymentId)
	if err != nil {
		log.Warningf("fetch deployment %s for distributed flags: %v", llm.LLMDeploymentId, err)
		return nil
	}
	if deploy, ok := obj.(*models.SLLMDeployment); ok {
		return deploy
	}
	return nil
}

// injectPipelineParallelParameter replaces every pipeline-parallel pin
// (--pipeline-parallel-size / --pp-size / --pp aliases) with one synthesized
// entry so the deployment pin wins over SKU backend parameters. Other
// parameters are preserved verbatim.
func injectPipelineParallelParameter(items []string, flag string, pp int) []string {
	out := make([]string, 0, len(items)+1)
	for _, item := range items {
		if arg, ok, err := parseBackendParameterArg(item); err == nil && ok {
			switch strings.TrimSpace(arg.Key) {
			case "pipeline-parallel-size", "pp-size", "pp":
				log.Warningf("drop backend parameter %q: overridden by deployment pipeline_parallel_size", item)
				continue
			}
		}
		out = append(out, item)
	}
	return append(out, fmt.Sprintf("--%s %d", flag, pp))
}

// appendDistributedEnv adds the environment shared by every multi-host member.
//
// VLLM_HOST_IP is emitted only when the pod IP is already known (restart or
// upgrade of an existing member). At first create the pod IP does not exist
// yet, so the entrypoint script resolves the node's own IP at container start;
// it must never fall back to the leader/master address.
func appendDistributedEnv(envs []*commonapi.ContainerKeyValue, llm *models.SLLM) []*commonapi.ContainerKeyValue {
	if llm == nil || !llm.IsDistributedMember() {
		return envs
	}
	envs = append(envs,
		&commonapi.ContainerKeyValue{Key: "NCCL_SOCKET_IFNAME", Value: distSocketIfName},
		&commonapi.ContainerKeyValue{Key: "GLOO_SOCKET_IFNAME", Value: distSocketIfName},
	)
	if hostIP := strings.TrimSpace(llm.LLMIp); hostIP != "" {
		envs = append(envs, &commonapi.ContainerKeyValue{Key: "VLLM_HOST_IP", Value: hostIP})
	}
	return envs
}

// appendVLLMDPEnv adds the vLLM data-parallel coordinator environment.
// vLLM engines read VLLM_DP_MASTER_IP/VLLM_DP_MASTER_PORT to find the DP
// coordinator; without them every engine waits forever for the READY message.
// On a hostlocal network followers connect through the leader host's access IP
// and host-mapped port, so those values come from the persisted dist fields.
func appendVLLMDPEnv(envs []*commonapi.ContainerKeyValue, llm *models.SLLM) []*commonapi.ContainerKeyValue {
	if llm == nil || !llm.IsDistributedMember() {
		return envs
	}
	if addr := distMasterAddrLiteral(llm); addr != "" {
		envs = append(envs, &commonapi.ContainerKeyValue{Key: "VLLM_DP_MASTER_IP", Value: addr})
	}
	envs = append(envs, &commonapi.ContainerKeyValue{
		Key:   "VLLM_DP_MASTER_PORT",
		Value: fmt.Sprintf("%d", distMasterPort(llm)),
	})
	return envs
}

// vllmDPEnvScriptPrefix fills the DP coordinator env at container start when
// it is not already set. The leader does not know its pod IP at create time,
// so it resolves its own address here; the port defaults to the in-pod master
// port (followers already carry the leader's host-mapped port as env).
func vllmDPEnvScriptPrefix(llm *models.SLLM) string {
	if llm == nil || !llm.IsDistributedMember() {
		return ""
	}
	return strings.Join([]string{
		`export VLLM_DP_MASTER_IP="${VLLM_DP_MASTER_IP:-$(hostname -I | awk '{print $1}')}"`,
		fmt.Sprintf(`export VLLM_DP_MASTER_PORT="${VLLM_DP_MASTER_PORT:-%d}"`, distMasterPort(llm)),
	}, "\n")
}

// distributedStartScriptPrefix resolves per-node values at container start:
//   - NCCL/GLOO socket interface: keep the configured one when the interface
//     exists, otherwise fall back to the default-route interface. This covers
//     hostlocal / bonded / non-eth0 networks without hardcoded assumptions.
//   - VLLM_HOST_IP from the chosen interface when not already known, so
//     multi-NIC hosts advertise the address of the socket interface instead of
//     the first one hostname(1) reports. The ip(8) expression requires iproute2
//     in the image; when missing, falls back to the previous behavior.
func distributedStartScriptPrefix() string {
	return strings.Join([]string{
		`if [ ! -e "/sys/class/net/${NCCL_SOCKET_IFNAME:-eth0}" ]; then ` +
			`dev="$(awk '$2=="00000000" {print $1; exit}' /proc/net/route 2>/dev/null)"; ` +
			`if [ -n "$dev" ]; then export NCCL_SOCKET_IFNAME="$dev" GLOO_SOCKET_IFNAME="$dev"; fi; ` +
			`fi`,
		`export VLLM_HOST_IP="${VLLM_HOST_IP:-$(ip -4 addr show dev "${NCCL_SOCKET_IFNAME:-eth0}" 2>/dev/null | awk '/inet /{print $2}' | cut -d/ -f1 | head -1)}"`,
		`[ -z "$VLLM_HOST_IP" ] && export VLLM_HOST_IP="${VLLM_HOST_IP:-$(hostname -I | awk '{print $1}')}"`,
	}, "\n")
}

func distMasterAddrShellExpr(llm *models.SLLM) string {
	if addr := distMasterAddrLiteral(llm); addr != "" {
		return shellQuoteSingle(addr)
	}
	// Leader may be built before LLMIp is known; resolve at container start.
	return `"$(hostname -I | awk '{print $1}')"`
}

func distMasterAddrLiteral(llm *models.SLLM) string {
	if llm == nil {
		return ""
	}
	addr := strings.TrimSpace(llm.DistMasterAddr)
	if addr == "" {
		addr = strings.TrimSpace(llm.LLMIp)
	}
	return addr
}

func distInitAddrShellExpr(llm *models.SLLM, port int) string {
	if addr := distMasterAddrLiteral(llm); addr != "" {
		return shellQuoteSingle(fmt.Sprintf("%s:%d", addr, port))
	}
	return fmt.Sprintf(`"$(hostname -I | awk '{print $1}'):%d"`, port)
}

// distMasterPort returns the port used in --master-port / --dist-init-addr.
// The leader binds the in-pod port; a follower on a hostlocal network connects
// through the leader host's mapped port, persisted in DistMasterPort.
func distMasterPort(llm *models.SLLM) int {
	if llm == nil {
		return api.LLM_DIST_MASTER_PORT
	}
	if llm.DistMasterPort > 0 {
		return llm.DistMasterPort
	}
	return api.LLMDistMasterPort(llm.DistGroup)
}

func distDPRPCPort(llm *models.SLLM) int {
	if llm == nil {
		return api.LLM_DIST_DP_RPC_PORT
	}
	if llm.DistDPRPCPort > 0 {
		return llm.DistDPRPCPort
	}
	return api.LLMDistDPRPCPort(llm.DistGroup)
}

func buildVLLMDistributedFlags(llm *models.SLLM, sku *models.SLLMSku, backendParameters []string) (tensorParallelSize int, extraFlags []string, shape scheduler.MultinodeShape, err error) {
	localGPUs := llmLocalGPUCount(llm, sku)
	tensorParallelSize = localGPUs
	user := scheduler.ParseUserParallelism(backendParameters)
	if llm == nil || !llm.IsDistributedMember() {
		if err := scheduler.ValidateSingleNodePipelineParallel(localGPUs, user.PP); err != nil {
			return localGPUs, nil, "", err
		}
		if user.PP != nil && *user.PP > 1 && user.TP == nil {
			// TP derives as gpus/pp so the world size fills the node exactly;
			// an explicit user TP is left untouched and wins on the command
			// line (backend parameters are appended after the base flags).
			tensorParallelSize = localGPUs / *user.PP
		}
		return tensorParallelSize, nil, "", nil
	}
	if llm.DistNnodes < 2 {
		return localGPUs, nil, "", fmt.Errorf("distributed member %s has dist_nnodes %d, want >= 2", llm.Id, llm.DistNnodes)
	}

	gpuPerNode := llmDistributedGPUPerNode(llm, sku)
	// The multi-node layout must match what the planner derived from the same
	// inputs, otherwise the planned layout and the emitted flags diverge.
	parallelMode := deploymentParallelMode(llm)
	scheduler.ApplyParallelMode(&user, parallelMode, gpuPerNode)
	scheduler.ApplyDefaultMultinodeParallelism(&user, gpuPerNode)
	topo, err := scheduler.CalMultinodeTopology(gpuPerNode, llm.DistRank, user)
	if err != nil {
		return localGPUs, nil, "", err
	}
	tensorParallelSize = topo.TP
	shape = topo.Shape
	masterExpr := distMasterAddrShellExpr(llm)

	flags := []string{"--distributed-executor-backend mp"}
	switch topo.Shape {
	case scheduler.ShapeDPOnly:
		flags = append(flags,
			fmt.Sprintf("--data-parallel-size %d", topo.DP),
			fmt.Sprintf("--data-parallel-size-local %d", topo.DPL),
			fmt.Sprintf("--data-parallel-address %s", masterExpr),
			fmt.Sprintf("--data-parallel-rpc-port %d", distDPRPCPort(llm)),
		)
		if topo.StartRank > 0 {
			flags = append(flags, fmt.Sprintf("--data-parallel-start-rank %d", topo.StartRank))
		}
	case scheduler.ShapeMPOnly:
		flags = append(flags,
			fmt.Sprintf("--nnodes %d", topo.NNodes),
			fmt.Sprintf("--node-rank %d", topo.NodeRank),
			fmt.Sprintf("--master-addr %s", masterExpr),
			fmt.Sprintf("--master-port %d", distMasterPort(llm)),
		)
	default: // nested
		flags = append(flags,
			fmt.Sprintf("--nnodes %d", topo.NNodes),
			fmt.Sprintf("--node-rank %d", topo.NodeRank),
			fmt.Sprintf("--master-addr %s", masterExpr),
			fmt.Sprintf("--master-port %d", distMasterPort(llm)),
			fmt.Sprintf("--data-parallel-size %d", topo.DP),
			fmt.Sprintf("--data-parallel-size-local %d", topo.DPL),
			fmt.Sprintf("--data-parallel-address %s", masterExpr),
			fmt.Sprintf("--data-parallel-rpc-port %d", distDPRPCPort(llm)),
		)
		if topo.StartRank > 0 {
			flags = append(flags, fmt.Sprintf("--data-parallel-start-rank %d", topo.StartRank))
		}
	}
	if topo.PP > 1 {
		flags = append(flags, fmt.Sprintf("--pipeline-parallel-size %d", topo.PP))
	}
	if topo.PCP > 1 {
		flags = append(flags, fmt.Sprintf("--prefill-context-parallel-size %d", topo.PCP))
	}
	if llm.IsDistFollower() {
		flags = append(flags, "--headless")
	}
	return tensorParallelSize, compactFlags(flags), shape, nil
}

func compactFlags(flags []string) []string {
	out := make([]string, 0, len(flags))
	for _, f := range flags {
		f = strings.TrimSpace(f)
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

func buildSGLangDistributedFlags(llm *models.SLLM, sku *models.SLLMSku, backendParameters []string) (tpSize int, extraFlags []string, err error) {
	tpSize = llmLocalGPUCount(llm, sku)
	user := scheduler.ParseUserParallelism(backendParameters)
	if llm == nil || !llm.IsDistributedMember() {
		if err := scheduler.ValidateSingleNodePipelineParallel(tpSize, user.PP); err != nil {
			return tpSize, nil, err
		}
		if user.PP != nil && *user.PP > 1 && user.TP == nil {
			// TP derives as gpus/pp so the world size fills the node exactly;
			// an explicit user TP is left untouched and wins on the command
			// line (backend parameters are appended after the base flags).
			tpSize = tpSize / *user.PP
		}
		return tpSize, nil, nil
	}
	if llm.DistNnodes < 2 {
		return tpSize, nil, fmt.Errorf("distributed member %s has dist_nnodes %d, want >= 2", llm.Id, llm.DistNnodes)
	}
	// Homogeneous equal GPUs: tp = per-node GPUs, pp = nnodes
	// (GPUStack cal_distributed_parallelism_arguments). A user-pinned --pp-size
	// is respected instead of being overridden by the node count, but must not
	// exceed the node count: tp is fixed to the per-node GPU count, so a larger
	// pp would make the world size exceed the GPUs available in the group.
	// The planner validates the same rule, so both layers reject the same
	// layouts with the same diagnostics.
	topo, err := scheduler.ValidateSGLangMultinodeTopology(llmDistributedGPUPerNode(llm, sku), user)
	if err != nil {
		return tpSize, nil, fmt.Errorf("distributed member %s: %v", llm.Id, err)
	}
	return tpSize, []string{
		fmt.Sprintf("--pp-size %d", topo.PP),
		fmt.Sprintf("--nnodes %d", topo.NNodes),
		fmt.Sprintf("--node-rank %d", llm.DistRank),
		fmt.Sprintf("--dist-init-addr %s", distInitAddrShellExpr(llm, distMasterPort(llm))),
	}, nil
}

// protectedDistributedVLLMKeys are managed by the planner only for multi-host
// members. Single-node vLLM deployments may still set DP/PP themselves.
func protectedDistributedVLLMKeys() map[string]struct{} {
	return map[string]struct{}{
		"distributed-executor-backend":  {},
		"nnodes":                        {},
		"node-rank":                     {},
		"master-addr":                   {},
		"master-port":                   {},
		"headless":                      {},
		"data-parallel-address":         {},
		"data-parallel-rpc-port":        {},
		"data-parallel-start-rank":      {},
		"data-parallel-size":            {},
		"data-parallel-size-local":      {},
		"tensor-parallel-size":          {},
		"pipeline-parallel-size":        {},
		"prefill-context-parallel-size": {},
	}
}

// protectedDistributedSGLangKeys are managed by the planner only for multi-host
// members.
func protectedDistributedSGLangKeys() map[string]struct{} {
	return map[string]struct{}{
		"nnodes":         {},
		"node-rank":      {},
		"dist-init-addr": {},
		"pp-size":        {},
	}
}

// splitDistributedBackendParameters separates the user's raw parameters from
// the ones used to build the final command. Topology derivation must see the
// raw flags (a user-pinned --pp-size or --data-parallel-size decides the
// layout), while the command drops every planner-managed flag so the derived
// values are the only ones left.
func splitDistributedBackendParameters(items []string, llm *models.SLLM, protected map[string]struct{}) (topology []string, command []string) {
	topology = items
	command = items
	if llm != nil && llm.IsDistributedMember() {
		command = filterDistributedBackendParameters(items, protected)
	}
	return topology, command
}

// filterDistributedBackendParameters drops planner-managed flags from a
// multi-host member's backend parameters. Single-node deployments are not
// filtered, so DP/PP remain available to them.
func filterDistributedBackendParameters(items []string, protected map[string]struct{}) []string {
	if len(items) == 0 {
		return items
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		arg, ok, err := parseBackendParameterArg(item)
		if err == nil && ok {
			if _, blocked := protected[arg.Key]; blocked {
				log.Warningf("drop backend parameter %q: managed by distributed inference", item)
				continue
			}
		}
		out = append(out, item)
	}
	return out
}

func filterVllmCustomizedArgsForDistributed(args []*api.VllmCustomizedArg, protected map[string]struct{}) []*api.VllmCustomizedArg {
	if len(args) == 0 {
		return args
	}
	out := make([]*api.VllmCustomizedArg, 0, len(args))
	for _, arg := range args {
		if arg == nil {
			continue
		}
		if _, blocked := protected[strings.TrimSpace(arg.Key)]; blocked {
			log.Warningf("drop vllm customized arg %q: managed by distributed inference", arg.Key)
			continue
		}
		out = append(out, arg)
	}
	return out
}

func filterSGLangCustomizedArgsForDistributed(args []*api.SGLangCustomizedArg, protected map[string]struct{}) []*api.SGLangCustomizedArg {
	if len(args) == 0 {
		return args
	}
	out := make([]*api.SGLangCustomizedArg, 0, len(args))
	for _, arg := range args {
		if arg == nil {
			continue
		}
		if _, blocked := protected[strings.TrimSpace(arg.Key)]; blocked {
			log.Warningf("drop sglang customized arg %q: managed by distributed inference", arg.Key)
			continue
		}
		out = append(out, arg)
	}
	return out
}
