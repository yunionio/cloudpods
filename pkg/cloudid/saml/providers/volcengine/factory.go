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

package volcengine

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"

	"yunion.io/x/onecloud/pkg/cloudid/models"
)

type SVolcEngineSAMLDriver struct{}

func (d *SVolcEngineSAMLDriver) GetEntityID() string {
	return cloudprovider.SAML_ENTITY_ID_VOLC_ENGINE
}

func (d *SVolcEngineSAMLDriver) GetMetadataFilename() string {
	return "volcengine.xml"
}

func (d *SVolcEngineSAMLDriver) GetMetadataUrl() string {
	return "https://signin.volcengine.com/saml_role/SpMetadata.xml"
}

func init() {
	models.Register(&SVolcEngineSAMLDriver{})
}
