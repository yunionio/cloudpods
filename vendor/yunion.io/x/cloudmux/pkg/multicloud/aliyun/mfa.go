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

package aliyun

import "time"

type MFADevice struct {
	SerialNumber     string
	QRCodePNG        string
	Base32StringSeed string
	ActivateDate     time.Time
	User             struct {
		DisplayName       string
		UserId            string
		UserPrincipalName string
	}
}

func (self *SAliyunClient) ListVirtualMFADevices() ([]MFADevice, error) {
	ret := []MFADevice{}
	params := map[string]string{}
	for {
		resp, err := self.imsRequest("ListVirtualMFADevices", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			VirtualMFADevices struct {
				VirtualMFADevice []MFADevice
			}
			Marker string
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.VirtualMFADevices.VirtualMFADevice...)
		if len(part.Marker) == 0 {
			break
		}
		params["Marker"] = part.Marker
	}
	return ret, nil
}

func (self *SAliyunClient) UnbindMFADevice(userName string) error {
	params := map[string]string{
		"UserPrincipalName": userName,
	}
	_, err := self.imsRequest("UnbindMFADevice", params)
	return err
}

func (self *SAliyunClient) CreateVirtualMFADevice(name string) (*MFADevice, error) {
	params := map[string]string{
		"VirtualMFADeviceName": name,
	}
	resp, err := self.imsRequest("CreateVirtualMFADevice", params)
	if err != nil {
		return nil, err
	}
	ret := &MFADevice{}
	err = resp.Unmarshal(ret, "VirtualMFADevice")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SAliyunClient) DisableVirtualMFA(user string) error {
	params := map[string]string{
		"UserPrincipalName": user,
	}
	_, err := self.imsRequest("DisableVirtualMFA", params)
	return err
}

func (self *SAliyunClient) BindMFADevice(user string, seriaNum string, code1, code2 string) error {
	params := map[string]string{
		"UserPrincipalName":   user,
		"SerialNumber":        seriaNum,
		"AuthenticationCode1": code1,
		"AuthenticationCode2": code2,
	}
	_, err := self.imsRequest("BindMFADevice", params)
	return err
}
