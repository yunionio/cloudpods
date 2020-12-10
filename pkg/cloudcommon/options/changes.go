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

package options

import (
	"sort"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
)

func OnBaseOptionsChange(oOpts, nOpts interface{}) bool {
	oldOpts := oOpts.(*BaseOptions)
	newOpts := nOpts.(*BaseOptions)

	changed := false
	if oldOpts.RequestWorkerCount != newOpts.RequestWorkerCount {
		log.Debugf("RequestWorkerCount changed from %d to %d", oldOpts.RequestWorkerCount, newOpts.RequestWorkerCount)
		changed = true
	}
	if oldOpts.TimeZone != newOpts.TimeZone {
		changed = true
	}
	if oldOpts.EnableRbac != newOpts.EnableRbac {
		changed = true
	}
	if oldOpts.NonDefaultDomainProjects != newOpts.NonDefaultDomainProjects {
		consts.SetNonDefaultDomainProjects(newOpts.NonDefaultDomainProjects)
		changed = true
	}
	if oldOpts.DomainizedNamespace != newOpts.DomainizedNamespace {
		consts.SetDomainizedNamespace(newOpts.DomainizedNamespace)
		changed = true
	}
	if privatePrrefixesChanged(oldOpts.CustomizedPrivatePrefixes, newOpts.CustomizedPrivatePrefixes) {
		netutils.SetPrivatePrefixes(newOpts.CustomizedPrivatePrefixes)
		log.Debugf("Customized private prefixes: %s", netutils.GetPrivateIPRanges())
	}
	if oldOpts.EnableQuotaCheck != newOpts.EnableQuotaCheck {
		consts.SetEnableQuotaCheck(newOpts.EnableQuotaCheck)
	}
	return changed
}

func privatePrrefixesChanged(oldprefs, newprefs []string) bool {
	if len(oldprefs) != len(newprefs) {
		return true
	}
	if len(oldprefs) == 0 {
		return false
	}
	sort.Strings(oldprefs)
	sort.Strings(newprefs)
	for i := range newprefs {
		if oldprefs[i] != newprefs[i] {
			return true
		}
	}
	return false
}

func OnCommonOptionsChange(oOpts, nOpts interface{}) bool {
	oldOpts := oOpts.(*CommonOptions)
	newOpts := nOpts.(*CommonOptions)

	changed := false
	if OnBaseOptionsChange(&oldOpts.BaseOptions, &newOpts.BaseOptions) {
		changed = true
	}

	return changed
}

func OnDBOptionsChange(oOpts, nOpts interface{}) bool {
	oldOpts := oOpts.(*DBOptions)
	newOpts := nOpts.(*DBOptions)

	changed := false

	if oldOpts.HistoricalUniqueName != newOpts.HistoricalUniqueName {
		if newOpts.HistoricalUniqueName {
			consts.EnableHistoricalUniqueName()
		} else {
			consts.DisableHistoricalUniqueName()
		}
	}

	return changed
}
