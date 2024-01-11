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

package bingocloud

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	xj "github.com/basgys/goxml2json"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	CLOUD_PROVIDER_BINGO_CLOUD = api.CLOUD_PROVIDER_BINGO_CLOUD

	MAX_RESULT = 20
)

type BingoCloudConfig struct {
	cpcfg     cloudprovider.ProviderConfig
	endpoint  string
	accessKey string
	secretKey string

	debug bool
}

func NewBingoCloudClientConfig(endpoint, accessKey, secretKey string) *BingoCloudConfig {
	cfg := &BingoCloudConfig{
		endpoint:  endpoint,
		accessKey: accessKey,
		secretKey: secretKey,
	}
	return cfg
}

func (cfg *BingoCloudConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *BingoCloudConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *BingoCloudConfig) Debug(debug bool) *BingoCloudConfig {
	cfg.debug = debug
	return cfg
}

type SBingoCloudClient struct {
	*BingoCloudConfig

	regions []SRegion
}

func NewBingoCloudClient(cfg *BingoCloudConfig) (*SBingoCloudClient, error) {
	client := &SBingoCloudClient{BingoCloudConfig: cfg}
	var err error
	client.regions, err = client.GetRegions()
	if err != nil {
		return nil, err
	}
	for i := range client.regions {
		client.regions[i].client = client
	}
	return client, nil
}

func (self *SBingoCloudClient) GetAccountId() string {
	return self.endpoint
}

