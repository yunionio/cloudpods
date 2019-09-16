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

package xsky

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"math/rand"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SXskyAdminApi struct {
	endpoint string
	username string
	password string
	token    *sLoginResponse
	client   *http.Client
	debug    bool
}

func newXskyAdminApi(user, passwd, ep string, debug bool) *SXskyAdminApi {
	return &SXskyAdminApi{
		endpoint: ep,
		username: user,
		password: passwd,
		client:   httputils.GetDefaultClient(),
		debug:    debug,
	}
}

func getJsonBodyReader(body jsonutils.JSONObject) io.Reader {
	var reqBody io.Reader
	if body != nil {
		reqBody = strings.NewReader(body.String())
	}
	return reqBody
}

func (api *SXskyAdminApi) jsonRequest(ctx context.Context, method httputils.THttpMethod, path string, hdr http.Header, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
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

	if api.isValidToken() {
		req.Header.Set("xms-auth-token", api.token.Token.Uuid)
	}

	if api.debug {
		log.Debugf("request: %s %s %s %s", method, urlStr, req.Header, body)
	}
	resp, err := api.client.Do(req)

	return httputils.ParseJSONResponse(resp, err, api.debug)
}

type sLoginResponse struct {
	Token struct {
		Create  time.Time
		Expires time.Time
		Roles   []string
		User struct {
			Create             time.Time
			Name               string
			Email              string
			Enabled            bool
			Id                 int
			PasswordLastUpdate time.Time
		}
		Uuid  string
		Valid bool
	}
}

func (api *SXskyAdminApi) isValidToken() bool {
	if api.token != nil && len(api.token.Token.Uuid) > 0 && api.token.Token.Expires.After(time.Now()) {
		return true
	} else {
		return false
	}
}

func (api *SXskyAdminApi) auth(ctx context.Context) (*sLoginResponse, error) {
	input := STokenCreateReq{}
	input.Auth.Identity.Password.User.Name = api.username
	input.Auth.Identity.Password.User.Password = api.password

	_, resp, err := api.jsonRequest(ctx, httputils.POST, "/api/v1/auth/tokens", nil, jsonutils.Marshal(input))
	if err != nil {
		return nil, errors.Wrap(err, "api.jsonRequest")
	}
	output := sLoginResponse{}
	err = resp.Unmarshal(&output)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return &output, err
}

type STokenCreateReq struct {
	Auth STokenCreateReqAuth `json:"auth"`
}

type STokenCreateReqAuth struct {
	Identity STokenCreateReqAuthIdentity `json:"identity"`
}

type STokenCreateReqAuthIdentity struct {
	// password for auth
	Password SAuthPasswordReq `json:"password,omitempty"`
	// token for auth
	Token SAuthTokenReq `json:"token,omitempty"`
}

type SAuthPasswordReq struct {
	User SAuthPasswordReqUser `json:"user"`
}

type SAuthPasswordReqUser struct {
	// user email for auth
	Email string `json:"email,omitempty"`
	// user id for auth
	Id int64 `json:"id,omitzero"`
	// user name or email for auth
	Name string `json:"name,omitempty"`
	// password for auth
	Password string `json:"password"`
}

type SAuthTokenReq struct {
	// uuid of authorized token
	Uuid string `json:"uuid"`
}

func (api *SXskyAdminApi) authRequest(ctx context.Context, method httputils.THttpMethod, path string, hdr http.Header, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	if !api.isValidToken() {
		loginResp, err := api.auth(ctx)
		if err != nil {
			return nil, nil, errors.Wrap(err, "api.auth")
		}
		api.token = loginResp
	}
	return api.jsonRequest(ctx, method, path, hdr, body)
}

type sUser struct {
	BucketNum             int
	BucketQuotaMaxObjects int
	BucketQuotaMaxSize    int64
	Create                time.Time
	DisplayName           string
	Email                 string
	Id                    int
	MaxBuckets            int
	Name                  string
	OpMask                string
	Status                string
	Suspended             bool
	Update                time.Time
	UserQuotaMaxObjects   int
	UserQuotaMaxSize      int64
	samples               []sSample
	Keys                  []sKey
}

func (u sUser) getMinKey() string {
	minKey := ""
	for i := range u.Keys {
		if len(minKey) == 0 || minKey > u.Keys[i].AccessKey {
			minKey = u.Keys[i].AccessKey
		}
	}
	return minKey
}

