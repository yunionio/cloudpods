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

package ucloud

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type SBucket struct {
	multicloud.SBaseBucket

	region *SRegion

	// projectId string

	Domain        Domain   `json:"Domain"`
	BucketID      string   `json:"BucketId"`
	Region        string   `json:"Region"`
	CreateTime    int64    `json:"CreateTime"`
	Biz           string   `json:"Biz"`
	BucketName    string   `json:"BucketName"`
	ModifyTime    int64    `json:"ModifyTime"`
	Type          string   `json:"Type"`
	Tag           string   `json:"Tag"`
	HasUserDomain int64    `json:"HasUserDomain"`
	CDNDomainID   []string `json:"CdnDomainId"`
}

type Domain struct {
	Src       []string      `json:"Src"`
	CDN       []string      `json:"Cdn"`
	CustomCDN []interface{} `json:"CustomCdn"`
	CustomSrc []interface{} `json:"CustomSrc"`
}

type SFile struct {
	bucket *SBucket

	BucketName   string `json:"BucketName"`
	FileName     string `json:"FileName"`
	Size         int64  `json:"Size"`
	Hash         string `json:"Hash"`
	MimeType     string `json:"MimeType"`
	CreateTime   int64  `json:"CreateTime"`
	ModifyTime   int64  `json:"ModifyTime"`
	StorageClass string `json:"StorageClass"`

	file io.Reader
}

func (client *SUcloudClient) signHeader(httpMethod string, path string, md5 string) string {
	contentType := ""
	if httpMethod == http.MethodPut {
		contentType = "application/octet-stream"
	}

	data := httpMethod + "\n"
	data += md5 + "\n"
	data += contentType + "\n"
	data += "\n"
	data += path

	log.Debugf("sign %s", data)
	h := hmac.New(sha1.New, []byte(client.accessKeySecret))
	h.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (self *SFile) signHeader(httpMethod string) string {
	return self.bucket.region.client.signHeader(httpMethod, "/"+self.bucket.BucketName+"/"+self.FileName, self.Hash)
}

func (self *SFile) auth(httpMethod string) string {
	return "UCloud" + " " + self.bucket.region.client.accessKeyId + ":" + self.signHeader(httpMethod)
}

func (self *SFile) GetHost() string {
	return self.bucket.Domain.Src[0]
}

func (self *SFile) GetUrl() string {
	return fmt.Sprintf("http://%s/%s", self.GetHost(), self.FileName)
}

// https://github.com/ufilesdk-dev/ufile-gosdk/blob/master/auth.go
func (self *SFile) FetchFileUrl() string {
	expired := strconv.FormatInt(time.Now().Add(6*time.Hour).Unix(), 10)
	// sign
	data := "GET\n\n\n" + expired + "\n"
	data += "/" + self.bucket.BucketName + "/" + self.FileName
	h := hmac.New(sha1.New, []byte(self.bucket.region.client.accessKeySecret))
	h.Write([]byte(data))
	sign := base64.StdEncoding.EncodeToString(h.Sum(nil))

	urlEncoder := url.Values{}
	urlEncoder.Add("UCloudPublicKey", self.bucket.region.client.accessKeyId)
	urlEncoder.Add("Signature", sign)
	urlEncoder.Add("Expires", expired)
	querys := urlEncoder.Encode()
	return fmt.Sprintf("%s?%s", self.GetUrl(), querys)
}

func (self *SFile) Upload() error {
	req, _ := http.NewRequest(http.MethodPut, self.GetUrl(), self.file)
	req.Header.Add("Authorization", self.auth(http.MethodPut))
	req.Header.Add("Content-MD5", self.Hash)
	req.Header.Add("Content-Type", "application/octet-stream")
	req.Header.Add("Content-Length", strconv.FormatInt(self.Size, 10))
	_, err := doRequest(req)
	return err
}

func (self *SFile) Delete() error {
	req, _ := http.NewRequest(http.MethodDelete, self.GetUrl(), nil)
	req.Header.Add("Authorization", self.auth(http.MethodDelete))
	_, err := doRequest(req)
	return err
}

func (self *SFile) GetIBucket() cloudprovider.ICloudBucket {
	return self.bucket
}

func (self *SFile) GetKey() string {
	return self.FileName
}

func (self *SFile) GetSizeBytes() int64 {
	return self.Size
}

func (self *SFile) GetLastModified() time.Time {
	return time.Unix(self.ModifyTime, 0)
}

func (self *SFile) GetStorageClass() string {
	return self.StorageClass
}

func (self *SFile) GetETag() string {
	return self.Hash
}

func (self *SFile) GetContentType() string {
	return self.MimeType
}

func (self *SFile) GetAcl() cloudprovider.TBucketACLType {
	return self.bucket.GetAcl()
}

func (self *SFile) SetAcl(cloudprovider.TBucketACLType) error {
	return nil
}

func (self *SFile) GetMeta() http.Header {
	return nil
}

func (self *SFile) SetMeta(ctx context.Context, meta http.Header) error {
	return cloudprovider.ErrNotSupported
}

func doRequest(req *http.Request) (jsonutils.JSONObject, error) {
	res, err := httputils.GetDefaultClient().Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "httpclient Do")
	}
	_, body, err := httputils.ParseJSONResponse(res, err, false)
	if err != nil {
		return nil, errors.Wrap(err, "ParseJSONResponse")
	}
	return body, nil
}

type sPrefixFileListOutput struct {
	BucketName string
	BucketId   string
	NextMarker string
	DataSet    []SFile
}

