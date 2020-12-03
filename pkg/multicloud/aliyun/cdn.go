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

package aliyun

import (
	"time"

	"github.com/pkg/errors"
)

type SCdnDomainNames struct {
	DomainNames []string `json:"domainNames"`
}
type SDomainInfo struct {
	DomainCname string    `json:"DomainCname"`
	Status      string    `json:"Status"`
	CreateTime  time.Time `json:"CreateTime"`
	UpdateTime  time.Time `json:"UpdateTime"`
	DomainName  string    `json:"DomainName"`
}
type SCdnDomainInfos struct {
	DomainInfo []SDomainInfo `json:"domainInfo"`
}
type SCdnDomainsData struct {
	Source      string          `json:"Source"`
	Domains     SCdnDomainNames `json:"Domains"`
	DomainInfos SCdnDomainInfos `json:"DomainInfos"`
}
type SCdnDomainsList struct {
	DomainsData []SCdnDomainsData `json:"DomainsData"`
}

type SCdnSource struct {
	Port     int    `json:"Port"`
	Weight   string `json:"Weight"`
	Type     string `json:"Type"`
	Content  string `json:"Content"`
	Priority string `json:"Priority"`
}
type SCdnSources struct {
	Source []SCdnSource `json:"Source"`
}
type SCdnPageData struct {
	Cname           string     `json:"Cname"`
	Description     string     `json:"Description"`
	CdnType         string     `json:"CdnType"`
	ResourceGroupID string     `json:"ResourceGroupId"`
	DomainStatus    string     `json:"DomainStatus"`
	SslProtocol     string     `json:"SslProtocol"`
	DomainName      string     `json:"DomainName"`
	Coverage        string     `json:"Coverage"`
	Sources         SCdnSource `json:"Sources"`
	GmtModified     string     `json:"GmtModified"`
	Sandbox         string     `json:"Sandbox"`
	GmtCreated      time.Time  `json:"GmtCreated"`
}
type SCdnDomains struct {
	PageData []SCdnPageData `json:"PageData"`
}

func (client *SAliyunClient) DescribeDomainsBySource(origin string) (SCdnDomainsList, error) {
	sproducts := SCdnDomainsList{}
	params := map[string]string{}
	params["Action"] = "DescribeDomainsBySource"
	params["Sources"] = origin
	resp, err := client.cdnRequest("DescribeDomainsBySource", params)
	if err != nil {
		return sproducts, errors.Wrap(err, "DescribeDomainsBySource")
	}
	err = resp.Unmarshal(&sproducts, "DomainsList")
	if err != nil {
		return sproducts, errors.Wrap(err, "resp.Unmarshal")
	}
	return sproducts, nil
}

func (client *SAliyunClient) DescribeUserDomains(domain string) (SCdnDomains, error) {
	sproducts := SCdnDomains{}
	params := map[string]string{}
	params["Action"] = "DescribeUserDomains"
	params["DomainName"] = domain
	params["DomainSearchType"] = "full_match"
	resp, err := client.cdnRequest("DescribeUserDomains", params)
	if err != nil {
		return sproducts, errors.Wrap(err, "DescribeUserDomains")
	}
	err = resp.Unmarshal(&sproducts, "Domains")
	if err != nil {
		return sproducts, errors.Wrap(err, "resp.Unmarshal")
	}
	return sproducts, nil
}
