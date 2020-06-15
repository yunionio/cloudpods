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

package loader

import (
	"net/http"
	"net/url"
	"testing"

	"yunion.io/x/s3cli"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

func TestProxyFunc(t *testing.T) {
	runT := func(t *testing.T, cpcfg *cloudprovider.ProviderConfig) {
		vendor := cpcfg.Vendor
		proxied := false
		proxyFunc := func(req *http.Request) (*url.URL, error) {
			proxied = true
			return url.Parse("http://TestProxyFunc" + "." + vendor + "/")
		}
		cpcfg.ProxyFunc = proxyFunc
		_, err := cloudprovider.GetProvider(*cpcfg)
		if !proxied {
			if err != nil {
				t.Logf("vendor %s: err: %v", vendor, err)
			}
			t.Errorf("vendor %s: proxyFunc not working", vendor)
		}
		t.Run("default client no proxy", func(t *testing.T) {
			proxied = false
			client := httputils.GetDefaultClient()
			client.Get("http://default-client-no-proxy.TestProxyFunc." + vendor + "/")
			if proxied {
				t.Errorf("%s: default client proxy changed", vendor)
			}
		})
	}

	t.Parallel()
	t.Run("cloud", func(t *testing.T) {
		cpcfgs := map[string]*cloudprovider.ProviderConfig{
			compute.CLOUD_PROVIDER_VMWARE:    &cloudprovider.ProviderConfig{},
			compute.CLOUD_PROVIDER_QCLOUD:    &cloudprovider.ProviderConfig{},
			compute.CLOUD_PROVIDER_AZURE:     &cloudprovider.ProviderConfig{},
			compute.CLOUD_PROVIDER_HUAWEI:    &cloudprovider.ProviderConfig{},
			compute.CLOUD_PROVIDER_OPENSTACK: &cloudprovider.ProviderConfig{},
			compute.CLOUD_PROVIDER_ZSTACK:    &cloudprovider.ProviderConfig{},
			compute.CLOUD_PROVIDER_UCLOUD:    &cloudprovider.ProviderConfig{},
			compute.CLOUD_PROVIDER_ALIYUN:    &cloudprovider.ProviderConfig{},
			compute.CLOUD_PROVIDER_AWS:       &cloudprovider.ProviderConfig{},
			compute.CLOUD_PROVIDER_CTYUN:     &cloudprovider.ProviderConfig{},
			compute.CLOUD_PROVIDER_GOOGLE:    &cloudprovider.ProviderConfig{},
		}
		for vendor := range cpcfgs {
			cpcfgs[vendor].Vendor = vendor
			cpcfgs[vendor].Id = vendor + "-Id"
			cpcfgs[vendor].Name = vendor + "-Name"
			cpcfgs[vendor].URL = "http://" + vendor + ".AccessUrl/TestProxyFunc"
			cpcfgs[vendor].Account = vendor + "-Account"
			cpcfgs[vendor].Secret = vendor + "-Secret"
		}

		if true {
			cpcfgs[compute.CLOUD_PROVIDER_OPENSTACK].Account = "projectId/username/domainName"
		}
		if true {
			cpcfgs[compute.CLOUD_PROVIDER_AZURE].URL = "AzureChinaCloud"
			cpcfgs[compute.CLOUD_PROVIDER_AZURE].Account = "tenantId/subscriptionId"
			cpcfgs[compute.CLOUD_PROVIDER_AZURE].Secret = "appId/appKey"
		}
		if true {
			cpcfgs[compute.CLOUD_PROVIDER_UCLOUD].Account = "accessKey::projectId"
		}
		if true {
			const pkey = `-----BEGIN RSA PRIVATE KEY-----` + "\n" +
				`MIICXgIBAAKBgQDIUZ819CKNzPf3dL1aTC9tx6Y+Z/wQ74hWgyxM4DM5kxMZDoWd` + "\n" +
				`2zzj7B8vUU64tYaeCaPFahxcMNs705wNACkFnmqop08zUSWTfbPg/uNdhGvuF0vT` + "\n" +
				`3o5oNbUfVLrusvVJwP6mAnfJsECZJsPMqVIyM5z6uMBpAkjsaqEP7cOISwIDAQAB` + "\n" +
				`AoGBAIzBXZ/ptcXxumM0skCau9DlspizNFkqFqDDdwNlnljcUVUP8S9kd7qnlZoI` + "\n" +
				`BcqgMsElgOAcmWTmJ4Y4QLSZ4jgDthDUqt+dv384G4tUHv5LDU4SMnKiPFzqzsOn` + "\n" +
				`PR72ZcoZZZW9PvNMmJDWSIwuEXgWovXhK5pO3wuuHUDjPHlxAkEA8DdcBRAV9yuq` + "\n" +
				`AbVFSWxBIpNkXkIoOCQiBOP8LBTvua5Dkuxe5qZkDXuTupYmLRmIoDJuZVKo0jWw` + "\n" +
				`6xBg1Io4BQJBANV7J3u5HV7IxlG35g8iCsU/AnLYQYWIWqF9+DKM8fqkDHGx6KGl` + "\n" +
				`THG0gKGjhkWe/qzTsQbe4gHWZp6z1apVQA8CQQC9aDkzeMKJGTG8lQzj3urg82zx` + "\n" +
				`caK62arFRmrA62U2eHSptZ0CqvI7O1R5jAjgCMEU547qb6nTzigIwOpIzA9tAkB6` + "\n" +
				`EyaM1Vo6LU1axXJFDQ5jpJhd29P76/EHj+Ux4u8siEqlaTrB1KhWDQXRaJULktPH` + "\n" +
				`OHZum11Z96RO9D8mXAF5AkEAq1xtCrJp8rWTGH5PGShyCOk2ZNbV0AL+may4FAgc` + "\n" +
				`hbZ+WzxylVxMKJmWqEBAYF3/7oouyteF8Vq3TVOv442NSg==` + "\n" +
				`-----END RSA PRIVATE KEY-----`
			cpcfgs[compute.CLOUD_PROVIDER_GOOGLE].Account = "projectId/email"
			cpcfgs[compute.CLOUD_PROVIDER_GOOGLE].Secret = "keyid/" + pkey
		}
		for _, cpcfg := range cpcfgs {
			cpcfg := cpcfg
			t.Run(cpcfg.Vendor, func(t *testing.T) {
				//t.Parallel()
				runT(t, cpcfg)
			})
		}
	})
	t.Run("objectstore", func(t *testing.T) {
		s3cli.MaxRetry = 1
		cpcfgs := map[string]*cloudprovider.ProviderConfig{
			compute.CLOUD_PROVIDER_CEPH:      &cloudprovider.ProviderConfig{},
			compute.CLOUD_PROVIDER_XSKY:      &cloudprovider.ProviderConfig{},
			compute.CLOUD_PROVIDER_GENERICS3: &cloudprovider.ProviderConfig{},
		}
		for vendor := range cpcfgs {
			cpcfgs[vendor].Vendor = vendor
			cpcfgs[vendor].Id = vendor + "-Id"
			cpcfgs[vendor].Name = vendor + "-Name"
			cpcfgs[vendor].URL = "http://" + vendor + ".AccessUrl/TestProxyFunc"
			cpcfgs[vendor].Account = vendor + "-Account"
			cpcfgs[vendor].Secret = vendor + "-Secret"
		}
		for _, cpcfg := range cpcfgs {
			cpcfg := cpcfg
			t.Run(cpcfg.Vendor, func(t *testing.T) {
				//t.Parallel()
				runT(t, cpcfg)
			})
		}
	})
}
