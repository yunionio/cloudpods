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

package ovnutils

type SOvnOptions struct {
	OvnSouthDatabase          string `help:"address for accessing ovn south database" default:"$HOST_OVN_SOUTH_DATABASE|unix:/var/run/openvswitch/ovnsb_db.sock"`
	OvnIntegrationBridge      string `help:"name of integration bridge for logical ports" default:"$HOST_OVN_INTEGRATION_BRIDGE|brvpc"`
	OvnEncapIpDetectionMethod string `help:"detection method for ovn_encap_ip" default:"$HOST_OVN_ENCAP_IP_DETECTION_METHOD"`
	OvnEncapIp                string `help:"encap ip for ovn datapath.  Default to src address of default route" default:"$HOST_OVN_ENCAP_IP"`
	OvnUnderlayMtu            int    `help:"mtu of ovn underlay network" default:"1500"`
}
