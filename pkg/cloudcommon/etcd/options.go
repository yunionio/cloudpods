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

package etcd

import "crypto/tls"

type SEtcdOptions struct {
	EtcdEndpoint              []string `help:"etcd endpoints in format of addr:port"`
	EtcdTimeoutSeconds        int      `default:"5" help:"etcd dial timeout in seconds"`
	EtcdRequestTimeoutSeconds int      `default:"2" help:"etcd request timeout in seconds"`
	EtcdLeaseExpireSeconds    int      `default:"5" help:"etcd expire timeout in seconds"`

	EtcdNamspace string `help:"etcd namespace"`

	EtcdUsername string `help:"etcd username"`
	EtcdPassword string `help:"etcd password"`

	EtcdEnabldSsl     bool        `help:"enable SSL/TLS"`
	EtcdSslCertfile   string      `help:"ssl certification file"`
	EtcdSslKeyfile    string      `help:"ssl certification private key file"`
	EtcdSslCaCertfile string      `help:"ssl ca certification file"`
	TLSConfig         *tls.Config `help:"tls config"`
}
