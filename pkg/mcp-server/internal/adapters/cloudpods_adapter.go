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

package adapters

import (
	"context"
	"github.com/sirupsen/logrus"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcp-server/internal/config"
)

type CloudpodsAdapter struct {
	config  *config.Config
	logger  *logrus.Logger
	client  *mcclient.Client
	session *mcclient.ClientSession
}

type CloudRegion struct {
	RegionId string `json:"region_id"`
}

func NewCloudpodsAdapter(cfg *config.Config, logger *logrus.Logger) *CloudpodsAdapter {

	client := mcclient.NewClient(
		cfg.External.Cloudpods.BaseURL,
		cfg.External.Cloudpods.Timeout,
		false,
		true,
		"",
		"",
	)

	return &CloudpodsAdapter{
		config: cfg,
		logger: logger,
		client: client,
	}
}

func (a *CloudpodsAdapter) authenticate() error {
	if a.session != nil {
		return nil
	}

	token, err := a.client.AuthenticateByAccessKey("", "", "")
	if err != nil {
		return err
	}

	a.session = a.client.NewSession(
		context.Background(),
		"",
		"",
		"publicURL",
		token,
	)

	return nil
}

func (a *CloudpodsAdapter) getSession() (*mcclient.ClientSession, error) {
	if err := a.authenticate(); err != nil {
		return nil, err
	}
	return a.session, nil
}