func (b *SBucket) doPrefixFileList(prefix string, marker string, limit int) (*sPrefixFileListOutput, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(""), "list")
	if len(prefix) > 0 {
		params.Add(jsonutils.NewString(prefix), "prefix")
	}
	if len(marker) > 0 {
		params.Add(jsonutils.NewString(marker), "marker")
	}
	if limit > 0 {
		params.Add(jsonutils.NewInt(int64(limit)), "limit")
	}
	host := fmt.Sprintf("https://%s.ufile.ucloud.cn", b.BucketName)
	path := fmt.Sprintf("/?%s", params.QueryString())

	log.Debugf("Request %s%s", host, path)

	req, _ := http.NewRequest(http.MethodGet, host+path, nil)

	sign := b.region.client.signHeader(http.MethodGet, path, "")
	auth := "UCloud" + " " + b.region.client.accessKeyId + ":" + sign

	req.Header.Add("Authorization", auth)

	output := sPrefixFileListOutput{}

	body, err := doRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "doRequest")
	}
	err = body.Unmarshal(&output)
	if err != nil {
		return nil, errors.Wrap(err, "body.Unmarshal")
	}

	return &output, nil
}

func (b *SBucket) GetProjectId() string {
	return b.region.client.projectId
}

func (b *SBucket) GetGlobalId() string {
	return b.BucketID
}

func (b *SBucket) GetName() string {
	return b.BucketName
}

func (b *SBucket) GetLocation() string {
	return b.region.GetId()
}

func (b *SBucket) GetIRegion() cloudprovider.ICloudRegion {
	return b.region
}

func (b *SBucket) GetCreateAt() time.Time {
	return time.Unix(b.CreateTime, 0)
}

func (b *SBucket) GetStorageClass() string {
	return ""
}

func (b *SBucket) GetAcl() cloudprovider.TBucketACLType {
	switch b.Type {
	case "public":
		return cloudprovider.ACLPublicRead
	default:
		return cloudprovider.ACLPrivate
	}
}

func (b *SBucket) SetAcl(aclStr cloudprovider.TBucketACLType) error {
	aclType := "private"
	if aclStr == cloudprovider.ACLPublicRead || aclStr == cloudprovider.ACLPublicReadWrite {
		aclType = "public"
	}
	return b.region.updateBucket(b.BucketName, aclType)
}

func (b *SBucket) getSrcUrl() string {
	if len(b.Domain.Src) > 0 {
		return b.Domain.Src[0]
	}
	return ""
}

func (b *SBucket) GetAccessUrls() []cloudprovider.SBucketAccessUrl {
	ret := make([]cloudprovider.SBucketAccessUrl, 0)
	for i, u := range b.Domain.Src {
		primary := false
		if i == 0 {
			primary = true
		}
		ret = append(ret, cloudprovider.SBucketAccessUrl{
			Url:         u,
			Description: fmt.Sprintf("src%d", i),
			Primary:     primary,
		})
	}
	for i, u := range b.Domain.CDN {
		ret = append(ret, cloudprovider.SBucketAccessUrl{
			Url:         u,
			Description: fmt.Sprintf("cdn%d", i),
		})
	}
	return ret
}

func (b *SBucket) GetStats() cloudprovider.SBucketStats {
	stats, _ := cloudprovider.GetIBucketStats(b)
	return stats
}

func (b *SBucket) ListObjects(prefix string, marker string, delimiter string, maxCount int) (cloudprovider.SListObjectResult, error) {
	result := cloudprovider.SListObjectResult{}

	output, err := b.doPrefixFileList(prefix, marker, maxCount)
	if err != nil {
		return result, errors.Wrap(err, "b.doPrefixFileList")
	}

	if len(output.NextMarker) > 0 {
		result.NextMarker = output.NextMarker
		result.IsTruncated = true
	}

	result.Objects = make([]cloudprovider.ICloudObject, len(output.DataSet))
	for i := range output.DataSet {
		result.Objects[i] = &output.DataSet[i]
	}

	return result, nil
}

func (b *SBucket) PutObject(ctx context.Context, key string, input io.Reader, sizeBytes int64, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	return cloudprovider.ErrNotSupported
}

func (b *SBucket) NewMultipartUpload(ctx context.Context, key string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (b *SBucket) UploadPart(ctx context.Context, key string, uploadId string, partIndex int, input io.Reader, partSize int64) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (b *SBucket) CompleteMultipartUpload(ctx context.Context, key string, uploadId string, partEtags []string) error {
	return cloudprovider.ErrNotSupported
}

func (b *SBucket) AbortMultipartUpload(ctx context.Context, key string, uploadId string) error {
	return cloudprovider.ErrNotSupported
}

func (b *SBucket) DeleteObject(ctx context.Context, key string) error {
	file := SFile{
		bucket:   b,
		FileName: key,
	}
	return file.Delete()
}

func (b *SBucket) GetTempUrl(method string, key string, expire time.Duration) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (b *SBucket) CopyObject(ctx context.Context, destKey string, srcBucket, srcKey string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	return cloudprovider.ErrNotSupported
}

func (b *SBucket) GetObject(ctx context.Context, key string, rangeOpt *cloudprovider.SGetObjectRange) (io.ReadCloser, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (b *SBucket) CopyPart(ctx context.Context, key string, uploadId string, partIndex int, srcBucketName string, srcKey string, srcOffset int64, srcLength int64) (string, error) {
	return "", cloudprovider.ErrNotSupported
}
