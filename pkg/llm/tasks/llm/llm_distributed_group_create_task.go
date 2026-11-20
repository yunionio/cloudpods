package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/llm/scheduler"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// LLMDistributedGroupCreateTask creates one logical multi-host replica:
// leader (auto_start=false) → wait IP → followers → start leader → start followers → wait leader HTTP.
type LLMDistributedGroupCreateTask struct {
	taskman.STask
}

type distributedGroupCreateParams struct {
	DeploymentId string                 `json:"deployment_id"`
	Group        int                    `json:"group"`
	Plan         *scheduler.ReplicaPlan `json:"plan"`
	LLMSpec      *api.LLMSpec           `json:"llm_spec"`
}

func init() {
	taskman.RegisterTask(LLMDistributedGroupCreateTask{})
}

func (task *LLMDistributedGroupCreateTask) taskFailed(ctx context.Context, deploy *models.SLLMDeployment, err error) {
	log.Errorf("LLMDistributedGroupCreateTask failed for %s: %s", deploy.Name, err)
	// Keep the members created so far: failed pods, container state and
	// placement are the debugging scene. Automatic reconcile preserves broken
	// distributed groups; clean up by scaling the deployment down (or deleting
	// it) once the cause is understood.
	db.OpsLog.LogEvent(deploy, db.ACT_CREATE, err, task.UserCred)
	models.RecordDeploymentGroupFailure(ctx, task.UserCred, deploy, err)
	task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (task *LLMDistributedGroupCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	deploy := obj.(*models.SLLMDeployment)
	params := distributedGroupCreateParams{}
	if err := task.GetParams().Unmarshal(&params); err != nil {
		task.taskFailed(ctx, deploy, errors.Wrap(err, "unmarshal params"))
		return
	}
	if params.Plan == nil || len(params.Plan.Members) < 2 {
		task.taskFailed(ctx, deploy, errors.Error("distributed plan requires at least 2 members"))
		return
	}

	skuObj, err := models.GetLLMSkuManager().FetchById(deploy.LLMSkuId)
	if err != nil {
		task.taskFailed(ctx, deploy, errors.Wrap(err, "fetch sku"))
		return
	}
	sku := skuObj.(*models.SLLMSku)
	if deploy.Nets == nil || len(*deploy.Nets) == 0 {
		task.taskFailed(ctx, deploy, errors.Error("deployment has no nets template"))
		return
	}

	leaderMember := params.Plan.Members[0]
	leaderName := models.DistLeaderInstanceName(deploy.Name, params.Group)
	leaderInput := buildDistributedMemberCreateInput(deploy, sku, params, leaderMember, "", api.LLM_DIST_ROLE_LEADER)
	leader, err := createLLMInstanceSync(ctx, task.UserCred, leaderName, leaderInput)
	if err != nil {
		task.taskFailed(ctx, deploy, errors.Wrapf(err, "create leader %s", leaderName))
		return
	}
	// Persist immediately so a later failure can roll the leader back.
	if err := task.persistGroupIDs(leader.Id, nil); err != nil {
		log.Warningf("persist distributed leader id: %s", err)
	}

	if err := waitLLMHasIP(ctx, leader, 600); err != nil {
		task.taskFailed(ctx, deploy, errors.Wrap(err, "wait leader IP"))
		return
	}
	leader, err = refetchLLM(leader.Id)
	if err != nil {
		task.taskFailed(ctx, deploy, errors.Wrap(err, "refetch leader"))
		return
	}
	if leader.LLMIp == "" {
		task.taskFailed(ctx, deploy, errors.Error("leader has empty LLMIp"))
		return
	}

	masterAddr, masterPort, dpRPCPort, err := task.resolveLeaderAddress(ctx, deploy, sku, params, leader)
	if err != nil {
		task.taskFailed(ctx, deploy, err)
		return
	}

	followerIds := make([]string, 0, len(params.Plan.Members)-1)
	members := []*models.SLLM{leader}
	for _, member := range params.Plan.Members[1:] {
		fname := models.DistFollowerInstanceName(deploy.Name, params.Group, member.Rank)
		fInput := buildDistributedMemberCreateInput(deploy, sku, params, member, leader.Id, api.LLM_DIST_ROLE_FOLLOWER)
		fInput.DistMasterAddr = masterAddr
		fInput.DistMasterPort = masterPort
		fInput.DistDPRPCPort = dpRPCPort
		follower, err := createLLMInstanceSync(ctx, task.UserCred, fname, fInput)
		if err != nil {
			task.taskFailed(ctx, deploy, errors.Wrapf(err, "create follower %s", fname))
			return
		}
		members = append(members, follower)
		followerIds = append(followerIds, follower.Id)
		if err := task.persistGroupIDs(leader.Id, followerIds); err != nil {
			log.Warningf("persist distributed follower ids: %s", err)
		}
	}

	if err := task.verifyMemberHosts(ctx, deploy, params.Plan, members); err != nil {
		task.taskFailed(ctx, deploy, err)
		return
	}

	if err := task.persistGroupIDs(leader.Id, followerIds); err != nil {
		log.Warningf("persist distributed group ids: %s", err)
	}

	// Deployments always auto-start: members were created with auto_start=false
	// so the whole group exists first, then the group task starts them
	// leader-first.
	task.SetStage("OnLeaderStarted", nil)
	if err := leader.StartStartTask(ctx, task.UserCred, task.GetTaskId()); err != nil {
		task.taskFailed(ctx, deploy, errors.Wrap(err, "start leader"))
		return
	}
}

// verifyMemberHosts checks the actual placement of a freshly created group.
// PreferHost is a scheduling hint, so a member landing on a different host than
// the planner picked is only logged. Two ranks sharing one host, however, breaks
// multi-host inference and must fail the group so reconcile can retry.
func (task *LLMDistributedGroupCreateTask) verifyMemberHosts(ctx context.Context, deploy *models.SLLMDeployment, plan *scheduler.ReplicaPlan, members []*models.SLLM) error {
	cmpIds := make([]string, 0, len(members))
	for _, m := range members {
		if m.CmpId != "" {
			cmpIds = append(cmpIds, m.CmpId)
		}
	}
	if len(cmpIds) == 0 {
		return nil
	}
	hostByCmp := models.FetchLLMHostIdsByCmpIds(ctx, cmpIds)
	planned := make(map[int]string, len(plan.Members))
	for _, m := range plan.Members {
		planned[m.Rank] = m.HostId
	}
	rankByHost := make(map[string]int, len(members))
	for _, m := range members {
		hostId, ok := hostByCmp[m.CmpId]
		if !ok || hostId == "" {
			continue
		}
		if prevRank, dup := rankByHost[hostId]; dup {
			return errors.Errorf("distributed group %d ranks %d and %d landed on the same host %s",
				m.DistGroup, prevRank, m.DistRank, hostId)
		}
		rankByHost[hostId] = m.DistRank
		if want := planned[m.DistRank]; want != "" && want != hostId {
			log.Warningf("distributed group %d rank %d landed on host %s, plan wanted %s",
				m.DistGroup, m.DistRank, hostId, want)
			db.OpsLog.LogEvent(deploy, db.ACT_UPDATE, errors.Errorf(
				"group %d rank %d placed on host %s instead of planned %s",
				m.DistGroup, m.DistRank, hostId, want), task.UserCred)
		}
	}
	return nil
}

func (task *LLMDistributedGroupCreateTask) persistGroupIDs(leaderId string, followerIds []string) error {
	extra := jsonutils.NewDict()
	extra.Set("leader_id", jsonutils.NewString(leaderId))
	extra.Set("follower_ids", jsonutils.NewStringArray(followerIds))
	return task.SaveParams(extra)
}

func (task *LLMDistributedGroupCreateTask) OnLeaderStarted(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	deploy := obj.(*models.SLLMDeployment)
	followerIds, _ := task.GetParams().GetArray("follower_ids")
	task.SetStage("OnFollowersStarted", nil)
	if len(followerIds) == 0 {
		task.OnFollowersStarted(ctx, deploy, nil)
		return
	}
	started := 0
	for _, idObj := range followerIds {
		id, _ := idObj.GetString()
		llm, err := refetchLLM(id)
		if err != nil {
			task.taskFailed(ctx, deploy, errors.Wrapf(err, "fetch follower %s", id))
			return
		}
		if err := llm.StartStartTask(ctx, task.UserCred, task.GetTaskId()); err != nil {
			task.taskFailed(ctx, deploy, errors.Wrapf(err, "start follower %s", id))
			return
		}
		started++
	}
	if started == 0 {
		task.taskFailed(ctx, deploy, errors.Error("no followers started"))
	}
}

func (task *LLMDistributedGroupCreateTask) OnLeaderStartedFailed(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	task.taskFailed(ctx, obj.(*models.SLLMDeployment), fmt.Errorf("leader start failed: %s", body))
}

func (task *LLMDistributedGroupCreateTask) OnFollowersStarted(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	deploy := obj.(*models.SLLMDeployment)
	leaderId, _ := task.GetParams().GetString("leader_id")
	leader, err := refetchLLM(leaderId)
	if err != nil {
		task.taskFailed(ctx, deploy, errors.Wrap(err, "fetch leader for ready wait"))
		return
	}
	if _, err := leader.WaitServiceReady(ctx, task.UserCred, 0); err != nil {
		task.taskFailed(ctx, deploy, errors.Wrap(err, "WaitServiceReady leader"))
		return
	}
	if err := leader.SetStatus(ctx, task.UserCred, api.LLM_STATUS_RUNNING, "distributed leader ready"); err != nil {
		log.Warningf("set leader running: %s", err)
	}

	followerIds, _ := task.GetParams().GetArray("follower_ids")
	for _, idObj := range followerIds {
		id, _ := idObj.GetString()
		follower, err := refetchLLM(id)
		if err != nil {
			task.taskFailed(ctx, deploy, errors.Wrapf(err, "fetch follower %s", id))
			return
		}
		if _, err := follower.WaitServiceReady(ctx, task.UserCred, 0); err != nil {
			task.taskFailed(ctx, deploy, errors.Wrapf(err, "WaitServiceReady follower %s", id))
			return
		}
		if err := follower.SetStatus(ctx, task.UserCred, api.LLM_STATUS_RUNNING, "distributed follower ready"); err != nil {
			log.Warningf("set follower running: %s", err)
		}
	}
	deploy.ResetReconcileFailures()
	task.SetStageComplete(ctx, nil)
}

func (task *LLMDistributedGroupCreateTask) OnFollowersStartedFailed(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	task.taskFailed(ctx, obj.(*models.SLLMDeployment), fmt.Errorf("follower start failed: %s", body))
}

func buildDistributedMemberCreateInput(
	deploy *models.SLLMDeployment,
	sku *models.SLLMSku,
	params distributedGroupCreateParams,
	member scheduler.ReplicaMemberPlan,
	leaderId string,
	role string,
) api.LLMCreateInput {
	devices := member.Devices
	return api.LLMCreateInput{
		LLMBaseCreateInput: api.LLMBaseCreateInput{
			AutoStart:  false,
			Nets:       []*computeapi.NetworkConfig(*deploy.Nets),
			PreferHost: member.HostId,
		},
		LLMSkuId:        deploy.LLMSkuId,
		LLMImageId:      sku.GetLLMImageId(),
		LLMDeploymentId: deploy.Id,
		LLMSpec:         params.LLMSpec,
		HostPaths:       deploy.HostPaths,
		Devices:         &devices,
		DistRole:        role,
		DistRank:        member.Rank,
		DistNnodes:      params.Plan.NNodes,
		DistLeaderId:    leaderId,
		DistGroup:       params.Group,
	}
}

func createLLMInstanceSync(ctx context.Context, userCred mcclient.TokenCredential, name string, input api.LLMCreateInput) (*models.SLLM, error) {
	createCtx := context.WithValue(ctx, appctx.APP_CONTEXT_KEY_AUTH_TOKEN, userCred)
	handler := db.NewModelHandler(models.GetLLMManager())
	params := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	params.Set("name", jsonutils.NewString(name))
	params.Remove("generate_name")
	resp, err := handler.Create(createCtx, jsonutils.NewDict(), params, nil)
	if err != nil {
		return nil, err
	}
	id, _ := resp.GetString("id")
	if id == "" {
		return nil, errors.Error("create llm response missing id")
	}
	return waitLLMCreateTaskSettled(ctx, id, 1800)
}

func refetchLLM(id string) (*models.SLLM, error) {
	obj, err := models.GetLLMManager().FetchById(id)
	if err != nil {
		return nil, err
	}
	return obj.(*models.SLLM), nil
}

func waitLLMHasIP(ctx context.Context, llm *models.SLLM, timeoutSecs int) error {
	deadline := time.Now().Add(time.Duration(timeoutSecs) * time.Second)
	for {
		cur, err := refetchLLM(llm.Id)
		if err != nil {
			return err
		}
		if cur.LLMIp != "" {
			*llm = *cur
			return nil
		}
		if cur.Status == api.LLM_STATUS_CREATE_FAIL {
			return fmt.Errorf("leader create failed before IP assignment")
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for leader IP")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func waitLLMCreateTaskSettled(ctx context.Context, id string, timeoutSecs int) (*models.SLLM, error) {
	deadline := time.Now().Add(time.Duration(timeoutSecs) * time.Second)
	for {
		llm, err := refetchLLM(id)
		if err != nil {
			return nil, err
		}
		switch llm.Status {
		case api.LLM_STATUS_CREATE_FAIL:
			return nil, fmt.Errorf("llm %s create failed", id)
		case api.LLM_STATUS_READY, api.LLM_STATUS_RUNNING, api.LLM_STATUS_PROBING:
			return llm, nil
		default:
			// creating / starting — keep waiting for CmpId+IP path
			if llm.CmpId != "" && llm.Status != "" {
				// After LLMCreateTask with AutoStart=false, status becomes server status (ready).
				if llm.LLMIp != "" || llm.Status == api.LLM_STATUS_READY || llm.Status == computeapi.VM_READY {
					return llm, nil
				}
			}
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for llm %s create settle (status=%s)", id, llm.Status)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// resolveLeaderAddress returns the address and ports followers use to reach the
// leader. On a VPC (guest) network the pod IP works directly. On a hostlocal
// network the pod IP only exists on its own host, so the leader host's access
// IP and the host-mapped ports are used instead. Only vLLM data-parallel
// layouts are supportable there: cross-host TP/PP needs NCCL on arbitrary
// ports, which a hostlocal bridge cannot carry.
func (task *LLMDistributedGroupCreateTask) resolveLeaderAddress(
	ctx context.Context,
	deploy *models.SLLMDeployment,
	sku *models.SLLMSku,
	params distributedGroupCreateParams,
	leader *models.SLLM,
) (addr string, masterPort, dpRPCPort int, err error) {
	if !models.IsHostLocalLLM(leader) {
		return leader.LLMIp, 0, 0, nil
	}
	if sku.LLMType != string(api.LLM_CONTAINER_VLLM) {
		return "", 0, 0, errors.Errorf(
			"hostlocal network only supports vLLM data-parallel (DP) multi-host replicas; use a VPC (guest) network for %s multi-node inference",
			sku.LLMType)
	}
	pipelineParallelSize := 0
	parallelMode := ""
	if deploy != nil {
		pipelineParallelSize = deploy.PipelineParallelSize
		parallelMode = deploy.ParallelMode
	}
	gpuPerNode := scheduler.GpuPerNodeFromPlan(params.Plan)
	topo, topoErr := scheduler.ValidateMultinodeTopology(
		gpuPerNode,
		scheduler.EffectiveParallelism(sku.BackendParameters, pipelineParallelSize, parallelMode, gpuPerNode),
	)
	if topoErr != nil {
		return "", 0, 0, errors.Wrap(topoErr, "validate distributed topology")
	}
	if topo.Shape != scheduler.ShapeDPOnly {
		return "", 0, 0, errors.Errorf(
			"hostlocal network only supports data-parallel (DP) layouts; the %q layout needs cross-host NCCL ports, use a VPC (guest) network instead",
			string(topo.Shape))
	}

	servers := models.FetchLLMServerMap(ctx, []string{leader.CmpId})
	server, ok := servers[leader.CmpId]
	if !ok {
		return "", 0, 0, errors.Errorf("fetch leader server %s for hostlocal address", leader.CmpId)
	}
	addr = strings.TrimSpace(server.HostAccessIp)
	if addr == "" {
		return "", 0, 0, errors.Errorf("hostlocal network: leader host of server %s has no access IP", leader.CmpId)
	}
	dpPodPort := api.LLMDistDPRPCPort(params.Group)
	dpRPCPort, ok = models.ServerHostMappedPort(&server, dpPodPort)
	if !ok {
		return "", 0, 0, errors.Errorf(
			"hostlocal network: leader has no host-mapped port for distributed port %d; ensure hosts run with --enable-container-cni-portmap and the host port range is available",
			dpPodPort)
	}
	masterPodPort := api.LLMDistMasterPort(params.Group)
	masterPort, ok = models.ServerHostMappedPort(&server, masterPodPort)
	if !ok {
		return "", 0, 0, errors.Errorf(
			"hostlocal network: leader has no host-mapped port for distributed port %d; ensure hosts run with --enable-container-cni-portmap and the host port range is available",
			masterPodPort)
	}
	return addr, masterPort, dpRPCPort, nil
}
