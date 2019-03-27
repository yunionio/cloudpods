package ucloud

import (
	"context"
	"crypto/sha1"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

const UCLOUD_API_HOST = "https://api.ucloud.cn"

// API返回结果对应的字段名
var UCLOUD_API_RESULT_KEYS = map[string]string{
	"AllocateEIP":            "EIPSet",
	"GetProjectList":         "ProjectSet",
	"GetRegion":              "Regions",
	"DescribeVPC":            "DataSet",
	"DescribeImage":          "ImageSet",
	"DescribeIsolationGroup": "IsolationGroupSet",
	"DescribeUHostInstance":  "UHostSet",
	"DescribeUHostTags":      "TagSet",
	"DescribeUDSet":          "UDSet",
	"DescribeUDisk":          "DataSet",
	"DescribeUDiskSnapshot":  "DataSet",
	"DescribeEIP":            "EIPSet",
	"DescribeFirewall":       "DataSet",
	"DescribeSubnet":         "DataSet",
	"DescribeBucket":         "DataSet",
	"CreateUDisk":            "UDiskId",
	"CreateVPC":              "VPCId",
	"CreateUDiskSnapshot":    "SnapshotId",
}

type SParams struct {
	data jsonutils.JSONDict
}

type SUcloudError struct {
	Action  string `json:"Action"`
	Message string `json:"Message"`
	RetCode int64  `json:"RetCode"`
}

func (self *SUcloudError) Error() string {
	return fmt.Sprintf("Do %s failed, code: %d, %s", self.Action, self.RetCode, self.Message)
}

func NewUcloudParams() SParams {
	data := jsonutils.NewDict()
	return SParams{data: *data}
}

func (self *SParams) Set(key string, value interface{}) {
	switch v := value.(type) {
	case string:
		self.data.Set(key, jsonutils.NewString(v))
	case int64:
		self.data.Set(key, jsonutils.NewInt(v))
	case int:
		self.data.Set(key, jsonutils.NewInt(int64(v)))
	case bool:
		self.data.Set(key, jsonutils.NewBool(v))
	case float64:
		self.data.Set(key, jsonutils.NewFloat(v))
	case float32:
		self.data.Set(key, jsonutils.NewFloat(float64(v)))
	case []string:
		self.data.Set(key, jsonutils.NewStringArray(v))
	default:
		log.Debugf("unsuported params type %T", value)
	}
}

func (self *SParams) SetAction(action string) {
	self.data.Set("Action", jsonutils.NewString(action))
}

func (self *SParams) SetPagination(limit, offset int) {
	if limit == 0 {
		limit = 20
	}

	self.data.Set("Limit", jsonutils.NewInt(int64(limit)))
	self.data.Set("Offset", jsonutils.NewInt(int64(offset)))
}

func (self *SParams) String() string {
	return self.data.String()
}

func (self *SParams) PrettyString() string {
	return self.data.PrettyString()
}

func (self *SParams) GetParams() jsonutils.JSONDict {
	return self.data
}

// https://docs.ucloud.cn/api/summary/signature
func BuildParams(params SParams, privateKey string) jsonutils.JSONObject {
	data := params.GetParams()
	// remove old Signature
	data.Remove("Signature")

	// 	排序并计算signture
	keys := data.SortedKeys()
	lst := []string{}
	for _, k := range keys {
		lst = append(lst, k)
		v, _ := data.GetString(k)
		lst = append(lst, v)
	}

	raw := strings.Join(lst, "") + privateKey
	signture := fmt.Sprintf("%x", sha1.Sum([]byte(raw)))

	data.Set("Signature", jsonutils.NewString(signture))
	return &data
}

func GetSignature(params SParams, privateKey string) string {
	sign, _ := BuildParams(params, privateKey).GetString("Signature")
	return sign
}

func parseUcloudResponse(resp jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := &SUcloudError{}
	e := resp.Unmarshal(err)
	if e != nil {
		return nil, e
	}

	if err.RetCode > 0 {
		return nil, err
	}

	return resp, nil
}

func jsonRequest(client *SUcloudClient, params SParams) (jsonutils.JSONObject, error) {
	ctx := context.Background()
	MAX_RETRY := 3
	retry := 0

	for retry < MAX_RETRY {
		_, resp, err := httputils.JSONRequest(
			client.httpClient,
			ctx,
			httputils.POST,
			UCLOUD_API_HOST,
			nil,
			BuildParams(params, client.accessKeySecret),
			client.Debug)

		if err == nil {
			return parseUcloudResponse(resp)
		}

		switch e := err.(type) {
		case *httputils.JSONClientError:
			if e.Code >= 500 {
				time.Sleep(3 * time.Second)
				retry += 1
				continue
			} else {
				return nil, err
			}
		default:
			return nil, err
		}
	}

	return nil, fmt.Errorf("timeout for request: %s", params)
}
