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
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const aliyunNoticeRssURL = "https://www.aliyun.com/rss/notice/zh.xml"

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

type sRssFeed struct {
	Channel sRssChannel `xml:"channel"`
}

type sRssChannel struct {
	Items []sRssItem `xml:"item"`
}

type sRssItem struct {
	Title          string `xml:"title"`
	Link           string `xml:"link"`
	Description    string `xml:"description"`
	PubDate        string `xml:"pubDate"`
	ContentEncoded string `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
}

func (self *SAliyunClient) GetNotices() ([]cloudprovider.INotice, error) {
	body, err := self.fetchNoticeRss()
	if err != nil {
		return nil, err
	}
	feed := &sRssFeed{}
	err = xml.Unmarshal(body, feed)
	if err != nil {
		return nil, errors.Wrap(err, "xml.Unmarshal")
	}
	ret := make([]cloudprovider.INotice, 0, len(feed.Channel.Items))
	for i := range feed.Channel.Items {
		item := &feed.Channel.Items[i]

		pubAt, err := parseRssPubDate(item.PubDate)
		if err != nil || !cloudprovider.IsNoticePublishedToday(pubAt) {
			continue
		}

		ret = append(ret, &SNotice{
			title:   strings.TrimSpace(item.Title),
			content: formatRssNoticeContent(item),
		})
	}
	return ret, nil
}

func parseRssPubDate(pubDate string) (time.Time, error) {
	pubDate = strings.TrimSpace(pubDate)
	if len(pubDate) == 0 {
		return time.Time{}, errors.New("empty pubDate")
	}
	return http.ParseTime(pubDate)
}

func (self *SAliyunClient) fetchNoticeRss() ([]byte, error) {
	client := httputils.GetAdaptiveTimeoutClient()
	httputils.SetClientProxyFunc(client, self.cpcfg.ProxyFunc)
	resp, err := httputils.Request(client, context.Background(), httputils.GET, aliyunNoticeRssURL, nil, nil, self.debug)
	if err != nil {
		return nil, errors.Wrap(err, "Request")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "ReadAll")
	}
	return body, nil
}

func formatRssNoticeContent(item *sRssItem) string {
	content := stripHTML(item.ContentEncoded)
	if len(content) == 0 {
		content = stripHTML(item.Description)
	}
	if len(item.Link) > 0 {
		if len(content) > 0 {
			content = fmt.Sprintf("%s\n\nLink: %s", content, item.Link)
		} else {
			content = item.Link
		}
	}
	return content
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
