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

package utils

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

func ClearTaskPendingUsage(ctx context.Context, task taskman.ITask) error {
	index := 0
	pendingUsage := models.SQuota{}
	err := task.GetPendingUsage(&pendingUsage, index)
	if err != nil {
		log.Errorf("GetPendingUsage fail %s", err)
		// ignore error
		// return errors.Wrap(err, "task.GetPendingUsage")
		return nil
	}

	log.Debugf("ClearTaskPendingUsage %s", jsonutils.Marshal(pendingUsage))

	if pendingUsage.IsEmpty() {
		return nil
	}

	err = quotas.CancelPendingUsage(ctx, task.GetUserCred(), &pendingUsage, &pendingUsage, false)
	if err != nil {
		return errors.Wrap(err, "models.QuotaManager.CancelPendingUsage")
	}
	if pendingUsage.IsEmpty() {
		err = task.ClearPendingUsage(index)
		if err != nil {
			return errors.Wrap(err, "task.ClearPendingUsage")
		}
	} else {
		log.Warningf("pendingUsage %s is not empty after cancel????", jsonutils.Marshal(pendingUsage).String())
	}
	return nil
}

func ClearTaskPendingRegionUsage(ctx context.Context, task taskman.ITask) error {
	index := 1
	pendingUsage := models.SRegionQuota{}
	err := task.GetPendingUsage(&pendingUsage, index)
	if err != nil {
		log.Errorf("GetPendingUsage fail %s", err)
		// ignore error
		// return errors.Wrap(err, "task.GetPendingUsage")
		return nil
	}

	if pendingUsage.IsEmpty() {
		return nil
	}

	err = quotas.CancelPendingUsage(ctx, task.GetUserCred(), &pendingUsage, &pendingUsage, false)
	if err != nil {
		return errors.Wrap(err, "models.QuotaManager.CancelPendingUsage")
	}
	if pendingUsage.IsEmpty() {
		err = task.ClearPendingUsage(index)
		if err != nil {
			return errors.Wrap(err, "task.ClearPendingUsage")
		}
	} else {
		log.Warningf("pendingUsage %s is not empty after cancel????", jsonutils.Marshal(pendingUsage).String())
	}
	return nil
}
