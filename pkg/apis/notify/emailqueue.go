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

package notify

import (
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

type SEmailMessage struct {
	To      []string `json:"to"`
	Cc      []string `json:"cc"`
	Bcc     []string `json:"bcc"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`

	Attachments []SEmailAttachment `json:"attachments"`
}

type SEmailAttachment struct {
	Filename string `json:"filename"`
	Mime     string `json:"mime"`

	Base64Content string `json:"content"`
}

type SEmailConfig struct {
	Hostname      string `json:"hostname"`
	Hostport      int    `json:"hostport"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	SenderAddress string `json:"sender_address"`
	SslGlobal     bool   `json:"ssl_global"`
}

type EmailQueueCreateInput struct {
	SEmailMessage

	// swagger: ignore
	Dest string `json:"dest"`
	// swagger: ignore
	DestCc string `json:"dest_cc"`
	// swagger: ignore
	DestBcc string `json:"dest_bcc"`
	// swagger: ignore
	Content jsonutils.JSONObject `json:"content"`

	// swagger: ignore
	ProjectId string `json:"project_id"`
	// swagger: ignore
	Project string `json:"project"`
	// swagger: ignore
	ProjectDomainId string `json:"project_domain_id"`
	// swagger: ignore
	ProjectDomain string `json:"project_domain"`
	// swagger: ignore
	UserId string `json:"user_id"`
	// swagger: ignore
	User string `json:"user"`
	// swagger: ignore
	DomainId string `json:"domain_id"`
	// swagger: ignore
	Domain string `json:"domain"`
	// swagger: ignore
	Roles string `json:"roles"`

	MoreDetails map[string]string `json:"more_datails"`

	SessionId string `json:"session_id"`
}

type EmailQueueListInput struct {
	apis.ModelBaseListInput

	Id        []int    `json:"id"`
	To        []string `json:"to"`
	Subject   string   `json:"subject"`
	SessionId []string `json:"session_id"`
}

const (
	EmailQueued  = "queued"
	EmailSending = "sending"
	EmailSuccess = "success"
	EmailFail    = "fail"
)

type EmailQueueSendInput struct {
	Sync bool `json:"sync"`
}

type EmailQueueDetails struct {
	apis.ModelBaseDetails

	SEmailQueue

	SentAt time.Time `json:"sent_at"`

	Status string `json:"status"`

	Results string `json:"results"`
}
