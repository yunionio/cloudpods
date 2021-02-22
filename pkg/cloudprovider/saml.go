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

import "yunion.io/x/onecloud/pkg/util/samlutils"

const (
	SAML_ENTITY_ID_ALIYUN_ROLE  = "urn:alibaba:cloudcomputing"
	SAML_ENTITY_ID_AWS_CN       = "urn:amazon:webservices:cn-north-1"
	SAML_ENTITY_ID_AWS          = "urn:amazon:webservices"
	SAML_ENTITY_ID_QCLOUD       = "cloud.tencent.com"
	SAML_ENTITY_ID_HUAWEI_CLOUD = "https://auth.huaweicloud.com/"
	SAML_ENTITY_ID_GOOGLE       = "google.com"
	SAML_ENTITY_ID_AZURE        = "urn:federation:MicrosoftOnline"
)

type SAMLProviderCreateOptions struct {
	Name     string
	Metadata samlutils.EntityDescriptor
}
