package models

import (
	"context"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/scheduler"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

// ListFreeGPUInventoryByModel returns unused GPUs of the given model, grouped by host.
func ListFreeGPUInventoryByModel(ctx context.Context, userCred mcclient.TokenCredential, model string) ([]scheduler.HostGPUInventory, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return nil, errors.Error("gpu model is empty")
	}
	s := auth.GetSession(ctx, userCred, "")
	unused := true
	params := jsonutils.Marshal(computeapi.IsolatedDeviceListInput{
		Unused: &unused,
		Model:  []string{model},
	}).(*jsonutils.JSONDict)
	params.Set("limit", jsonutils.NewInt(2048))
	params.Set("details", jsonutils.JSONTrue)
	params.Set("scope", jsonutils.NewString("system"))

	result, err := compute.IsolatedDevices.List(s, params)
	if err != nil {
		return nil, errors.Wrap(err, "list unused isolated_devices")
	}

	byHost := make(map[string]*scheduler.HostGPUInventory)
	order := make([]string, 0)
	for i := range result.Data {
		hostId, _ := result.Data[i].GetString("host_id")
		if hostId == "" {
			continue
		}
		hostName, _ := result.Data[i].GetString("host")
		inv, ok := byHost[hostId]
		if !ok {
			inv = &scheduler.HostGPUInventory{HostId: hostId, HostName: hostName, Online: true}
			byHost[hostId] = inv
			order = append(order, hostId)
		}
		inv.FreeGPUs++
		if devModel, _ := result.Data[i].GetString("model"); devModel != "" {
			inv.GpuModel = devModel
		}
		if memObj, err := result.Data[i].Get("memory_size"); err == nil {
			if mem, err := memObj.Int(); err == nil && mem > 0 && (inv.MinVRAMMb == 0 || int(mem) < inv.MinVRAMMb) {
				inv.MinVRAMMb = int(mem)
			}
		}
	}
	out := make([]scheduler.HostGPUInventory, 0, len(order))
	for _, id := range order {
		out = append(out, *byHost[id])
	}
	enrichHostInventory(ctx, userCred, out)
	return out, nil
}

// enrichHostInventory fills online state, access IP and a fallback host name
// from the compute host rows. Hosts missing from the lookup keep Online=true so
// a permission/partial-list failure does not hide otherwise valid GPUs.
func enrichHostInventory(ctx context.Context, userCred mcclient.TokenCredential, hosts []scheduler.HostGPUInventory) {
	if len(hosts) == 0 {
		return
	}
	ids := make([]string, 0, len(hosts))
	for i := range hosts {
		ids = append(ids, hosts[i].HostId)
	}
	s := auth.GetSession(ctx, userCred, "")
	params := jsonutils.NewDict()
	params.Set("id", jsonutils.NewStringArray(ids))
	params.Set("limit", jsonutils.NewInt(int64(len(ids)+1)))
	params.Set("details", jsonutils.JSONTrue)
	params.Set("scope", jsonutils.NewString("system"))
	result, err := compute.Hosts.List(s, params)
	if err != nil {
		return
	}
	type hostRow struct {
		Id         string
		Name       string
		AccessIp   string
		HostStatus string
	}
	byId := make(map[string]hostRow, len(result.Data))
	for i := range result.Data {
		row := hostRow{}
		if err := result.Data[i].Unmarshal(&row); err != nil {
			continue
		}
		byId[row.Id] = row
	}
	for i := range hosts {
		row, ok := byId[hosts[i].HostId]
		if !ok {
			log.Warningf("enrichHostInventory: host %s not found in hosts list; keeping optimistic online state", hosts[i].HostId)
			continue
		}
		if row.Name != "" {
			hosts[i].HostName = row.Name
		}
		hosts[i].HostIp = row.AccessIp
		if row.HostStatus == computeapi.HOST_OFFLINE {
			hosts[i].Online = false
		}
	}
}

func skuDeviceTemplate(sku *SLLMSku) (api.Device, bool) {
	if sku == nil || sku.Devices == nil || len(*sku.Devices) == 0 {
		return api.Device{}, false
	}
	tmpl := (*sku.Devices)[0]
	normalizeLLMSkuDevice(&tmpl)
	return tmpl, true
}

