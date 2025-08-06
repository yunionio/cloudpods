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

package hostdrivers

import (
	"context"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SVirtualizationHostDriver struct {
	SBaseHostDriver
}

func (self *SVirtualizationHostDriver) RequestRemoteUpdateDisk(ctx context.Context, userCred mcclient.TokenCredential, storage *models.SStorage, disk *models.SDisk, replaceTags bool) error {
	iDisk, err := disk.GetIDisk(ctx)
	if err != nil {
		return errors.Wrap(err, "GetIDisk")
	}

	err = func() error {
		oldTags, err := iDisk.GetTags()
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
				return nil
			}
			return errors.Wrap(err, "iVM.GetTags()")
		}
		tags, err := disk.GetAllUserMetadata()
		if err != nil {
			return errors.Wrapf(err, "GetAllUserMetadata")
		}
		tagsUpdateInfo := cloudprovider.TagsUpdateInfo{OldTags: oldTags, NewTags: tags}

		err = cloudprovider.SetTags(ctx, iDisk, storage.ManagerId, tags, replaceTags)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
				return nil
			}
			logclient.AddSimpleActionLog(disk, logclient.ACT_UPDATE_TAGS, err, userCred, false)
			return errors.Wrap(err, "iVM.SetTags")
		}
		logclient.AddSimpleActionLog(disk, logclient.ACT_UPDATE_TAGS, tagsUpdateInfo, userCred, true)
		return nil
	}()
	if err != nil {
		return err
	}

	return nil
}
