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

package qcloud

import (
	"fmt"
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SCdnOrigin struct {
	Origins            []string      `json:"Origins"`
	OriginType         string        `json:"OriginType"`
	ServerName         string        `json:"ServerName"`
	CosPrivateAccess   string        `json:"CosPrivateAccess"`
	OriginPullProtocol string        `json:"OriginPullProtocol"`
	BackupOrigins      []interface{} `json:"BackupOrigins"`
	BackupOriginType   interface{}   `json:"BackupOriginType"`
	BackupServerName   interface{}   `json:"BackupServerName"`
}
type SCdnDomain struct {
	Area        string     `json:"Area"`
	Cname       string     `json:"Cname"`
	CreateTime  string     `json:"CreateTime"`
	Disable     string     `json:"Disable"`
	Domain      string     `json:"Domain"`
	Origin      SCdnOrigin `json:"Origin"`
	ProjectID   int        `json:"ProjectId"`
	Readonly    string     `json:"Readonly"`
	ResourceID  string     `json:"ResourceId"`
	ServiceType string     `json:"ServiceType"`
	Status      string     `json:"Status"`
	UpdateTime  string     `json:"UpdateTime"`
}

type SDomains struct {
	RequestID   string       `json:"RequestId"`
	Domains     []SCdnDomain `json:"Domains"`
	TotalNumber int          `json:"TotalNumber"`
}

func (client *SQcloudClient) AddCdnDomain(domain string, originType string, origins []string, cosPrivateAccess string) error {
	params := map[string]string{}
	params["Domain"] = domain
	params["ServiceType"] = "web"
	for i := range origins {
		params[fmt.Sprintf("Origin.Origins.%d", i)] = origins[i]
	}
	params["Origin.OriginType"] = originType
	params["Origin.CosPrivateAccess"] = cosPrivateAccess
	_, err := client.cdnRequest("AddCdnDomain", params)
	if err != nil {
		return errors.Wrapf(err, ` client.cdnRequest("AddCdnDomain", %s)`, jsonutils.Marshal(params).String())
	}
	return nil
}

func (client *SQcloudClient) DescribeCdnDomains(domains, origins []string, domainType string, offset int, limit int) ([]SCdnDomain, int, error) {
	params := map[string]string{}
	params["Offset"] = strconv.Itoa(offset)
	params["Limit"] = strconv.Itoa(limit)
	filterIndex := 0
	if len(domains) > 0 {
		params[fmt.Sprintf("Filters.%d.Name", filterIndex)] = "domain"
		for i := range domains {
			params[fmt.Sprintf("Filters.%d.Value.%d", filterIndex, i)] = domains[i]
		}
		filterIndex++
	}
	if len(origins) > 0 {
		params[fmt.Sprintf("Filters.%d.Name", filterIndex)] = "origin"
		for i := range origins {
			params[fmt.Sprintf("Filters.%d.Value.%d", filterIndex, i)] = origins[i]
		}
		filterIndex++
	}

	if len(domainType) > 0 {
		params[fmt.Sprintf("Filters.%d.Name", filterIndex)] = "domainType"
		params[fmt.Sprintf("Filters.%d.Value.0", filterIndex)] = domainType
		filterIndex++
	}

	resp, err := client.cdnRequest("DescribeDomainsConfig", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeDomainsConfig %s", params)
	}
	cdnDomains := []SCdnDomain{}
	err = resp.Unmarshal(&cdnDomains, "Domains")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "%s.Unmarshal(records)", jsonutils.Marshal(resp).String())
	}
	totalcount, _ := resp.Float("TotalNumber")
	return cdnDomains, int(totalcount), nil
}

func (client *SQcloudClient) DescribeAllCdnDomains(domains, origins []string, domainType string) ([]SCdnDomain, error) {
	cdnDomains := make([]SCdnDomain, 0)
	for {
		part, total, err := client.DescribeCdnDomains(domains, origins, domainType, len(cdnDomains), 50)
		if err != nil {
			return nil, errors.Wrap(err, "client.DescribeCdnDomains(domains, origins, len(cdnDomains), 50)")
		}
		cdnDomains = append(cdnDomains, part...)
		if len(cdnDomains) >= total {
			break
		}
	}
	return cdnDomains, nil
}
