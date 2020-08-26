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
	"sort"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

func allowAction(manager IResource, userCred mcclient.TokenCredential, action string, testfunc func(scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, manager IResource) bool) bool {
	if action == "user" {
		return true
	}
	if action == "domain" && (testfunc(rbacutils.ScopeDomain, userCred, manager) || testfunc(rbacutils.ScopeSystem, userCred, manager)) {
		return true
	}
	if action == "admin" && testfunc(rbacutils.ScopeSystem, userCred, manager) {
		return true
	}
	return false
}

func allowRequired(manager IResource, userCred mcclient.TokenCredential, action string, testfunc func(scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, manager IResource) bool) bool {
	if action == "required" {
		return true
	}
	if action == "domain_required" && (testfunc(rbacutils.ScopeDomain, userCred, manager) || testfunc(rbacutils.ScopeSystem, userCred, manager)) {
		return true
	}
	if action == "admin_required" && testfunc(rbacutils.ScopeSystem, userCred, manager) {
		return true
	}
	return false
}

func allowOptional(manager IResource, userCred mcclient.TokenCredential, action string, testfunc func(scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, manager IResource) bool) bool {
	if action == "optional" {
		return true
	}
	if action == "domain_optional" && (testfunc(rbacutils.ScopeDomain, userCred, manager) || testfunc(rbacutils.ScopeSystem, userCred, manager)) {
		return true
	}
	if action == "admin_optional" && testfunc(rbacutils.ScopeSystem, userCred, manager) {
		return true
	}
	return false
}

/**
 * Column metadata fields to control list, search, details, update, create
 *  list: user | domain | admin
 *  search: user | domain | admin
 *  get: user | domain | admin
 *  create: required | optional | domain_required | domain_optional | admin_required | admin_optional
 *  update: user | domain | admin
 *
 */
func listFields(manager IModelManager, userCred mcclient.TokenCredential) ([]string, []string) {
	includes := make([]string, 0)
	excludes := make([]string, 0)
	for _, col := range manager.TableSpec().Columns() {
		tags := col.Tags()
		list, _ := tags["list"]
		if allowAction(manager, userCred, list, IsAllowList) {
			includes = append(includes, col.Name())
		} else {
			excludes = append(excludes, col.Name())
		}
	}
	return includes, excludes
}

func searchFields(manager IModelManager, userCred mcclient.TokenCredential) stringutils2.SSortedStrings {
	ret := make([]string, 0)
	for _, col := range manager.TableSpec().Columns() {
		tags := col.Tags()
		list := tags["list"]
		search := tags["search"]
		if allowAction(manager, userCred, list, IsAllowList) || allowAction(manager, userCred, search, IsAllowList) {
			ret = append(ret, col.Name())
		}
	}
	sort.Strings(ret)
	return stringutils2.SSortedStrings(ret)
}

func GetDetailFields(manager IModelManager, userCred mcclient.TokenCredential) ([]string, []string) {
	includes := make([]string, 0)
	excludes := make([]string, 0)
	for _, col := range manager.TableSpec().Columns() {
		tags := col.Tags()
		list := tags["list"]
		get := tags["get"]
		if allowAction(manager, userCred, list, IsAllowGet) || allowAction(manager, userCred, get, IsAllowGet) {
			includes = append(includes, col.Name())
		} else {
			excludes = append(excludes, col.Name())
		}
	}
	return includes, excludes
}

func createRequireFields(manager IModelManager, userCred mcclient.TokenCredential) []string {
	ret := make([]string, 0)
	for _, col := range manager.TableSpec().Columns() {
		tags := col.Tags()
		create, _ := tags["create"]
		if allowRequired(manager, userCred, create, IsAllowCreate) {
			ret = append(ret, col.Name())
		}
	}
	return ret
}

func createFields(manager IModelManager, userCred mcclient.TokenCredential) []string {
	ret := make([]string, 0)
	for _, col := range manager.TableSpec().Columns() {
		tags := col.Tags()
		create, _ := tags["create"]
		update := tags["update"]
		if allowRequired(manager, userCred, create, IsAllowCreate) || allowOptional(manager, userCred, create, IsAllowCreate) || allowAction(manager, userCred, update, IsAllowCreate) {
			ret = append(ret, col.Name())
		}
	}
	return ret
}

func updateFields(manager IModelManager, userCred mcclient.TokenCredential) []string {
	ret := make([]string, 0)
	for _, col := range manager.TableSpec().Columns() {
		tags := col.Tags()
		update := tags["update"]
		if allowAction(manager, userCred, update, IsAllowUpdate) {
			ret = append(ret, col.Name())
		}
	}
	return ret
}
