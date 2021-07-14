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
	"github.com/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SInternetGateway struct {
	multicloud.SResourceBase
	multicloud.AwsTags
	region *SRegion

	Attachments       []InternetGatewayAttachment `json:"Attachments"`
	InternetGatewayID string                      `json:"InternetGatewayId"`
	OwnerID           string                      `json:"OwnerId"`
}

type InternetGatewayAttachment struct {
	State string `json:"State"`
	VpcID string `json:"VpcId"`
}

func (i *SInternetGateway) GetId() string {
	return i.InternetGatewayID
}

func (i *SInternetGateway) GetName() string {
	return i.InternetGatewayID
}

func (i *SInternetGateway) GetGlobalId() string {
	return i.GetId()
}

func (i *SInternetGateway) GetStatus() string {
	return ""
}

func (i *SInternetGateway) Refresh() error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Refresh")
}

func (i *SInternetGateway) IsEmulated() bool {
	return false
}
