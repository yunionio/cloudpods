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
	"encoding/base64"
	"io/ioutil"
	"path/filepath"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type EmailQueueListOptions struct {
	options.BaseListOptions

	// Id        []int    `json:"id"`
	To        []string `json:"to"`
	Subject   string   `json:"subject"`
	SessionId []string `json:"session_id"`
}

func (rl *EmailQueueListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(rl)
}

type EmailQueueCreateOptions struct {
	SUBJECT string   `help:"email subject"`
	BODY    string   `help:"email body"`
	TO      []string `json:"to" help:"receiver email"`
	Cc      []string `json:"cc" help:"cc receivers"`
	Bcc     []string `json:"bcc" help:"bcc receivers"`

	SessionId string `help:"session id of sending email"`

	Attach []string `help:"path to attachment"`
}

func (rc *EmailQueueCreateOptions) Params() (jsonutils.JSONObject, error) {
	input := api.EmailQueueCreateInput{}
	input.To = rc.TO
	input.Cc = rc.Cc
	input.Bcc = rc.Bcc
	input.Subject = rc.SUBJECT
	body, err := ioutil.ReadFile(rc.BODY)
	if err != nil {
		return nil, errors.Wrap(err, "Read content")
	}
	input.Body = string(body)
	for _, attach := range rc.Attach {
		contBytes, err := ioutil.ReadFile(attach)
		if err != nil {
			return nil, errors.Wrapf(err, "read %s", attach)
		}
		input.Attachments = append(input.Attachments, api.SEmailAttachment{
			Filename:      filepath.Base(attach),
			Base64Content: base64.StdEncoding.EncodeToString(contBytes),
		})
	}
	log.Debugf("%s", jsonutils.Marshal(input))
	return jsonutils.Marshal(input), nil
}

type EmailQueueOptions struct {
	ID string `help:"Id of email queue" json:"-"`
}

func (r *EmailQueueOptions) GetId() string {
	return r.ID
}

func (r *EmailQueueOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(r), nil
}

type EmailQueueSendOptions struct {
	EmailQueueOptions

	Sync bool `json:"sync" help:"send email synchronously"`
}

func (r *EmailQueueSendOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(r), nil
}
