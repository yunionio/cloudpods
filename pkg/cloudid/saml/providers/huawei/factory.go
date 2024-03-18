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

package huawei

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"

	"yunion.io/x/onecloud/pkg/cloudid/models"
)

type SHuaweiSAMLDriver struct {
	EntityId         string
	MetadataFileName string
	MetadataUrl      string
}

func (d *SHuaweiSAMLDriver) GetEntityID() string {
	return d.EntityId
}

func (d *SHuaweiSAMLDriver) GetMetadataFilename() string {
	return d.MetadataFileName
}

func (d *SHuaweiSAMLDriver) GetMetadataUrl() string {
	return d.MetadataUrl
}

func init() {
	models.Register(&SHuaweiSAMLDriver{
		EntityId:         cloudprovider.SAML_ENTITY_ID_HUAWEI_CLOUD,
		MetadataFileName: "huawei.xml",
		MetadataUrl:      "https://auth.huaweicloud.com/authui/saml/metadata.xml",
	})
}
