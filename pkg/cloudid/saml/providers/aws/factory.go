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

package aws

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"

	"yunion.io/x/onecloud/pkg/cloudid/models"
)

type SAWSSAMLDriver struct{}

func (d *SAWSSAMLDriver) GetEntityID() string {
	return cloudprovider.SAML_ENTITY_ID_AWS
}

func (d *SAWSSAMLDriver) GetMetadataFilename() string {
	return "aws.xml"
}

func (d *SAWSSAMLDriver) GetMetadataUrl() string {
	return "https://signin.aws.amazon.com/static/saml-metadata.xml"
}

func init() {
	models.Register(&SAWSSAMLDriver{})
}
