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

package db

import (
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
)

type IOwnerResourceBaseModel interface {
	GetChangeOwnerCandidateDomainIds() []string
	GetChangeOwnerRequiredDomainIds() []string
}

type IManagedResourceBase interface {
	IsManaged() bool
}

func IOwnerResourceBaseModelGetChangeOwnerCandidateDomains(model IOwnerResourceBaseModel) (apis.ChangeOwnerCandidateDomainsOutput, error) {
	output := apis.ChangeOwnerCandidateDomainsOutput{}
	candidateIds := model.GetChangeOwnerCandidateDomainIds()
	if len(candidateIds) == 0 {
		return output, nil
	}
	domainMap := make(map[string]STenant)
	err := FetchQueryObjectsByIds(TenantCacheManager.GetDomainQuery(), "id", candidateIds, &domainMap)
	if err != nil {
		return output, errors.Wrap(err, "FetchQueryObjectsByIds")
	}
	output.Candidates = make([]apis.SharedDomain, len(candidateIds))
	for i := range candidateIds {
		output.Candidates[i].Id = candidateIds[i]
		if domain, ok := domainMap[candidateIds[i]]; ok {
			output.Candidates[i].Name = domain.Name
		}
	}
	return output, nil
}
