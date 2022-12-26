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

package rbacscope

import (
	"strings"
)

type TRbacScope string

const (
	ScopeSystem  = TRbacScope("system")
	ScopeDomain  = TRbacScope("domain")
	ScopeProject = TRbacScope("project")
	// ScopeObject  = "object"
	ScopeUser = TRbacScope("user")
	ScopeNone = TRbacScope("none")
)

var (
	scopeScore = map[TRbacScope]int{
		ScopeNone:    0,
		ScopeUser:    1,
		ScopeProject: 2,
		ScopeDomain:  3,
		ScopeSystem:  4,
	}
)

func (s1 TRbacScope) HigherEqual(s2 TRbacScope) bool {
	return scopeScore[s1] >= scopeScore[s2]
}

func (s1 TRbacScope) HigherThan(s2 TRbacScope) bool {
	return scopeScore[s1] > scopeScore[s2]
}

func String2Scope(str string) TRbacScope {
	return String2ScopeDefault(str, ScopeProject)
}

func String2ScopeDefault(str string, defScope TRbacScope) TRbacScope {
	switch strings.ToLower(str) {
	case string(ScopeSystem):
		return ScopeSystem
	case string(ScopeDomain):
		return ScopeDomain
	case string(ScopeProject):
		return ScopeProject
	case string(ScopeUser):
		return ScopeUser
	case "true":
		return ScopeSystem
	default:
		return defScope
	}
}