type sKey struct {
	AccessKey string
	Create    time.Time
	Id        int
	Reserved  bool
	SecretKey string
	Status    string
	Type      string
	Update    time.Time
	User struct {
		Id   int
		Name string
	}
}

type sSample struct {
	AllocatedObjects    int
	AllocatedSize       int64
	Create              time.Time
	DelOpsPm            int
	RxBandwidthKbyte    int
	RxOpsPm             int
	TxBandwidthKbyte    int
	TxOpsPm             int
	TotalDelOps         int
	TotalDelSuccessOps  int
	TotalRxBytes        int64
	TotalRxOps          int
	TotalRxSuccessOps   int
	TotalTxBytes        int64
	TotalTxOps          int
	TotalTxSuccessKbyte int
}

type sPaging struct {
	Count      int
	Limit      int
	Offset     int
	TotalCount int
}

type sUsersResponse struct {
	OsUsers []sUser
	Paging  sPaging
}

func (api *SXskyAdminApi) getUsers(ctx context.Context) ([]sUser, error) {
	usrs := make([]sUser, 0)
	totalCount := 0
	for totalCount <= 0 || len(usrs) < totalCount {
		_, resp, err := api.authRequest(ctx, httputils.GET, fmt.Sprintf("/api/v1/os-users/?limit=1000&offset=%d", len(usrs)), nil, nil)
		if err != nil {
			return nil, errors.Wrap(err, "api.authRequest")
		}
		output := sUsersResponse{}
		err = resp.Unmarshal(&output)
		if err != nil {
			return nil, errors.Wrap(err, "resp.Unmarshal")
		}
		usrs = append(usrs, output.OsUsers...)
		totalCount = output.Paging.TotalCount
	}
	return usrs, nil
}

func (api *SXskyAdminApi) findUserByAccessKey(ctx context.Context, accessKey string) (*sUser, *sKey, error) {
	usrs, err := api.getUsers(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "api.getUsers")
	}
	for i := range usrs {
		for j := range usrs[i].Keys {
			if usrs[i].Keys[j].AccessKey == accessKey {
				return &usrs[i], &usrs[i].Keys[j], nil
			}
		}
	}
	return nil, nil, httperrors.ErrNotFound
}

type sBucket struct {
	ActionStatus       string
	AllUserPermission  string
	AuthUserPermission string
	BucketPolicy       string
	Create             time.Time
	Flag struct {
		Versioned         bool
		VersionsSuspended bool
		Worm              bool
	}
	Id int
	// LifeCycle
	MetadataSearchEnabled bool
	Name                  string
	NfsClientNum          int
	OsReplicationPathNum  int
	OsReplicationZoneNum  int
	// osZone
	OsZoneUuid string
	Owner struct {
		Id   string
		Name string
	}
	OwnerPermission string
	Policy          sPolicy
	PolicyEnabled   bool
	QuotaMaxObjects int
	QuotaMaxSize    int64
	// RemteClusters
	ReplicationUuid string
	Samples         []sSample
	Shards          int
	Status          string
	Update          time.Time
	Virtual         bool
	// NfsGatewayMaps
}

type sPolicy struct {
	BucketNum int
	Compress  bool
	Create    time.Time
	Crypto    bool
	DataPool struct {
		Id   int
		Name string
	}
	Default     bool
	Description string
	Id          int
	IdexPool struct {
		Id   int
		Name string
	}
	Name                string
	ObjectSizeThreshold int64
	PolicyName          string
	Status              string
	Update              time.Time
}

type sBucketsResponse struct {
	OsBuckets []sBucket
	Paging    sPaging
}

func (api *SXskyAdminApi) getBuckets(ctx context.Context) ([]sBucket, error) {
	buckets := make([]sBucket, 0)
	totalCount := 0
	for totalCount <= 0 || len(buckets) < totalCount {
		_, resp, err := api.authRequest(ctx, httputils.GET, fmt.Sprintf("/api/v1/os-buckets/?limit=1000&offset=%d", len(buckets)), nil, nil)
		if err != nil {
			return nil, errors.Wrap(err, "api.authRequest")
		}
		output := sBucketsResponse{}
		err = resp.Unmarshal(&output)
		if err != nil {
			return nil, errors.Wrap(err, "resp.Unmarshal")
		}
		buckets = append(buckets, output.OsBuckets...)
		totalCount = output.Paging.TotalCount
	}
	return buckets, nil
}

