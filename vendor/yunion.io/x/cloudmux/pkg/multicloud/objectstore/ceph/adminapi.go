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

package ceph

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/s3auth"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SCephAdminApi struct {
	adminPath string
	accessKey string
	secret    string
	endpoint  string
	client    *http.Client
	debug     bool
}

func newCephAdminApi(ak, sk, ep string, debug bool, adminPath string) *SCephAdminApi {
	if adminPath == "" {
		adminPath = "admin"
	}
	return &SCephAdminApi{
		adminPath: adminPath,
		accessKey: ak,
		secret:    sk,
		endpoint:  ep,
		// ceph use no timeout client so as to download/upload large files
		client: httputils.GetAdaptiveTimeoutClient(),
		debug:  debug,
	}
}

func getJsonBodyReader(body jsonutils.JSONObject) io.Reader {
	var reqBody io.Reader
	if body != nil {
		reqBody = strings.NewReader(body.String())
	}
	return reqBody
}

func (api *SCephAdminApi) httpClient() *http.Client {
	return api.client
}

func (api *SCephAdminApi) jsonRequest(ctx context.Context, method httputils.THttpMethod, path string, hdr http.Header, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	urlStr := strings.TrimRight(api.endpoint, "/") + "/" + strings.TrimLeft(path, "/")
	req, err := http.NewRequest(string(method), urlStr, getJsonBodyReader(body))
	if err != nil {
		return nil, nil, errors.Wrap(err, "http.NewRequest")
	}
	if hdr != nil {
		for k, vs := range hdr {
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
	}

	newReq := s3auth.SignV4(*req, api.accessKey, api.secret, "cn-beijing", getJsonBodyReader(body))

	resp, err := api.client.Do(newReq)

	var bodyStr string
	if body != nil {
		bodyStr = body.String()
	}
	return httputils.ParseJSONResponse(bodyStr, resp, err, api.debug)
}

func (api *SCephAdminApi) GetUsage(ctx context.Context, uid string) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/usage?format=json&uid=%s&show-entries=False&show-summary=True", api.adminPath, uid)
	_, resp, err := api.jsonRequest(ctx, httputils.GET, path, nil, nil)
	return resp, err
}

// {"caps":[{"perm":"*","type":"buckets"},{"perm":"*","type":"usage"},{"perm":"*","type":"users"}],
// "display_name":"First User","email":"",
// "keys":[{"access_key":"70TTA6IU5D9LQ0IVA3Z6","secret_key":"AK3Kd9L1elPin9wEXNRuY9L2yxZ5U3mGsGYoyIxL","user":"testuser"}],
// "max_buckets":1000,"subusers":[],"suspended":0,"swift_keys":[],"tenant":"","user_id":"testuser"}

type SUserInfo struct {
	Caps        []SUserCapability
	DisplayName string
	Email       string
	UserId      string
	Tenant      string
	Suspended   int
	SubUsers    []string
	MaxBuckets  int
	Keys        []SUserAccessKey
}

type SUserCapability struct {
	Perm string
	Type string
}

type SUserAccessKey struct {
	AccessKey string
	SecretKey string
	User      string
}

func (api *SCephAdminApi) GetUserInfo(ctx context.Context, uid string) (*SUserInfo, error) {
	path := fmt.Sprintf("/%s/user?format=json&uid=%s", api.adminPath, uid)
	_, resp, err := api.jsonRequest(ctx, httputils.GET, path, nil, nil)
	if err != nil {
		if httputils.ErrorCode(err) == 403 {
			msg := `add users read cap by: radosgw-admin caps add --uid %s --caps="users=read"`
			return nil, errors.Wrapf(cloudprovider.ErrForbidden, msg, uid)
		}
		return nil, errors.Wrap(err, "api.jsonRequest")
	}
	usrInfo := SUserInfo{}
	err = resp.Unmarshal(&usrInfo)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return &usrInfo, nil
}

type SQuotaQuery struct {
	QuotaType string `json:"quota-type"`
	Uid       string `json:"uid"`
	Bucket    string `json:"bucket"`
}

type SQuota struct {
	Enabled    tristate.TriState `json:"enabled"`
	CheckOnRaw bool              `json:"check_on_raw,omitfalse"`
	MaxSize    int64             `json:"max_size,omitzero"`
	MaxSizeKB  int64             `json:"max_size_kb,omitzero"`
	MaxObjects int               `json:"max_objects,omitzero"`
}

func (q SQuotaQuery) Query() string {
	return "quota&format=json&" + jsonutils.Marshal(q).QueryString()
}

func (q *SQuotaQuery) SetBucket(uid string, level string, bucket string) {
	q.Uid = uid
	q.Bucket = bucket
	q.QuotaType = level
}

