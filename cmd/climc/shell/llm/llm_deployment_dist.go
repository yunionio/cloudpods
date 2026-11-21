package llm

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/llm"
	options "yunion.io/x/onecloud/pkg/mcclient/options/llm"
)

func init() {
	shell.R(
		new(options.LLMDeploymentDistGroupsOptions),
		"llm-deployment-dist-groups",
		"Show distributed replica groups (leader/follower members) of a llm_deployment",
		doLLMDeploymentDistGroups,
	)
}

func doLLMDeploymentDistGroups(s *mcclient.ClientSession, args *options.LLMDeploymentDistGroupsOptions) error {
	ret, err := modules.LLMDeployments.Get(s, args.GetId(), nil)
	if err != nil {
		return err
	}
	details := api.LLMDeploymentDetails{}
	if err := ret.Unmarshal(&details); err != nil {
		return err
	}
	if len(details.DistGroups) == 0 {
		name, _ := ret.GetString("name")
		fmt.Printf("%s is not a distributed deployment (distributed_nnodes=%d)\n",
			name, details.DistributedNnodes)
		return nil
	}
	rows := make([]jsonutils.JSONObject, 0, 8)
	for _, g := range details.DistGroups {
		for _, m := range g.Members {
			rows = append(rows, jsonutils.Marshal(map[string]interface{}{
				"group":  g.Group,
				"ready":  g.Ready,
				"leader": g.Leader,
				"role":   m.Role,
				"rank":   m.Rank,
				"name":   m.Name,
				"status": m.Status,
				"llm_ip": m.LlmIp,
				"gpus":   m.Gpus,
				"host":   m.Host,
			}))
		}
	}
	shell.PrintList(&printutils.ListResult{Data: rows},
		[]string{"group", "ready", "leader", "role", "rank", "name", "status", "llm_ip", "gpus", "host"})
	return nil
}
