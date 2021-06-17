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

package compute

import (
	"reflect"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
)

const (
	PROJECT_MAPPING_STATUS_AVAILABLE = "available"

	MAPPING_CONDITION_AND = "and"
	MAPPING_CONDITION_OR  = "or"
)

type STag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ProjectMappingRuleInfo struct {
	// 标签列表, 不可为空
	Tags []STag `json:"tags"`
	// 条件表达式
	// enmu: and, or
	// default: and
	Condition string `json:"condition"`
	// 是否自动根据标签值创建项目, 仅标签列表中有且仅有一个没有value的key时支持
	AutoCreateProject bool `json:"auto_create_project"`
	// 符合条件时，资源放置的项目id, 此参数和auto_create_project互斥
	ProjectId string `json:"project_id"`
	// 只读信息
	// swagger:ignore
	Project string `json:"project"`
	// swagger:ignore
	DomainId string `json:"domain_id"`
	// 只读信息
	// swagger:ignore
	Domain string `json:"domain"`
}

type MappingRules []ProjectMappingRuleInfo

func (rule *ProjectMappingRuleInfo) Validate() error {
	if len(rule.Tags) == 0 {
		return httperrors.NewInputParameterError("missing tags")
	}
	if len(rule.Condition) == 0 {
		rule.Condition = MAPPING_CONDITION_AND
	}
	if !utils.IsInStringArray(rule.Condition, []string{MAPPING_CONDITION_AND, MAPPING_CONDITION_OR}) {
		return httperrors.NewInputParameterError("invalid condition")
	}
	emptyValueTags := 0
	for _, tag := range rule.Tags {
		if len(tag.Key) == 0 {
			return httperrors.NewInputParameterError("missing tag key for")
		}
		if len(tag.Value) == 0 {
			emptyValueTags++
		}
	}
	if emptyValueTags != 1 && rule.AutoCreateProject {
		return httperrors.NewInputParameterError("not support auto_create_project")
	}
	if !rule.AutoCreateProject && len(rule.ProjectId) == 0 {
		return httperrors.NewInputParameterError("missing project_id")
	}
	return nil
}

// return domainId, projectId, newProj, isMatch
func (self *ProjectMappingRuleInfo) IsMatchTags(_extTags map[string]string) (string, string, string, bool) {
	newProj := ""
	extTags := map[string]string{}
	for k, v := range _extTags {
		extTags[strings.ToUpper(k)] = v
	}
	switch self.Condition {
	case MAPPING_CONDITION_AND:
		for _, tag := range self.Tags {
			extTag, ok := extTags[strings.ToUpper(tag.Key)]
			if !ok || (len(tag.Value) > 0 && tag.Value != extTag) {
				return "", "", "", false
			}
			if self.AutoCreateProject && len(tag.Value) == 0 && len(extTag) > 0 {
				newProj = extTag
			}
		}
		return self.DomainId, self.ProjectId, newProj, true
	case MAPPING_CONDITION_OR:
		for _, tag := range self.Tags {
			extTag, ok := extTags[strings.ToUpper(tag.Key)]
			if ok && (len(tag.Value) == 0 || tag.Value == extTag) {
				if self.AutoCreateProject && len(tag.Value) == 0 && len(extTag) > 0 {
					return "", "", extTag, true
				} else {
					return self.DomainId, self.ProjectId, "", true
				}
			}
		}
	}
	return "", "", "", false
}

func (rules MappingRules) Validate() error {
	for i := range rules {
		err := rules[i].Validate()
		if err != nil {
			return err
		}
	}
	return nil
}

type ProjectMappingRuleInfoDetails struct {
	ProjectMappingRuleInfo
	Project  string
	TenantId string
	Tenant   string
	Domain   string
}

type SProjectMappingAccount struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type ProjectMappingDetails struct {
	apis.EnabledStatusInfrasResourceBaseDetails

	Rules []ProjectMappingRuleInfoDetails

	// 所绑定的云账号列表
	Accounts []SProjectMappingAccount `json:"accounts"`

	// 所绑定的云订阅列表
	Managers []SProjectMappingAccount `json:"managers"`
}

type ProjectMappingCreateInput struct {
	apis.EnabledStatusInfrasResourceBaseCreateInput

	Rules MappingRules
}

type ProjectMappingListInput struct {
	apis.EnabledStatusInfrasResourceBaseListInput
}

type ProjectMappingResourceInfo struct {
	ProjectMapping string
}

type ProjectMappingUpdateInput struct {
	apis.EnabledStatusInfrasResourceBaseUpdateInput

	Rules MappingRules
}

type ProjectMappingFilterListInput struct {
	ProjectMappingId      string `json:"project_mapping_id"`
	OrderByProjectMapping string
}

func (self MappingRules) String() string {
	return jsonutils.Marshal(self).String()
}

func (self MappingRules) IsZero() bool {
	if len(self) == 0 {
		return true
	}
	return false
}

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&MappingRules{}), func() gotypes.ISerializable {
		return &MappingRules{}
	})
}
