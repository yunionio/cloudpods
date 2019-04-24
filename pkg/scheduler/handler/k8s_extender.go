// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	gin "gopkg.in/gin-gonic/gin.v1"
	v1 "k8s.io/api/core/v1"
	schedulerapi "k8s.io/kubernetes/pkg/scheduler/api/v1"

	"yunion.io/x/log"

	k8spredicates "yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates/k8s"
	schedman "yunion.io/x/onecloud/pkg/scheduler/manager"
)

func InstallK8sSchedExtenderHandler(r *gin.Engine) {
	r.POST("/k8s/predicates", timer(k8sPredicatesHandler))
	//r.POST("/k8s/priorities", timer(k8sPrioritizeHandler))
}

func k8sPredicatesHandler(c *gin.Context) {
	if !schedman.IsReady() {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("Global scheduler not init"))
		return
	}

	var extenderArgs schedulerapi.ExtenderArgs
	var extenderFilterResult *schedulerapi.ExtenderFilterResult

	if err := json.NewDecoder(c.Request.Body).Decode(&extenderArgs); err != nil {
		extenderFilterResult = &schedulerapi.ExtenderFilterResult{
			Nodes:       nil,
			FailedNodes: nil,
			Error:       err.Error(),
		}
	} else {
		extenderFilterResult = k8sPredicateFunc(extenderArgs)
	}
	c.JSON(http.StatusOK, extenderFilterResult)
}

func k8sPredicateFunc(args schedulerapi.ExtenderArgs) *schedulerapi.ExtenderFilterResult {
	pod := args.Pod
	canSchedule := make([]v1.Node, 0, len(args.Nodes.Items))
	canNotSchedule := make(map[string]string)
	for _, node := range args.Nodes.Items {
		result, err := doK8sPredicates(pod, &node)
		if err != nil {
			canNotSchedule[node.Name] = fmt.Sprintf("%s: %v", node.Name, err)
		} else {
			if result {
				canSchedule = append(canSchedule, node)
			}
		}
	}
	return &schedulerapi.ExtenderFilterResult{
		Nodes: &v1.NodeList{
			Items: canSchedule,
		},
		FailedNodes: canNotSchedule,
		Error:       "",
	}
}

func doK8sPredicates(pod *v1.Pod, node *v1.Node) (bool, error) {
	hosts, err := schedman.GetK8sCandidateHosts(node.Name)
	if err != nil {
		return false, err
	}
	if len(hosts) == 0 {
		return false, fmt.Errorf("Not found candidate host %s", node.Name)
	}
	k8sCli, err := schedman.GetK8sClient()
	if err != nil {
		log.Warningf("Get k8s client error: %v, some predicates will not execute", err)
	}
	return k8spredicates.PredicatesManager.DoFilter(k8sCli, pod, node, hosts[0])
}

func k8sPrioritizeHandler(c *gin.Context) {
	var extenderArgs schedulerapi.ExtenderArgs
	var hostPriorityList *schedulerapi.HostPriorityList

	if err := json.NewDecoder(c.Request.Body).Decode(&extenderArgs); err != nil {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("Decode json body: %v", err))
		return
	}
	hostPriorityList, err := k8sPrioritizeFunc(extenderArgs)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Prioritize error: %v", err))
		return
	}
	c.JSON(http.StatusOK, hostPriorityList)
}

// TODO: support priority extender
func k8sPrioritizeFunc(args schedulerapi.ExtenderArgs) (*schedulerapi.HostPriorityList, error) {
	nodes := args.Nodes.Items
	var priorityList schedulerapi.HostPriorityList = make([]schedulerapi.HostPriority, 0)
	for _, node := range nodes {
		priorityList = append(priorityList, schedulerapi.HostPriority{
			Host:  node.Name,
			Score: 0,
		})
	}
	return &priorityList, nil
}
