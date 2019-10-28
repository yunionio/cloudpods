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

	"github.com/coredns/coredns/plugin/pkg/log"

	"yunion.io/x/onecloud/pkg/util/httputils"
)

type SBucket struct {
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
	region *SRegion

	BucketName string
	File       io.Reader
	FileSize   int64
	FileName   string
	FileMD5    string
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
	data += "/" + self.BucketName + "/" + self.FileName

	h := hmac.New(sha1.New, []byte(self.region.client.accessKeySecret))
	h.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (self *SFile) auth(httpMethod string) string {
	return "UCloud" + " " + self.region.client.accessKeyId + ":" + self.signHeader(httpMethod)
}

func (self *SFile) GetHost() string {
	host, err := self.region.GetBucketDomain(self.BucketName)
	if err != nil {
		log.Errorf("SFile GetHost %s", err)
		return ""
	}
	return host
}

func (self *SFile) GetUrl() string {
	return fmt.Sprintf("http://%s/%s", self.GetHost(), self.FileName)
}

// https://github.com/ufilesdk-dev/ufile-gosdk/blob/master/auth.go
func (self *SFile) FetchFileUrl() string {
	expired := strconv.FormatInt(time.Now().Add(6*time.Hour).Unix(), 10)
	// sign
	data := "GET\n\n\n" + expired + "\n"
	data += "/" + self.BucketName + "/" + self.FileName
	h := hmac.New(sha1.New, []byte(self.region.client.accessKeySecret))
	h.Write([]byte(data))
	sign := base64.StdEncoding.EncodeToString(h.Sum(nil))

	urlEncoder := url.Values{}
	urlEncoder.Add("UCloudPublicKey", self.region.client.accessKeyId)
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
	res, err := http.DefaultClient.Do(req)
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
