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

package multicloud

import (
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SBaseClouduser struct {
}

func (self *SBaseClouduser) CreateAccessKey(name string) (*cloudprovider.SAccessKey, error) {
	return nil, errors.Wrap(cloudprovider.ErrNotImplemented, "base CreateAccessKey")
}

func (self *SBaseClouduser) GetAccessKeys() ([]cloudprovider.SAccessKey, error) {
	return nil, errors.Wrap(cloudprovider.ErrNotImplemented, "base GetAccessKeys")
}

func (self *SBaseClouduser) DeleteAccessKey(accesskey string) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "base DeleteAccessKey")
}

func (self *SBaseClouduser) SetDisable() error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetDisable")
}

func (self *SBaseClouduser) SetEnable(opts *cloudprovider.SClouduserEnableOptions) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetDisable")
}