func (api *SXskyAdminApi) getBucketByName(ctx context.Context, name string) (*sBucket, error) {
	buckets, err := api.getBuckets(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "api.getBuckets")
	}
	for i := range buckets {
		if buckets[i].Name == name {
			return &buckets[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

type sBucketQuotaInput struct {
	OsBucket struct {
		QuotaMaxSize    int64
		QuotaMaxObjects int
	}
}

func (api *SXskyAdminApi) setBucketQuota(ctx context.Context, bucketId int, input sBucketQuotaInput) error {
	_, _, err := api.authRequest(ctx, httputils.PATCH, fmt.Sprintf("/api/v1/os-buckets/%d", bucketId), nil, jsonutils.Marshal(&input))
	if err != nil {
		return errors.Wrap(err, "api.authRequest")
	}
	return nil
}

type sS3LbGroup struct {
	ActionStatus    string
	Create          time.Time
	Description     string
	HttpsPort       int
	Id              int
	Name            string
	Port            int
	Roles           []string
	SearchHttpsPort int
	SearchPort      int
	Status          string
	SyncPort        int
	Update          time.Time
	S3LoadBalancers []sS3LoadBalancer `json:"s3_load_balancers"`
}

type sS3LoadBalancer struct {
	Create      time.Time
	Description string
	Group struct {
		Id     int
		Name   string
		Status string
	}
	Host struct {
		AdminIp string
		Id      int
		Name    string
	}
	HttpsPort     int
	Id            int
	InterfaceName string
	Ip            string
	Name          string
	Port          int
	Roles         []string
	Samples []struct {
		ActiveAconnects     int
		CpuUtil             float64
		Create              time.Time
		DownBandwidthKbytes int64
		FailureRequests     int
		MemUsagePercent     float64
		SuccessRequests     int
		UpBandwidthKbyte    int64
	}
	SearchHttpsPort int
	SearchPort      int
	SslCertificate  interface{}
	Status          string
	SyncPort        int
	Update          time.Time
	Vip             string
	VipMask         int
	Vips            string
}

func (lb sS3LoadBalancer) GetGatewayEndpoint() string {
	if lb.SslCertificate == nil {
		return fmt.Sprintf("http://%s:%d", lb.Vip, lb.Port)
	} else {
		return fmt.Sprintf("https://%s:%d", lb.Vip, lb.HttpsPort)
	}
}

type sS3LbGroupResponse struct {
	S3LoadBalancerGroups []sS3LbGroup `json:"s3_load_balancer_groups"`
	Paging               sPaging
}

func (api *SXskyAdminApi) getS3LbGroup(ctx context.Context) ([]sS3LbGroup, error) {
	lbGroups := make([]sS3LbGroup, 0)
	totalCount := 0
	for totalCount <= 0 || len(lbGroups) < totalCount {
		_, resp, err := api.authRequest(ctx, httputils.GET, fmt.Sprintf("/api/v1/s3-load-balancer-groups/?limit=1000&offset=%d", len(lbGroups)), nil, nil)
		if err != nil {
			return nil, errors.Wrap(err, "api.authRequest")
		}
		output := sS3LbGroupResponse{}
		err = resp.Unmarshal(&output)
		if err != nil {
			return nil, errors.Wrap(err, "resp.Unmarshal")
		}
		lbGroups = append(lbGroups, output.S3LoadBalancerGroups...)
		totalCount = output.Paging.TotalCount
	}
	return lbGroups, nil
}

func (api *SXskyAdminApi) getS3GatewayEndpoint(ctx context.Context) (string, error) {
	s3LbGrps, err := api.getS3LbGroup(ctx)
	if err != nil {
		return "", errors.Wrap(err, "api.getS3LbGroup")
	}
	lbs := make([]sS3LoadBalancer, 0)
	for i := range s3LbGrps {
		lbs = append(lbs, s3LbGrps[i].S3LoadBalancers...)
	}
	if len(lbs) == 0 {
		return "", errors.Wrap(httperrors.ErrNotFound, "empty S3 Lb group")
	}
	lb := lbs[rand.Intn(len(lbs))]
	return lb.GetGatewayEndpoint(), nil
}
