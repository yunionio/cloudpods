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

package predicates

import (
	"context"
	"fmt"
	"sync"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	schedulerapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

const hostPathPredicateParallelism = 16

type hostPathChecker func(context.Context, *models.SHost, []schedulerapi.HostPathRequirement) (*hostapi.HostPathCheckOutput, error)

type HostPathPredicate struct {
	BasePredicate

	checker     hostPathChecker
	failReasons map[string]string
}

func NewHostPathPredicate() *HostPathPredicate {
	return &HostPathPredicate{}
}

func (p *HostPathPredicate) Name() string {
	return "host_path"
}

func (p *HostPathPredicate) Clone() core.FitPredicate {
	return &HostPathPredicate{
		checker: p.checker,
	}
}

func (p *HostPathPredicate) PreExecute(ctx context.Context, u *core.Unit, cs []core.Candidater) (bool, error) {
	reqs := u.SchedData().HostPathRequirements
	if len(reqs) == 0 {
		return false, nil
	}

	checker := p.checker
	if checker == nil {
		userCred := u.SchedInfo.UserCred
		checker = func(ctx context.Context, host *models.SHost, reqs []schedulerapi.HostPathRequirement) (*hostapi.HostPathCheckOutput, error) {
			return checkHostPathsOnHost(ctx, userCred, host, reqs)
		}
	}

	p.failReasons = make(map[string]string)
	sem := make(chan struct{}, hostPathPredicateParallelism)
	wg := sync.WaitGroup{}
	lock := sync.Mutex{}
	for _, c := range cs {
		candidate := c
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() {
				<-sem
			}()

			reason := checkHostPathCandidate(ctx, checker, candidate, reqs)
			if reason == "" {
				return
			}
			lock.Lock()
			p.failReasons[candidate.IndexKey()] = reason
			lock.Unlock()
		}()
	}
	wg.Wait()

	return true, nil
}

func (p *HostPathPredicate) Execute(ctx context.Context, u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := NewPredicateHelper(p, u, c)
	if reason := p.failReasons[c.IndexKey()]; reason != "" {
		h.Exclude(reason)
	}
	return h.GetResult()
}

func checkHostPathCandidate(ctx context.Context, checker hostPathChecker, candidate core.Candidater, reqs []schedulerapi.HostPathRequirement) string {
	host := candidate.Getter().Host()
	if host == nil {
		return "host path check unavailable: candidate host is nil"
	}
	output, err := checker(ctx, host, reqs)
	return hostPathCheckFailureReason(reqs, output, err)
}

func checkHostPathsOnHost(ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, reqs []schedulerapi.HostPathRequirement) (*hostapi.HostPathCheckOutput, error) {
	input := hostapi.HostPathCheckInput{
		Paths: make([]hostapi.HostPathCheckItem, 0, len(reqs)),
	}
	for _, req := range reqs {
		input.Paths = append(input.Paths, hostapi.HostPathCheckItem{
			Path: req.Path,
			Type: req.Type,
		})
	}

	resp, err := host.Request(ctx, userCred, "POST", fmt.Sprintf("/hosts/%s/check-host-paths", host.Id), mcclient.GetTokenHeaders(userCred), jsonutils.Marshal(input))
	if err != nil {
		return nil, err
	}
	output := new(hostapi.HostPathCheckOutput)
	if err := resp.Unmarshal(output); err != nil {
		return nil, err
	}
	return output, nil
}

func hostPathCheckFailureReason(reqs []schedulerapi.HostPathRequirement, output *hostapi.HostPathCheckOutput, err error) string {
	if err != nil {
		return fmt.Sprintf("host path check unavailable: %v", err)
	}
	if output == nil {
		return "host path check unavailable: empty response"
	}

	results := make(map[string]hostapi.HostPathCheckResult, len(output.Results))
	for _, result := range output.Results {
		results[hostPathRequirementKey(result.Path, result.Type)] = result
	}
	for _, req := range reqs {
		result, ok := results[hostPathRequirementKey(req.Path, req.Type)]
		if !ok {
			return fmt.Sprintf("host path %s check result missing", req.Path)
		}
		if !result.Exists {
			if result.Error != "" {
				return result.Error
			}
			return fmt.Sprintf("host path %s does not exist", req.Path)
		}
		if !result.TypeMatched {
			if result.Error != "" {
				return result.Error
			}
			return fmt.Sprintf("host path %s is not %s", req.Path, req.Type)
		}
	}
	return ""
}

func hostPathRequirementKey(path string, typ apis.ContainerVolumeMountHostPathType) string {
	return fmt.Sprintf("%s\x00%s", path, string(typ))
}
