// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package sender

import (
	"context"
	"fmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/notify/models"
)

type SWorkwxRobotSender struct {
	config map[string]api.SNotifyConfigContent
}

func (workwxRobotSender *SWorkwxRobotSender) GetSenderType() string {
	return api.WORKWX_ROBOT
}

func (workwxRobotSender *SWorkwxRobotSender) Send(ctx context.Context, args api.SendParams) error {
	errs := []error{}
	content := fmt.Sprintf("# %s\n\n%s", args.Title, args.Message)
	mid := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"content": content,
		},
	}
	req, err := sendRequest(ctx, args.Receivers.Contact, httputils.POST, nil, nil, jsonutils.Marshal(mid))
	if err != nil {
		return errors.Wrap(err, "sendRequest")
	}
	errCode, err := req.GetString("errcode")
	if err != nil {
		return errors.Wrap(err, "req.GetString")
	}
	if errCode != "0" {
		errs = append(errs, errors.Errorf(req.PrettyString()))
	}
	return errors.NewAggregate(errs)
}

func (workwxRobotSender *SWorkwxRobotSender) ValidateConfig(ctx context.Context, config api.NotifyConfig) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (workwxRobotSender *SWorkwxRobotSender) ContactByMobile(ctx context.Context, mobile, domainId string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (workwxRobotSender *SWorkwxRobotSender) IsPersonal() bool {
	return true
}

func (workwxRobotSender *SWorkwxRobotSender) IsRobot() bool {
	return true
}

func (workwxRobotSender *SWorkwxRobotSender) IsValid() bool {
	return len(workwxRobotSender.config) > 0
}

func (workwxRobotSender *SWorkwxRobotSender) IsPullType() bool {
	return true
}

func (workwxRobotSender *SWorkwxRobotSender) IsSystemConfigContactType() bool {
	return true
}

func (workwxRobotSender *SWorkwxRobotSender) GetAccessToken(ctx context.Context, key string) error {
	return nil
}

func (workwxRobotSender *SWorkwxRobotSender) RegisterConfig(config models.SConfig) {
}

func init() {
	models.Register(&SWorkwxRobotSender{
		config: map[string]api.SNotifyConfigContent{},
	})
}
