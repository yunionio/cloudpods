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

package options

type ElasticCacheCreateOptions struct {
	NAME          string
	Manager       string
	Cloudregion   string
	Zone          string
	VpcId         string
	Network       string `help:"network id"`
	SecurityGroup string `help:"elastic cache security group. required by huawei."`
	Engine        string `choices:"redis"`
	EngineVersion string `choices:"2.8|3.0|4.0|5.0"`
	PrivateIP     string `help:"private ip address in specificated network"`
	Password      string `help:"set auth password"`
	InstanceType  string
	CapacityMB    string `help:"elastic cache capacity. required by huawei."`
	BillingType   string `choices:"postpaid|prepaid" default:"postpaid"`
	Month         int    `help:"billing duration (unit:month)"`
}

type ElasticCacheAccountCreateOptions struct {
	Elasticcache     string `help:"elastic cache instance id"`
	Name             string
	Password         string
	AccountPrivilege string `help:"account privilege" choices:"read|write|repl" default:"read"`
}

type ElasticCacheBackupCreateOptions struct {
	Elasticcache string `help:"elastic cache instance id"`
	Name         string
}

type ElasticCacheAclCreateOptions struct {
	Elasticcache string `help:"elastic cache instance id"`
	Name         string
	IpList       string `help:"elastic cache acl ip list, split by ','"`
}

type ElasticCacheAclUpdateOptions struct {
	Id     string `help:"elastic cache acl id"`
	IpList string `help:"elastic cache acl ip list, split by ','"`
}

type ElasticCacheParameterUpdateOptions struct {
	Id    string `help:"elastic cache parameter id"`
	Value string `help:"elastic cache parameter value"`
}

type ElasticCacheIdOptions struct {
	ID string
}
