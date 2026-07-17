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
	_ struct{} `mcp-desc:"【创建流程中的中间步骤】公有云/非KVM 选镜像。必须带 provider（如 [\"Aliyun\"]），region 必须传 climc_cloud_region_list 返回的 id（UUID），禁止传 cn-shanghai 这类云厂商 region code。不要用 ISO。查完后继续 network/sku，最后 climc_server_create"`

	options.BaseListOptions
	ImageType string `help:"image type；公有云系统盘常用 system" choices:"system|customized|shared|market" mcp:"true"`

	// Region 序列化为 cloudregion_id；优先传 cloudregion UUID
	Region string `help:"cloudregion id（推荐）或 name；不要传 cn-shanghai 这类外部 region code" json:"cloudregion_id" mcp:"true"`
	Zone   string `help:"show images cached at zone" mcp:"true"`

	HostSchedtagId string `help:"filter cached image with host schedtag" mcp:"true"`
	Valid          *bool  `help:"valid cachedimage" mcp:"true"`
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
