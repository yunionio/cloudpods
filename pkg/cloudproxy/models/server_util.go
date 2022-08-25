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

	compute_apis "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	compute_modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type serverInfo struct {
	Server *compute_apis.ServerDetails

	// PrivateKey is the one corresponds to userCred when getting this
	// serverInfo instance.  It can be empty
	PrivateKey string
}

func (si *serverInfo) GetNic() *compute_apis.GuestnetworkShortDesc {
	nic := si.getVPCNic()
	if nic != nil {
		return nic
	}

	for _, nic := range si.Server.Nics {
		if nic.IpAddr != "" && nic.VpcId == compute_apis.DEFAULT_VPC_ID {
			return &nic
		}
	}
	return nil
}

func (si *serverInfo) getVPCNic() *compute_apis.GuestnetworkShortDesc {
	for _, nic := range si.Server.Nics {
		if nic.IpAddr != "" && nic.VpcId != compute_apis.DEFAULT_VPC_ID {
			return &nic
		}
	}
	return nil
}

func getServerInfo(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	serverId string,
) (*serverInfo, error) {
	sess := auth.GetSession(ctx, userCred, "")
	serverJson, err := compute_modules.Servers.Get(sess, serverId, nil)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	server := &compute_apis.ServerDetails{}
	if err := serverJson.Unmarshal(server); err != nil {
		return nil, httperrors.NewServerError("unmarshal server %s: %v", serverId, err)
	}

	privateKey, _ := compute_modules.Sshkeypairs.FetchPrivateKey(ctx, userCred)

	serverInfo := &serverInfo{
		Server:     server,
		PrivateKey: privateKey,
	}
	return serverInfo, nil
}
