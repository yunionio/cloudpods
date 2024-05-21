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

package yunionmeta

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/version"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

const (
	EMPTY_MD5 = "d751713988987e9331980363e24189ce"
)

var meta *SSkuResourcesMeta

type SSkuResourcesMeta struct {
	// RDS套餐
	DBInstanceBase string `json:"dbinstance_base"`
	// 虚拟机套餐
	ServerBase string `json:"server_base"`
	// Redis套餐
	ElasticCacheBase string `json:"elastic_cache_base"`
	// 公有云镜像
	ImageBase string `json:"image_base"`

	NatBase string `json:"nat_base"`
	NasBase string `json:"nas_base"`
	WafBase string `json:"waf_base"`

	CloudpolicyBase string `json:"cloudpolicy_base"`

	RateBase string `json:"rate_base"`
	// 汇率转换
	CurrencyExchangeBase string `json:"currency_exchange_base"`

	// 3天过期, 重新刷新
	expire time.Time
}

func GetZoneIdBySuffix(zoneMaps map[string]string, suffix string) string {
	for externalId, id := range zoneMaps {
		if strings.HasSuffix(externalId, suffix) {
			return id
		}
	}
	return ""
}

func FetchYunionmeta(ctx context.Context) (*SSkuResourcesMeta, error) {
	if !gotypes.IsNil(meta) && meta.expire.After(time.Now()) {
		return meta, nil
	}
	s := auth.GetAdminSession(ctx, "")
	transport := httputils.GetTransport(true)
	transport.Proxy = options.Options.HttpTransportProxyFunc()
	client := &http.Client{Transport: transport}
	resp, err := compute.OfflineCloudmeta.GetSkuSourcesMeta(s, client)
	if err != nil {
		return nil, errors.Wrap(err, "fetchSkuSourceUrls.GetSkuSourcesMeta")
	}

	meta = &SSkuResourcesMeta{}
	err = resp.Unmarshal(meta)
	if err != nil {
		return nil, errors.Wrap(err, "fetchSkuSourceUrls.Unmarshal")
	}

	meta.expire = time.Now().AddDate(0, 0, 3)
	return meta, nil
}

func (self *SSkuResourcesMeta) request(url string) (jsonutils.JSONObject, error) {
	client := httputils.GetAdaptiveTimeoutClient()

	header := http.Header{}
	header.Set("User-Agent", "vendor/yunion-OneCloud@"+version.Get().GitVersion)
	_, resp, err := httputils.JSONRequest(client, context.TODO(), httputils.GET, url, header, nil, false)
	return resp, err
}

func (self *SSkuResourcesMeta) head(url string) (http.Header, jsonutils.JSONObject, error) {
	client := httputils.GetAdaptiveTimeoutClient()

	header := http.Header{}
	header.Set("User-Agent", "vendor/yunion-OneCloud@"+version.Get().GitVersion)
	_header, resp, err := httputils.JSONRequest(client, context.TODO(), httputils.HEAD, url, header, nil, false)
	return _header, resp, err
}

func (self *SSkuResourcesMeta) GetCurrencyRate(src, dest string) (float64, error) {
	url := fmt.Sprintf("%s/%s-%s", self.CurrencyExchangeBase, src, dest)
	header, _, err := self.head(url)
	if err != nil {
		return 0.0, errors.Wrapf(err, "head %s", url)
	}
	rate := header.Get("x-oss-meta-rate")
	if len(rate) == 0 {
		return 0.0, errors.Wrapf(cloudprovider.ErrNotFound, "x-oss-meta-rate %s -> %s", src, dest)
	}
	return strconv.ParseFloat(rate, 64)
}

func (self *SSkuResourcesMeta) _get(url string) ([]jsonutils.JSONObject, error) {
	objs, err := self.request(url)
	if err != nil {
		return nil, errors.Wrapf(err, "request %s", url)
	}
	var ret []jsonutils.JSONObject
	return ret, objs.Unmarshal(&ret)
}

func (self *SSkuResourcesMeta) Get(url string, retVal interface{}) error {
	obj, err := self.request(url)
	if err != nil {
		return errors.Wrapf(err, "request %s", url)
	}
	return obj.Unmarshal(retVal)
}

func (self *SSkuResourcesMeta) Index(resType string) (map[string]string, error) {
	var url string
	switch resType {
	case "dbinstance_sku":
		url = fmt.Sprintf("%s/index.json", self.DBInstanceBase)
	case "serversku":
		url = fmt.Sprintf("%s/index.json", self.ServerBase)
	case "elasticcachesku":
		url = fmt.Sprintf("%s/index.json", self.ElasticCacheBase)
	case "cloudimage":
		url = fmt.Sprintf("%s/index.json", self.ImageBase)
	case "nat_sku":
		url = fmt.Sprintf("%s/index.json", self.NatBase)
	case "nas_sku":
		url = fmt.Sprintf("%s/index.json", self.NasBase)
	case "waf_rule":
		url = fmt.Sprintf("%s/index.json", self.WafBase)
	case "cloudpolicy":
		url = fmt.Sprintf("%s/index.json", self.CloudpolicyBase)
	case "cloudrate":
		url = fmt.Sprintf("%s/index.json", self.RateBase)
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, resType)
	}
	ret := map[string]string{}
	resp, err := self.request(url)
	if err != nil {
		return nil, errors.Wrapf(err, "url")
	}
	return ret, resp.Unmarshal(ret)
}

func (self *SSkuResourcesMeta) List(resType string, regionId string, retVal interface{}) error {
	if strings.HasPrefix(regionId, api.CLOUD_PROVIDER_HUAWEI) && strings.Contains(regionId, "_") {
		idx := strings.Index(regionId, "_")
		regionId = regionId[:idx]
	}
	var url string
	switch resType {
	case "dbinstance_sku":
		url = fmt.Sprintf("%s/%s.status.json", self.DBInstanceBase, regionId)
	case "serversku":
		url = fmt.Sprintf("%s/%s.status.json", self.ServerBase, regionId)
	case "elasticcachesku":
		url = fmt.Sprintf("%s/%s.status.json", self.ElasticCacheBase, regionId)
	case "cloudimage":
		url = fmt.Sprintf("%s/%s.status.json", self.ImageBase, regionId)
	case "nat_sku":
		url = fmt.Sprintf("%s/%s.status.json", self.NatBase, regionId)
	case "nas_sku":
		url = fmt.Sprintf("%s/%s.status.json", self.NasBase, regionId)
	case "waf_rule":
		url = fmt.Sprintf("%s/%s.json", self.WafBase, regionId)
	case "cloudpolicy":
		url = fmt.Sprintf("%s/%s.json", self.CloudpolicyBase, regionId)
	default:
		return errors.Wrapf(cloudprovider.ErrNotFound, resType)
	}
	resp, err := self._get(url)
	if err != nil {
		return errors.Wrapf(err, resType)
	}
	return jsonutils.Update(retVal, resp)
}
