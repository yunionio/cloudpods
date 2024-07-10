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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type CachedImageListOptions struct {
	options.BaseListOptions
	ImageType string `help:"image type" choices:"system|customized|shared|market"`

	Region string `help:"show images cached at cloud region"`
	Zone   string `help:"show images cached at zone"`

	HostSchedtagId string `help:"filter cached image with host schedtag"`
	Valid          *bool  `help:"valid cachedimage"`
}

func (opts *CachedImageListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type CachedImageCacheImageOptions struct {
	ID string
}

func (opts *CachedImageCacheImageOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"image_id": opts.ID}), nil
}
