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

package k8s

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates/guest"
	"yunion.io/x/onecloud/pkg/scheduler/cache/candidate"
)

type HostStatusPredicate struct{}

func (p *HostStatusPredicate) Clone() IPredicate {
	return &HostStatusPredicate{}
}

func (p *HostStatusPredicate) Name() string {
	return "host_status"
}

func (p *HostStatusPredicate) PreExecute(cli *kubernetes.Clientset, pod *v1.Pod, node *v1.Node, host *candidate.HostDesc) bool {
	return true
}

func (p *HostStatusPredicate) Execute(cli *kubernetes.Clientset, pod *v1.Pod, node *v1.Node, host *candidate.HostDesc) (bool, error) {
	if host.Status != guest.ExpectedStatus {
		return false, fmt.Errorf("Host status is %s", host.Status)
	}

	if !host.GetEnabled() {
		return false, fmt.Errorf("Host is disabled")
	}
	return true, nil
}
