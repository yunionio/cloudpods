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

package oracle

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SAnnouncementSummary struct {
	Id               string   `json:"id"`
	Summary          string   `json:"summary"`
	AnnouncementType string   `json:"announcementType"`
	LifecycleState   string   `json:"lifecycleState"`
	Services         []string `json:"services"`
	AffectedRegions  []string `json:"affectedRegions"`
	ReferenceTicket  string   `json:"referenceTicketNumber"`
	TimeCreated      string   `json:"timeCreated"`
	TimeUpdated      string   `json:"timeUpdated"`
}

type SAnnouncement struct {
	SAnnouncementSummary
	Description           string `json:"description"`
	AdditionalInformation string `json:"additionalInformation"`
}

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

func (self *SOracleClient) GetNotices() ([]cloudprovider.INotice, error) {
	region, err := self.Region()
	if err != nil {
		return nil, err
	}
	summaries, err := self.listAnnouncements(region)
	if err != nil {
		return nil, err
	}
	ret := make([]cloudprovider.INotice, 0, len(summaries))
	for i := range summaries {
		if !isAnnouncementPublishedToday(&summaries[i]) {
			continue
		}
		content := self.announcementContent(region, &summaries[i])
		ret = append(ret, &SNotice{
			title:   summaries[i].Summary,
			content: content,
		})
	}
	return ret, nil
}

func isAnnouncementPublishedToday(summary *SAnnouncementSummary) bool {
	ts := summary.TimeCreated
	if len(ts) == 0 {
		ts = summary.TimeUpdated
	}
	if len(ts) == 0 {
		return false
	}
	t, err := parseAnnouncementTime(ts)
	if err != nil {
		return false
	}
	return cloudprovider.IsNoticePublishedToday(t)
}

func parseAnnouncementTime(ts string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000+0000",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, ts); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.Errorf("parse announcement time %q", ts)
}

func (self *SOracleClient) listAnnouncements(region string) ([]SAnnouncementSummary, error) {
	query := url.Values{}
	query.Set("compartmentId", self.tenancyOCID)
	query.Set("lifecycleState", "ACTIVE")

	ret := []SAnnouncementSummary{}
	for {
		resp, token, err := self.request(httputils.GET, SERVICE_ANNOUNCEMENTS, region, "announcements", query, nil)
		if err != nil {
			return nil, err
		}
		items, err := resp.GetArray("items")
		if err != nil {
			return nil, errors.Wrapf(err, "GetArray items")
		}
		part := []SAnnouncementSummary{}
		err = jsonutils.Update(&part, items)
		if err != nil {
			return nil, errors.Wrapf(err, "Unmarshal announcement summaries")
		}
		ret = append(ret, part...)
		if len(token) == 0 {
			break
		}
		query.Set("page", token)
	}
	return ret, nil
}

func (self *SOracleClient) getAnnouncement(region, id string) (*SAnnouncement, error) {
	resp, err := self.get(SERVICE_ANNOUNCEMENTS, region, "announcements", id, nil)
	if err != nil {
		return nil, err
	}
	ann := &SAnnouncement{}
	err = resp.Unmarshal(ann)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal announcement")
	}
	return ann, nil
}

func (self *SOracleClient) announcementContent(region string, summary *SAnnouncementSummary) string {
	ann, err := self.getAnnouncement(region, summary.Id)
	if err == nil {
		return formatAnnouncementContent(ann)
	}
	return formatAnnouncementSummaryContent(summary)
}

func formatAnnouncementContent(ann *SAnnouncement) string {
	parts := []string{}
	if len(ann.Description) > 0 {
		parts = append(parts, ann.Description)
	}
	if len(ann.AdditionalInformation) > 0 {
		parts = append(parts, ann.AdditionalInformation)
	}
	if len(parts) > 0 {
		return strings.Join(parts, "\n\n")
	}
	return formatAnnouncementSummaryContent(&ann.SAnnouncementSummary)
}

func formatAnnouncementSummaryContent(summary *SAnnouncementSummary) string {
	lines := []string{}
	if len(summary.AnnouncementType) > 0 {
		lines = append(lines, fmt.Sprintf("Type: %s", summary.AnnouncementType))
	}
	if len(summary.Services) > 0 {
		lines = append(lines, fmt.Sprintf("Services: %s", strings.Join(summary.Services, ", ")))
	}
	if len(summary.AffectedRegions) > 0 {
		lines = append(lines, fmt.Sprintf("Affected regions: %s", strings.Join(summary.AffectedRegions, ", ")))
	}
	if len(summary.ReferenceTicket) > 0 {
		lines = append(lines, fmt.Sprintf("Reference: %s", summary.ReferenceTicket))
	}
	if len(summary.TimeCreated) > 0 {
		lines = append(lines, fmt.Sprintf("Created: %s", summary.TimeCreated))
	}
	if len(summary.TimeUpdated) > 0 {
		lines = append(lines, fmt.Sprintf("Updated: %s", summary.TimeUpdated))
	}
	if len(lines) == 0 {
		return summary.Summary
	}
	return strings.Join(lines, "\n")
}
