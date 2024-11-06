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

package conditions

import (
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/monitor/models"
)

func OkEvalMatchSetIsRecovery(
	alert *models.SCommonAlert,
	resId string,
	em *monitor.EvalMatch,
) error {
	isAlerting, err := alert.IsResourceMetricAlerting(resId, em.Metric)
	if err != nil {
		return errors.Wrap(err, "check if previous alert is alerting")
	}
	if isAlerting {
		em.IsRecovery = true
	}
	return nil
}
