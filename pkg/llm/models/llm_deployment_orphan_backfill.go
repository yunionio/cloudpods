package models

import (
	"context"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"

	"yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/llm/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

var orphanLLMBackfillExcludedStatuses = []string{
	api.LLM_STATUS_DELETING,
	api.LLM_STATUS_DELETED,
	api.LLM_STATUS_START_DELETE,
}

// BackfillOrphanLLMDeployments creates one SLLMDeployment per orphan SLLM instance
// (no llm_deployment_id) and links the instance to it. Idempotent across restarts.
func BackfillOrphanLLMDeployments(ctx context.Context) error {
	if options.Options.IsSlaveNode {
		return nil
	}
	userCred := auth.AdminCredential()

	llms, err := fetchOrphanLLMsForBackfill()
	if err != nil {
		return errors.Wrap(err, "fetch orphan llms")
	}
	if len(llms) == 0 {
		return nil
	}

	log.Infof("BackfillOrphanLLMDeployments: found %d orphan llm instance(s)", len(llms))
	for i := range llms {
		llm := &llms[i]
		if err := GetLLMDeploymentManager().adoptOrphanLLM(ctx, userCred, llm); err != nil {
			log.Errorf("BackfillOrphanLLMDeployments: adopt llm %s (%s): %s", llm.Name, llm.Id, err)
			continue
		}
		log.Infof("BackfillOrphanLLMDeployments: adopted llm %s (%s) into deployment %s",
			llm.Name, llm.Id, llm.LLMDeploymentId)
	}
	return nil
}

func fetchOrphanLLMsForBackfill() ([]SLLM, error) {
	q := GetLLMManager().Query()
	q = q.IsNullOrEmpty("llm_deployment_id")
	q = q.IsNotEmpty("llm_sku_id")
	q = q.Equals("deleted", false)
	q = q.Equals("pending_deleted", false)
	q = q.NotIn("status", orphanLLMBackfillExcludedStatuses)

	var llms []SLLM
	if err := db.FetchModelObjects(GetLLMManager(), q, &llms); err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return llms, nil
}

func isOrphanLLMEligibleForBackfill(llm *SLLM) bool {
	if llm == nil {
		return false
	}
	if llm.LLMDeploymentId != "" || llm.LLMSkuId == "" {
		return false
	}
	if llm.Deleted || llm.PendingDeleted {
		return false
	}
	for _, status := range orphanLLMBackfillExcludedStatuses {
		if llm.Status == status {
			return false
		}
	}
	return true
}

func netsFromOrphanLLM(llm *SLLM) (*api.LLMDeploymentNets, error) {
	if llm == nil || llm.NetworkId == "" {
		return nil, errors.Error("missing network_id")
	}
	nets := api.LLMDeploymentNets{
		&computeapi.NetworkConfig{
			Index:   0,
			Network: llm.NetworkId,
			NetType: computeapi.TNetworkType(llm.NetworkType),
		},
	}
	return &nets, nil
}

func pickDeploymentNameForOrphanLLM(llm *SLLM, nameTaken func(string) bool) string {
	if llm == nil {
		return ""
	}
	candidates := []string{
		llm.Name,
		llm.Name + "-deploy",
		llm.Id + "-deploy",
		llm.Id,
	}
	for _, name := range candidates {
		if name == "" {
			continue
		}
		if nameTaken == nil || !nameTaken(name) {
			return name
		}
	}
	return llm.Id
}

func deploymentNameTakenInProject(projectId, name string) (bool, error) {
	if name == "" {
		return true, nil
	}
	cnt, err := GetLLMDeploymentManager().Query().
		Equals("name", name).
		Equals("tenant_id", projectId).
		Equals("deleted", false).
		CountWithError()
	if err != nil {
		return false, errors.Wrap(err, "count deployment by name")
	}
	return cnt > 0, nil
}

func deploymentNameForOrphanLLM(llm *SLLM) (string, error) {
	return pickDeploymentNameForOrphanLLM(llm, func(name string) bool {
		taken, err := deploymentNameTakenInProject(llm.ProjectId, name)
		if err != nil {
			log.Warningf("deploymentNameForOrphanLLM: check name %q: %s", name, err)
			return true
		}
		return taken
	}), nil
}

func (man *SLLMDeploymentManager) adoptOrphanLLM(ctx context.Context, userCred mcclient.TokenCredential, llm *SLLM) error {
	if !isOrphanLLMEligibleForBackfill(llm) {
		return errors.Error("llm is not eligible for orphan backfill")
	}

	skuObj, err := GetLLMSkuManager().FetchById(llm.LLMSkuId)
	if err != nil {
		return errors.Wrapf(err, "fetch LLMSku %s", llm.LLMSkuId)
	}
	sku := skuObj.(*SLLMSku)

	nets, err := netsFromOrphanLLM(llm)
	if err != nil {
		return errors.Wrapf(err, "llm %s", llm.Id)
	}

	name, err := deploymentNameForOrphanLLM(llm)
	if err != nil {
		return err
	}
	if name == "" {
		return errors.Errorf("cannot derive deployment name for llm %s", llm.Id)
	}

	restartOnError := true
	dep := &SLLMDeployment{}
	dep.SetModelManager(man, dep)
	dep.Name = name
	dep.LLMSkuId = llm.LLMSkuId
	dep.Replicas = 1
	dep.ReadyReplicas = 0
	dep.PlacementStrategy = api.LLM_MODEL_PLACEMENT_SPREAD
	dep.AccessPolicy = api.LLM_MODEL_ACCESS_AUTHED
	dep.AutoRegisterAiproxy = false
	dep.AutoStart = false
	dep.Nets = nets
	dep.RestartOnError = &restartOnError
	dep.ProjectId = llm.ProjectId
	dep.DomainId = llm.DomainId
	dep.ProjectSrc = string(apis.OWNER_SOURCE_LOCAL)
	dep.Enabled = tristate.True
	if hostPaths := GetEffectiveHostPaths(llm, sku); hostPaths != nil && !hostPaths.IsZero() {
		dep.HostPaths = hostPaths
	}

	if err := man.TableSpec().Insert(ctx, dep); err != nil {
		return errors.Wrap(err, "insert deployment")
	}

	if _, err := db.Update(llm, func() error {
		llm.LLMDeploymentId = dep.Id
		return nil
	}); err != nil {
		return errors.Wrap(err, "link llm to deployment")
	}

	if err := dep.SyncReadyReplicas(ctx, userCred); err != nil {
		return errors.Wrap(err, "SyncReadyReplicas")
	}
	return nil
}