func skuForceSingleGPUPerNode(sku *SLLMSku) bool {
	if sku == nil || sku.Devices == nil {
		return false
	}
	for i := range *sku.Devices {
		dev := (*sku.Devices)[i]
		normalizeLLMSkuDevice(&dev)
		if computeapi.IsMthreadsHami(dev.Vendor, dev.SharingMode) {
			return true
		}
	}
	return false
}

// PlanDeploymentReplica builds a single- or multi-host placement for one logical replica.
func PlanDeploymentReplica(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	deploy *SLLMDeployment,
	sku *SLLMSku,
) (*scheduler.ReplicaPlan, error) {
	return PlanDeploymentReplicaWithReserved(ctx, userCred, deploy, sku, nil)
}

// PlanDeploymentReplicaWithReserved is PlanDeploymentReplica minus GPUs already
// promised to earlier replicas in the same reconcile pass. Without the
// reservation, concurrent scale-up plans can pick the same free GPUs again
// because the pods do not exist yet.
func PlanDeploymentReplicaWithReserved(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	deploy *SLLMDeployment,
	sku *SLLMSku,
	reserved map[string]int,
) (*scheduler.ReplicaPlan, error) {
	if sku == nil || sku.Devices == nil || len(*sku.Devices) == 0 {
		return nil, errors.Error("sku has no GPU devices")
	}
	tmpl, ok := skuDeviceTemplate(sku)
	if !ok || strings.TrimSpace(tmpl.Model) == "" {
		return nil, errors.Error("sku GPU model is empty")
	}
	hosts, err := ListFreeGPUInventoryByModel(ctx, userCred, tmpl.Model)
	if err != nil {
		return nil, err
	}
	for i := range hosts {
		if n, ok := reserved[hosts[i].HostId]; ok {
			hosts[i].FreeGPUs -= n
			if hosts[i].FreeGPUs < 0 {
				hosts[i].FreeGPUs = 0
			}
		}
	}
	distributed := deploy != nil && deploy.DistributedInference != nil && *deploy.DistributedInference
	placement := ""
	preferHosts := []string(nil)
	nnodes := 0
	if deploy != nil {
		placement = deploy.PlacementStrategy
		preferHosts = GetDeploymentPreferHosts(deploy, sku)
		// A stale nnodes on a deployment with distributed inference disabled
		// must not force a multi-host layout.
		if distributed {
			nnodes = deploy.Nnodes
		}
	}
	return scheduler.PlanReplica(scheduler.PlanOptions{
		TotalGPUs:             len(*sku.Devices),
		DeviceTemplate:        tmpl,
		PreferHosts:           preferHosts,
		PlacementStrategy:     placement,
		DistributedInference:  distributed,
		ForceSingleGPUPerNode: skuForceSingleGPUPerNode(sku),
		ForceNNodes:           nnodes,
		Backend:               sku.LLMType,
		BackendParameters:     sku.BackendParameters,
		PipelineParallelSize:  deploymentPipelineParallelSize(deploy),
		ParallelMode:          deploymentParallelMode(deploy),
		Hosts:                 hosts,
	})
}

// deploymentPipelineParallelSize returns the deployment-level pipeline pin
// (0 = unset). Nil-safe for callers that plan without a deployment row.
func deploymentPipelineParallelSize(deploy *SLLMDeployment) int {
	if deploy == nil {
		return 0
	}
	return deploy.PipelineParallelSize
}

// deploymentParallelMode returns the deployment-level parallel mode (default
// "pp"). Nil-safe for callers that plan without a deployment row.
func deploymentParallelMode(deploy *SLLMDeployment) string {
	if deploy == nil {
		return api.LLM_DEPLOYMENT_PARALLEL_MODE_PP
	}
	return api.NormalizeLLMDeploymentParallelMode(deploy.ParallelMode)
}

// ReservePlanGPUs records the GPUs promised to one replica plan.
func ReservePlanGPUs(reserved map[string]int, plan *scheduler.ReplicaPlan) {
	if plan == nil {
		return
	}
	for _, m := range plan.Members {
		reserved[m.HostId] += len(m.Devices)
	}
}
