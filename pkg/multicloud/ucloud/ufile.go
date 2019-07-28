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
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"yunion.io/x/onecloud/pkg/multicloud/objectstore"

	"yunion.io/x/log"

	"context"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type SBucket struct {
	objectstore.SBucket
	region *SRegion

	projectId string

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

	// BucketName string
	File     io.Reader
	FileSize int64
	FileName string
	FileMD5  string
}

func (self *SFile) signHeader(httpMethod string) string {
	md5 := ""
	contentType := ""
	if httpMethod == http.MethodPut {
		md5 = self.FileMD5
		contentType = "application/octet-stream"
	}

	data := httpMethod + "\n"
	data += md5 + "\n"
	data += contentType + "\n"
	data += "\n"
	data += "/" + self.bucket.BucketName + "/" + self.FileName

	h := hmac.New(sha1.New, []byte(self.bucket.region.client.accessKeySecret))
	h.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (self *SFile) auth(httpMethod string) string {
	return "UCloud" + " " + self.bucket.region.client.accessKeyId + ":" + self.signHeader(httpMethod)
}

func (self *SFile) GetHost() string {
	return self.bucket.Domain.Src[0]
	/*host, err := self.bucket.region.GetBucketDomain(self.BucketName)
	if err != nil {
		log.Errorf("SFile GetHost %s", err)
		return ""
	}
	return host*/
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
	req, _ := http.NewRequest(http.MethodPut, self.GetUrl(), self.File)
	req.Header.Add("Authorization", self.auth(http.MethodPut))
	req.Header.Add("Content-MD5", self.FileMD5)
	req.Header.Add("Content-Type", "application/octet-stream")
	req.Header.Add("Content-Length", strconv.FormatInt(self.FileSize, 10))

	return self.request(req)
}

func (self *SFile) Delete() error {
	req, _ := http.NewRequest(http.MethodDelete, self.GetUrl(), nil)
	req.Header.Add("Authorization", self.auth(http.MethodDelete))
	return self.request(req)
}

func (self *SFile) request(req *http.Request) error {
	res, err := httputils.GetDefaultClient().Do(req)
	if err != nil {
		return err
	}

	_, _, err = httputils.ParseJSONResponse(res, err, false)
	if err != nil {
		log.Errorf("SFile %s", err.Error())
		return err
	}

	return nil
}

func (b *SBucket) GetProjectId() string {
	return b.projectId
}

func (b *SBucket) GetGlobalId() string {
	return b.BucketID
}

func (b *SBucket) GetName() string {
	return b.BucketName
}

func (b *SBucket) GetLocation() string {
	return b.Region
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

func (b *SBucket) GetAcl() string {
	return b.Type
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
		ret = append(ret, cloudprovider.SBucketAccessUrl{
			Url:         u,
			Description: fmt.Sprintf("src%d", i),
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

func (b *SBucket) GetIObjects(prefix string, isRecursive bool) ([]cloudprovider.ICloudObject, error) {
	return cloudprovider.GetIObjects(b, prefix, isRecursive)
}

func (b *SBucket) ListObjects(prefix string, marker string, delimiter string, maxCount int) (cloudprovider.SListObjectResult, error) {
	result := cloudprovider.SListObjectResult{}
	return result, cloudprovider.ErrNotSupported
}

func (b *SBucket) PutObject(ctx context.Context, key string, reader io.Reader, contType string, storageClassStr string) error {
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
