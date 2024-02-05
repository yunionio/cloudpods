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

package types

import (
	"context"

	"yunion.io/x/jsonutils"

	compute_api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type OpenForwardRequest struct {
	NetworkId string `json:"network_id"`
	Proto     string `json:"proto"`
	Addr      string `json:"addr"`
	Port      int    `json:"port"`
}

type OpenForwardResponse struct {
	Proto     string `json:"proto"`
	ProxyAddr string `json:"proxy_addr"`
	ProxyPort int    `json:"proxy_port"`
	Addr      string `json:"addr"`
	Port      int    `json:"port"`
}

func NewOpenForwardRequestFromJSON(ctx context.Context, data jsonutils.JSONObject) (*OpenForwardRequest, error) {
	dict, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, httperrors.ErrInputParameter
	}
	var (
		protoV = validators.NewStringChoicesValidator("proto", compute_api.GuestForwardProtoChoices)
		portV  = validators.NewPortValidator("port")
		addrV  = validators.NewIPv4AddrValidator("addr")
	)
	for _, v := range []validators.IValidator{
		protoV.Default(compute_api.GuestForwardProtoTCP),
		portV,
		addrV.Optional(true),
	} {
		if err := v.Validate(ctx, dict); err != nil {
			return nil, err
		}
	}
	req := &OpenForwardRequest{
		Proto: protoV.Value,
		Port:  int(portV.Value),
	}
	if addrV.IP != nil && addrV.IP.String() != "" {
		req.Addr = addrV.IP.String()
	}
	return req, nil
}

func (resp *OpenForwardResponse) JSON() jsonutils.JSONObject {
	return jsonutils.Marshal(resp)
}

type CloseForwardRequest struct {
	NetworkId string `json:"network_id"`
	Proto     string `json:"proto"`
	ProxyAddr string `json:"addr"`
	ProxyPort int    `json:"port"`
}

type CloseForwardResponse struct {
	NetworkId string `json:"network_id"`
	Proto     string `json:"proto"`
	ProxyAddr string `json:"addr"`
	ProxyPort int    `json:"port"`
}

func NewCloseForwardRequestFromJSON(ctx context.Context, data jsonutils.JSONObject) (*CloseForwardRequest, error) {
	dict, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, httperrors.ErrInputParameter
	}
	var (
		protoV     = validators.NewStringChoicesValidator("proto", compute_api.GuestForwardProtoChoices)
		proxyAddrV = validators.NewIPv4AddrValidator("proxy_addr")
		proxyPortV = validators.NewPortValidator("proxy_port")
	)
	for _, v := range []validators.IValidator{
		protoV,
		proxyAddrV,
		proxyPortV,
	} {
		if err := v.Validate(ctx, dict); err != nil {
			return nil, err
		}
	}
	req := &CloseForwardRequest{
		Proto:     protoV.Value,
		ProxyAddr: proxyAddrV.IP.String(),
		ProxyPort: int(proxyPortV.Value),
	}
	return req, nil
}

func (resp *CloseForwardResponse) JSON() jsonutils.JSONObject {
	return jsonutils.Marshal(resp)
}

type ListForwardRequest struct {
	NetworkId string `json:"network_id"`
	Proto     string `json:"proto"`
	Addr      string `json:"addr"`
	Port      int    `json:"port"`
}

type ListForwardResponse struct {
	Forwards []OpenForwardResponse `json:"forwards"`
}

func NewListForwardRequestFromJSON(ctx context.Context, data jsonutils.JSONObject) (*ListForwardRequest, error) {
	dict, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, httperrors.ErrInputParameter
	}
	var (
		protoV = validators.NewStringChoicesValidator("proto", compute_api.GuestForwardProtoChoices)
		portV  = validators.NewPortValidator("port")
		addrV  = validators.NewIPv4AddrValidator("addr")
	)
	for _, v := range []validators.IValidator{
		protoV.Optional(true),
		portV.Optional(true),
		addrV.Optional(true),
	} {
		if err := v.Validate(ctx, dict); err != nil {
			return nil, err
		}
	}
	req := &ListForwardRequest{
		Proto: protoV.Value,
		Port:  int(portV.Value),
	}
	if addrV.IP != nil && addrV.IP.String() != "" {
		req.Addr = addrV.IP.String()
	}
	return req, nil
}

func (resp *ListForwardResponse) JSON() jsonutils.JSONObject {
	return jsonutils.Marshal(resp)
}