func (self *SBingoCloudClient) GetRegion(id string) (*SRegion, error) {
	for i := range self.regions {
		if self.regions[i].RegionId == id {
			return &self.regions[i], nil
		}
	}
	if len(id) == 0 {
		return &self.regions[0], nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SBingoCloudClient) getDefaultClient(timeout time.Duration) *http.Client {
	client := httputils.GetDefaultClient()
	if timeout > 0 {
		client = httputils.GetTimeoutClient(timeout)
	}
	if self.cpcfg.ProxyFunc != nil {
		httputils.SetClientProxyFunc(client, self.cpcfg.ProxyFunc)
	}
	return client
}

func (self *SBingoCloudClient) sign(query string) string {
	uri, _ := url.Parse(self.endpoint)
	items := strings.Split(query, "&")
	sort.Slice(items, func(i, j int) bool {
		x0, y0 := strings.Split(items[i], "=")[0], strings.Split(items[j], "=")[0]
		return x0 < y0
	})
	path := "/"
	if len(uri.Path) > 0 {
		path = uri.Path
	}
	stringToSign := fmt.Sprintf("POST\n%s\n%s\n", uri.Host, path) + strings.Join(items, "&")
	hmac := hmac.New(sha256.New, []byte(self.secretKey))
	hmac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(hmac.Sum(nil))
}

func setItemToArray(obj jsonutils.JSONObject) jsonutils.JSONObject {
	objDict, ok := obj.(*jsonutils.JSONDict)
	if ok {
		for k, v := range objDict.Value() {
			if v.String() == `""` {
				objDict.Remove(k)
				continue
			}
			vDict, ok := v.(*jsonutils.JSONDict)
			if ok {
				if vDict.Contains("item") {
					item, _ := vDict.Get("item")
					_, ok := item.(*jsonutils.JSONArray)
					if !ok {
						if k != "instancesSet" {
							item = setItemToArray(item)
							objDict.Set(k, jsonutils.NewArray(item))
						} else {
							objDict.Set(k, setItemToArray(item))
						}
					} else {
						items, _ := item.GetArray()
						for i := range items {
							items[i] = setItemToArray(items[i])
						}
						objDict.Set(k, jsonutils.NewArray(items...))
					}
					for _, nk := range []string{"nextToken", "NextToken"} {
						nextToken, _ := vDict.GetString(nk)
						if len(nextToken) > 0 {
							objDict.Set(nk, jsonutils.NewString(nextToken))
						}
					}
				} else {
					objDict.Set(k, setItemToArray(v))
				}
			} else if _, ok = v.(*jsonutils.JSONArray); ok {
				if ok {
					arr, _ := v.GetArray()
					for i := range arr {
						arr[i] = setItemToArray(arr[i])
					}
					objDict.Set(k, jsonutils.NewArray(arr...))
				}
			}
		}
	}
	_, ok = obj.(*jsonutils.JSONArray)
	if ok {
		arr, _ := obj.GetArray()
		for i := range arr {
			arr[i] = setItemToArray(arr[i])
		}
		return jsonutils.NewArray(arr...)
	}
	return objDict
}

type sBingoError struct {
	Response struct {
		Errors struct {
			Error struct {
				Code    string
				ErrorNo string
				Message string
			}
		}
	}
}

func (e sBingoError) Error() string {
	return jsonutils.Marshal(e.Response.Errors.Error).String()
}

func (self *SBingoCloudClient) invoke(action string, params map[string]string) (jsonutils.JSONObject, error) {
	if self.cpcfg.ReadOnly {
		for _, prefix := range []string{"Get", "List", "Describe"} {
			if strings.HasPrefix(action, prefix) {
				return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, action)
			}
		}
	}
	var encode = func(k, v string) string {
		d := url.Values{}
		d.Set(k, v)
		return d.Encode()
	}
	query := encode("Action", action)
	for k, v := range params {
		query += "&" + encode(k, v)
	}
	// 2022-02-11T03:57:37.000Z
	sh, _ := time.LoadLocation("Asia/Shanghai")
	timeStamp := time.Now().In(sh).Format("2006-01-02T15:04:05.000Z")
	query += "&" + encode("Timestamp", timeStamp)
	query += "&" + encode("AWSAccessKeyId", self.accessKey)
	query += "&" + encode("Version", "2009-08-15")
	query += "&" + encode("SignatureVersion", "2")
	query += "&" + encode("SignatureMethod", "HmacSHA256")
	query += "&" + encode("Signature", self.sign(query))
	client := self.getDefaultClient(time.Minute * 5)
	resp, err := httputils.Request(client, context.Background(), httputils.POST, self.endpoint, nil, strings.NewReader(query), self.debug)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	result, err := xj.Convert(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	obj, err := jsonutils.Parse([]byte(result.String()))
	if err != nil {
		return nil, errors.Wrapf(err, "jsonutils.Parse")
	}

	obj = setItemToArray(obj)

	if self.debug {
		log.Debugf("response: %s", obj.PrettyString())
	}

	be := &sBingoError{}
	_ = obj.Unmarshal(be)
	if len(be.Response.Errors.Error.Code) > 0 {
		return nil, be
	}

	respKey := action + "Response"
	if obj.Contains(respKey) {
		obj, err = obj.Get(respKey)
		if err != nil {
			return nil, err
		}
	}

	return obj, nil
}

func (self *SBingoCloudClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	var tags []struct {
		ResourceId string `json:"resourceId"`
		Value      string `json:"value"`
	}

	filter := map[string]string{}
	filter["resource-type"] = "user"
	filter["key"] = "ProjectID"
	result, err := self.describeTags(filter)
	if err != nil {
		return nil, err
	}
	_ = result.Unmarshal(&tags, "tagSet")

	var subAccounts []cloudprovider.SSubAccount
	for i := range tags {
		subAccount := cloudprovider.SSubAccount{
			Id:               tags[i].ResourceId,
			Account:          self.accessKey,
			Name:             tags[i].ResourceId,
			DefaultProjectId: tags[i].Value,
			HealthStatus:     api.CLOUD_PROVIDER_HEALTH_NORMAL,
		}
		subAccounts = append(subAccounts, subAccount)
	}
	return subAccounts, nil
}

func (self *SBingoCloudClient) GetEnrollmentAccounts() ([]cloudprovider.SEnrollmentAccount, error) {
	params := map[string]string{"Marker": "", "MaxItems": "1000", "AccountName": "paas_app"}
	var result struct {
		IsTruncated string
		Marker      string `json:"marker,omitempty"`
		Users       struct {
			Member *SAccount `json:"member,omitempty"`
		}
	}
	var eas []cloudprovider.SEnrollmentAccount
	for {
		resp, err := self.invoke("ListAccounts", params)
		if err != nil {
			return nil, err
		}
		err = resp.Unmarshal(&result, "ListAccountsResult")
		if err != nil {
			return nil, err
		}
		ea := cloudprovider.SEnrollmentAccount{
			Id:   result.Users.Member.UserId,
			Name: result.Users.Member.UserName,
		}
		eas = append(eas, ea)
		//for _, user := range result.Users.Member {
		//	ea := cloudprovider.SEnrollmentAccount{
		//		Id:   user.UserId,
		//		Name: user.UserName,
		//	}
		//	eas = append(eas, ea)
		//}
		if params["Marker"] == result.Marker {
			break
		}
		params["Marker"] = result.Marker
	}
	return eas, nil
}

func (self *SBingoCloudClient) GetIRegions() []cloudprovider.ICloudRegion {
	var ret []cloudprovider.ICloudRegion
	for i := range self.regions {
		self.regions[i].client = self
		ret = append(ret, &self.regions[i])
	}
	return ret
}

func (self *SBingoCloudClient) describeTags(filter map[string]string) (jsonutils.JSONObject, error) {
	params := map[string]string{"MaxResults": "10000"}
	i := 1
	for k, v := range filter {
		params[fmt.Sprintf("Filter.%v.Name", i)] = k
		params[fmt.Sprintf("Filter.%v.Value.1", i)] = v
		i++
	}
	return self.invoke("DescribeTags", params)
}

func (self *SBingoCloudClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	iregions := self.GetIRegions()
	for i := range iregions {
		if iregions[i].GetGlobalId() == id {
			return iregions[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SBingoCloudClient) GetCapabilities() []string {
	return []string{
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		cloudprovider.CLOUD_CAPABILITY_NETWORK,
		cloudprovider.CLOUD_CAPABILITY_SECURITY_GROUP,
		cloudprovider.CLOUD_CAPABILITY_EIP,
		cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
	}
}
