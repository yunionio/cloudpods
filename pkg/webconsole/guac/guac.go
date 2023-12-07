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

package guac

import (
	"fmt"
	"net"
)

func NewGuacamoleTunnel(host string, port int, user, password string, id string, w, h, dpi int, uId string) (*GuacamoleTunnel, error) {
	opts := NewGuacOptions()
	opts.ConnectionId = id
	opts.Protocol = "rdp"
	if w > 0 && h > 0 {
		opts.OptimalScreenHeight = h
		opts.OptimalScreenWidth = w
	}
	if dpi > 0 {
		opts.OptimalResolution = dpi
	}
	opts.AudioMimetypes = []string{"audio/L16", "rate=44100", "channels=2"}
	opts.Parameters = map[string]string{
		"scheme":            opts.Protocol,
		"hostname":          host,
		"port":              fmt.Sprintf("%d", port),
		"ignore-cert":       "true",
		"security":          "",
		"username":          user,
		"password":          password,
		"enable-drive":      "true",
		"drive-name":        "Cloudpods",
		"drive-path":        "/opt/cloudpods/" + uId,
		"create-drive-path": "true",
	}
	conn, err := net.Dial("tcp", "127.0.0.1:4822")
	if err != nil {
		return nil, err
	}
	ret := &GuacamoleTunnel{
		conn: conn,
		opts: opts,
	}
	err = ret.Handshake()
	if err != nil {
		return nil, err
	}
	return ret, nil
}