func (api *SCephAdminApi) GetUserQuota(ctx context.Context, uid string) (*SQuota, *SQuota, error) {
	userQuota, err := api.getQuota(ctx, uid, "user", "")
	if err != nil {
		return nil, nil, errors.Wrap(err, "api.getQuota user level")
	}
	bucketQuota, err := api.getQuota(ctx, uid, "bucket", "")
	if err != nil {
		return nil, nil, errors.Wrap(err, "api.getQuota bucket level")
	}
	return userQuota, bucketQuota, nil
}

// ceph目前不支持设置bucket的quota，因此返回全局的bucket quota
func (api *SCephAdminApi) GetBucketQuota(ctx context.Context, uid string, bucket string) (*SQuota, error) {
	/*quota, err := api.getQuota(ctx, uid, "", bucket)
	if err == nil {
		return quota, nil
	}*/
	return api.getQuota(ctx, uid, "bucket", "")
}

func (api *SCephAdminApi) getQuota(ctx context.Context, uid string, level string, bucket string) (*SQuota, error) {
	query := SQuotaQuery{}
	query.SetBucket(uid, level, bucket)
	var path string
	if len(bucket) > 0 {
		path = fmt.Sprintf("/%s/bucket?%s", api.adminPath, query.Query())
	} else {
		path = fmt.Sprintf("/%s/user?%s", api.adminPath, query.Query())
	}
	_, resp, err := api.jsonRequest(ctx, httputils.GET, path, nil, nil)
	if err != nil {
		if httputils.ErrorCode(err) == 403 {
			var msg string
			if len(bucket) > 0 {
				msg = `add buckets read cap by: radosgw-admin caps add --uid %s --caps="buckets=read"`
			} else {
				msg = `add users read cap by: radosgw-admin caps add --uid %s --caps="users=read"`
			}
			return nil, errors.Wrapf(cloudprovider.ErrForbidden, msg, uid)
		}
		return nil, errors.Wrap(err, "api.jsonRequest")
	}
	if resp == nil {
		return nil, cloudprovider.ErrNotSupported
	}
	quota := SQuota{}
	err = resp.Unmarshal(&quota)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return &quota, nil
}

func (api *SCephAdminApi) SetUserQuota(ctx context.Context, uid string, sizeBytes int64, objects int) error {
	_, err := api.setQuota(ctx, uid, "user", "", sizeBytes, objects)
	return errors.Wrap(err, "api.setQuota")
}

func (api *SCephAdminApi) SetAllBucketQuota(ctx context.Context, uid string, sizeBytes int64, objects int) error {
	_, err := api.setQuota(ctx, uid, "bucket", "", sizeBytes, objects)
	return errors.Wrap(err, "api.setQuota")
}

// ceph目前不支持设置quota，因此返回全局的bucket quota
func (api *SCephAdminApi) SetBucketQuota(ctx context.Context, uid string, bucket string, sizeBytes int64, objects int) error {
	var err error
	_, err = api.setQuota(ctx, uid, "", bucket, sizeBytes, objects)
	if err == nil {
		return nil
	}
	_, err = api.setQuota(ctx, uid, "bucket", "", sizeBytes, objects)
	if err == nil {
		return nil
	}
	_, err = api.setQuota(ctx, uid, "user", "", sizeBytes, objects)
	if err == nil {
		return nil
	}
	return errors.Wrap(err, "api.setQuota")
}

func (api *SCephAdminApi) setQuota(ctx context.Context, uid string, level string, bucket string, sizeBytes int64, objects int) (*SQuota, error) {
	query := SQuotaQuery{}
	query.SetBucket(uid, level, bucket)
	quota := SQuota{
		MaxSize:    sizeBytes,
		MaxObjects: objects,
	}
	if sizeBytes <= 0 && objects <= 0 {
		quota.Enabled = tristate.False
	} else {
		quota.Enabled = tristate.True
	}
	var path string
	if len(bucket) > 0 {
		path = fmt.Sprintf("/%s/bucket?%s", api.adminPath, query.Query())
	} else {
		path = fmt.Sprintf("/%s/user?%s", api.adminPath, query.Query())
	}
	body := jsonutils.Marshal(quota)
	log.Debugf("request %s %s %s", httputils.PUT, path, body)
	_, resp, err := api.jsonRequest(ctx, httputils.PUT, path, nil, body)
	if err != nil {
		if httputils.ErrorCode(err) == 403 {
			var msg string
			if len(bucket) > 0 {
				msg = `add buckets write cap by: radosgw-admin caps add --uid %s --caps="buckets=write"`
			} else {
				msg = `add users write cap by: radosgw-admin caps add --uid %s --caps="users=write"`
			}
			return nil, errors.Wrapf(cloudprovider.ErrForbidden, msg, uid)
		}
		return nil, errors.Wrap(err, "api.jsonRequest")
	}
	log.Debugf("%s", resp)
	quota = SQuota{}
	err = resp.Unmarshal(&quota)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return &quota, nil
}
