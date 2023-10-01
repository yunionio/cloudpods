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

package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

func (guest *SGuest) PerformLoginInfo(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.ServerLoginInfoInput,
) (*api.ServerLoginInfoOutput, error) {
	metadata, err := guest.GetAllMetadata(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "GetAllMetadata")
	}
	output := &api.ServerLoginInfoOutput{}
	output.Username = metadata["login_account"]
	output.Updated = metadata["login_key_timestamp"]
	output.LoginKey = metadata["login_key"]

	if len(output.LoginKey) > 0 {
		var passwd string
		keypair := guest.getKeypair()
		if keypair != nil {
			if len(input.PrivateKey) > 0 {
				passwd, err = seclib2.DecryptBase64(input.PrivateKey, output.LoginKey)
				if err != nil {
					return nil, errors.Wrap(err, "DecryptBase64")
				}
			} else {
				return nil, errors.Wrap(httperrors.ErrInputParameter, "empty private key")
			}
		} else {
			passwd, err = utils.DescryptAESBase64(guest.Id, output.LoginKey)
			if err != nil {
				return nil, errors.Wrap(err, "DescryptAESBase64")
			}
		}
		output.Password = passwd
	}

	return output, nil
}

func (host *SHost) PerformLoginInfo(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.HostLoginInfoInput,
) (*api.HostLoginInfoOutput, error) {
	metadata, err := host.GetAllMetadata(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "GetAllMetadata")
	}

	login_key := metadata["password"]
	// decrypt twice
	passwd, err := utils.DescryptAESBase64(host.Id, login_key)
	if err != nil {
		return nil, errors.Wrap(err, "DescryptAESBase64")
	}
	passwd, err = utils.DescryptAESBase64(host.Id, passwd)
	if err != nil {
		return nil, errors.Wrap(err, "DescryptAESBase64 twice")
	}

	ret := &api.HostLoginInfoOutput{}
	ret.Password = passwd
	ret.Username = metadata["username"]
	ret.Ip = metadata["ip"]

	return ret, nil
}
