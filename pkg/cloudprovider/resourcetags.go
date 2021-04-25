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

package cloudprovider

import (
	"context"
	"reflect"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
)

const (
	SET_TAGS = "set-tags"
)

type TagsUpdateInfo struct {
	OldTags map[string]string
	NewTags map[string]string
}

func (t TagsUpdateInfo) IsChanged() bool {
	return !reflect.DeepEqual(t.OldTags, t.NewTags)
}

func SetTags(ctx context.Context, res ICloudResource, managerId string, tags map[string]string, replace bool) error {
	// 避免同时设置多个资源标签出现以下错误
	// Code=ResourceInUse.TagDuplicate, Message=tagKey-tagValue have exists., RequestId=e87714c0-e50b-4241-b79d-32897437174d
	lockman.LockRawObject(ctx, SET_TAGS, managerId)
	defer lockman.ReleaseRawObject(ctx, SET_TAGS, managerId)

	return res.SetTags(tags, replace)
}
