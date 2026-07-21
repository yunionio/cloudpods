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
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

type SNotice struct {
	title   string
	content string
}

func (notice *SNotice) GetTitle() string {
	return notice.title
}

func (notice *SNotice) GetContent() string {
	return notice.content
}

type SSecurityBroadcast struct {
	Id         int    `json:"Id"`
	Title      string `json:"Title"`
	Subtitle   string `json:"Subtitle"`
	CreateTime string `json:"CreateTime"`
	Type       int    `json:"Type"`
}

type SSecurityBroadcastInfo struct {
	Id         int    `json:"Id"`
	Title      string `json:"Title"`
	Subtitle   string `json:"Subtitle"`
	CreateTime string `json:"CreateTime"`
	Content    string `json:"Content"`
	Type       int    `json:"Type"`
	GotoType   int    `json:"GotoType"`
}

func (self *SQcloudClient) GetNotices() ([]cloudprovider.INotice, error) {
	broadcasts, err := self.listSecurityBroadcastsToday()
	if err != nil {
		return nil, err
	}
	ret := make([]cloudprovider.INotice, 0, len(broadcasts))
	for i := range broadcasts {
		broadcast := &broadcasts[i]
		pubAt, err := parseSecurityBroadcastTime(broadcast.CreateTime)
		if err != nil || !cloudprovider.IsNoticePublishedToday(pubAt) {
			continue
		}
		info, err := self.getSecurityBroadcastInfo(broadcast.Id)
		if err != nil {
			continue
		}
		title := strings.TrimSpace(info.Title)
		if len(title) == 0 {
			title = strings.TrimSpace(broadcast.Title)
		}
		ret = append(ret, &SNotice{
			title:   title,
			content: formatSecurityBroadcastContent(info),
		})
	}
	return ret, nil
}

func (self *SQcloudClient) listSecurityBroadcastsToday() ([]SSecurityBroadcast, error) {
	today := securityBroadcastToday()
	params := map[string]string{
		"BeginDate": today,
		"EndDate":   today,
		"Limit":     "100",
	}
	ret := []SSecurityBroadcast{}
	offset := 0
	total := 100
	for total > offset {
		params["Offset"] = strconv.Itoa(offset)
		resp, err := self.cwpRequest("DescribeSecurityBroadcasts", params)
		if err != nil {
			return nil, errors.Wrap(err, "DescribeSecurityBroadcasts")
		}
		part := []SSecurityBroadcast{}
		err = resp.Unmarshal(&part, "List")
		if err != nil {
			return nil, errors.Wrap(err, "Unmarshal List")
		}
		ret = append(ret, part...)
		_total, err := resp.Int("TotalCount")
		if err != nil {
			break
		}
		total = int(_total)
		offset += len(part)
		if len(part) == 0 {
			break
		}
	}
	return ret, nil
}

func (self *SQcloudClient) getSecurityBroadcastInfo(id int) (*SSecurityBroadcastInfo, error) {
	params := map[string]string{
		"Id": strconv.Itoa(id),
	}
	resp, err := self.cwpRequest("DescribeSecurityBroadcastInfo", params)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeSecurityBroadcastInfo")
	}
	info := &SSecurityBroadcastInfo{}
	err = resp.Unmarshal(info, "BroadcastInfo")
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal BroadcastInfo")
	}
	return info, nil
}

func formatSecurityBroadcastContent(info *SSecurityBroadcastInfo) string {
	content := stripHTML(info.Content)
	if len(content) == 0 {
		content = strings.TrimSpace(info.Subtitle)
	}
	if len(info.CreateTime) > 0 {
		if len(content) > 0 {
			content = fmt.Sprintf("%s\n\n发布时间: %s", content, info.CreateTime)
		} else {
			content = fmt.Sprintf("发布时间: %s", info.CreateTime)
		}
	}
	return content
}

func parseSecurityBroadcastTime(ts string) (time.Time, error) {
	ts = strings.TrimSpace(ts)
	if len(ts) == 0 {
		return time.Time{}, errors.Error("empty CreateTime")
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, ts, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.Errorf("parse security broadcast time %q", ts)
}

func securityBroadcastToday() string {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	return time.Now().In(loc).Format("2006-01-02")
}

func stripHTML(s string) string {
	if len(s) == 0 {
		return ""
	}
	s = html.UnescapeString(s)
	s = htmlTagRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "\u00a0", " ")
	lines := strings.Split(s, "\n")
	parts := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 0 {
			parts = append(parts, line)
		}
	}
	return strings.Join(parts, "\n")
}
