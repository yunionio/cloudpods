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

package saml

import (
	"encoding/xml"
	"fmt"

	"yunion.io/x/pkg/util/samlutils"

	"yunion.io/x/onecloud/pkg/keystone/options"
)

func GetMetadata(idpName string, pretty bool) string {
	input := samlutils.SSAMLSpMetadataInput{
		EntityId:    options.Options.ApiServer,
		CertString:  saml.GetCertString(),
		ServiceName: fmt.Sprintf("%s (Keystone Service Provider)", idpName),

		AssertionConsumerUrl: "%SAMLACSURL%",
		RequestedAttributes: []samlutils.RequestedAttribute{
			{
				IsRequired:   "false",
				Name:         "userId",
				FriendlyName: "userId",
			},
			{
				IsRequired:   "false",
				Name:         "projectId",
				FriendlyName: "projectId",
			},
			{
				IsRequired:   "false",
				Name:         "roleId",
				FriendlyName: "roleId",
			},
		},
	}
	ed := samlutils.NewSpMetadata(input)
	var xmlBytes []byte
	if pretty {
		xmlBytes, _ = xml.MarshalIndent(ed, "", "  ")
	} else {
		xmlBytes, _ = xml.Marshal(ed)
	}
	return string(xmlBytes)
}
