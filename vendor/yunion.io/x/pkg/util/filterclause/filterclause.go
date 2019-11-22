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

package filterclause

import (
	"fmt"
	"regexp"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"
)

type SFilterClause struct {
	field    string
	funcName string
	params   []string
}

// "guestnetworks.guest_id(id).ip_addr.equals(10.168.222.232)"
type SJointFilterClause struct {
	SFilterClause
	JointModel string
	RelatedKey string
	OriginKey  string
}

func (jfc *SJointFilterClause) GetJointFilter(q *sqlchemy.SQuery) sqlchemy.ICondition {
	return jfc.QueryCondition(q)
}

func (jfc *SJointFilterClause) GetJointModelName() string {
	return jfc.JointModel[:len(jfc.JointModel)-1]
}

func condFunc(field sqlchemy.IQueryField, params []string, cond func(field sqlchemy.IQueryField, val string) sqlchemy.ICondition) sqlchemy.ICondition {
	if len(params) == 1 {
		return cond(field, params[0])
	} else if len(params) > 1 {
		conds := make([]sqlchemy.ICondition, len(params))
		for i := range params {
			conds[i] = cond(field, params[i])
		}
		return sqlchemy.OR(conds...)
	} else {
		return nil
	}
}

func (fc *SFilterClause) QueryCondition(q *sqlchemy.SQuery) sqlchemy.ICondition {
	field := q.Field(fc.field)
	if field == nil {
		log.Errorf("Cannot find field %s", fc.field)
		return nil
	}
	switch fc.funcName {
	case "in":
		return sqlchemy.In(field, fc.params)
	case "notin":
		return sqlchemy.NotIn(field, fc.params)
	case "between":
		if len(fc.params) == 2 {
			return sqlchemy.Between(field, fc.params[0], fc.params[1])
		}
	case "ge":
		if len(fc.params) == 1 {
			return sqlchemy.GE(field, fc.params[0])
		}
	case "gt":
		if len(fc.params) == 1 {
			return sqlchemy.GT(field, fc.params[0])
		}
	case "le":
		if len(fc.params) == 1 {
			return sqlchemy.LE(field, fc.params[0])
		}
	case "lt":
		if len(fc.params) == 1 {
			return sqlchemy.LT(field, fc.params[0])
		}
	case "like":
		return condFunc(field, fc.params, sqlchemy.Like)
	case "contains":
		return condFunc(field, fc.params, sqlchemy.Contains)
	case "startswith":
		return condFunc(field, fc.params, sqlchemy.Startswith)
	case "endswith":
		return condFunc(field, fc.params, sqlchemy.Endswith)
	case "equals":
		return condFunc(field, fc.params, func(f sqlchemy.IQueryField, p string) sqlchemy.ICondition {
			return sqlchemy.Equals(f, p)
		})
	case "notequals":
		if len(fc.params) == 1 {
			return sqlchemy.NOT(sqlchemy.Equals(field, fc.params[0]))
		}
	case "isnull":
		return sqlchemy.IsNull(field)
	case "isnotnull":
		return sqlchemy.IsNotNull(field)
	case "isempty":
		return sqlchemy.IsEmpty(field)
	case "isnotempty":
		return sqlchemy.IsNotEmpty(field)
	case "isnullorempty":
		return sqlchemy.IsNullOrEmpty(field)
	}
	return nil
}

func (fc *SFilterClause) GetField() string {
	return fc.field
}

func (fc *SFilterClause) String() string {
	return fmt.Sprintf("%s.%s(%s)", fc.field, fc.funcName, strings.Join(fc.params, ","))
}

var (
	filterClausePattern      *regexp.Regexp
	jointFilterClausePattern *regexp.Regexp
)

func init() {
	filterClausePattern = regexp.MustCompile(`^(\w+)\.(\w+)\((.*)\)`)
	jointFilterClausePattern = regexp.MustCompile(`^(\w+)\.(\w+)\((\w+)\).(\w+)\.(\w+)\((.*)\)`)
}

func ParseFilterClause(filter string) *SFilterClause {
	matches := filterClausePattern.FindStringSubmatch(filter)
	if matches == nil {
		return nil
	}
	params := utils.FindWords([]byte(matches[3]), 0)
	fc := SFilterClause{field: matches[1], funcName: matches[2], params: params}
	return &fc
}

func ParseJointFilterClause(jointFilter string) *SJointFilterClause {
	matches := jointFilterClausePattern.FindStringSubmatch(jointFilter)
	if matches == nil {
		return nil
	}
	params := utils.FindWords([]byte(matches[6]), 0)
	jfc := SJointFilterClause{
		SFilterClause: SFilterClause{
			field:    matches[4],
			funcName: matches[5],
			params:   params,
		},
		JointModel: matches[1],
		RelatedKey: matches[2],
		OriginKey:  matches[3],
	}
	return &jfc
}
