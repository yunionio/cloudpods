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

package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/pkg/errors"
)

func (client *SAwsClient) getConfig(ctx context.Context, regionId string, assumeRole bool) (aws.Config, error) {
	httpClient, err := client.getHttpClient()
	if err != nil {
		return aws.Config{}, errors.Wrap(err, "getHttpClient")
	}
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(regionId),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(client.accessKey, client.accessSecret, "")),
		config.WithHTTPClient(httpClient),
	}
	if assumeRole && len(client.accountId) > 0 {
		// need to assumeRole
		var env string
		switch client.GetAccessEnv() {
		case api.CLOUD_ACCESS_ENV_AWS_GLOBAL:
			env = "aws"
		default:
			env = "aws-cn"
		}
		roleARN := fmt.Sprintf("arn:%s:iam::%s:role/%s", env, client.accountId, client.getAssumeRoleName())

		opts = append(opts, config.WithAssumeRoleCredentialOptions(func(options *stscreds.AssumeRoleOptions) {
			options.RoleARN = roleARN
		}))
	}
	if client.debug {
		opts = append(opts, config.WithClientLogMode(aws.LogSigning|aws.LogRequestWithBody|aws.LogResponseWithBody))
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, errors.Wrap(err, "LoadDefaultConfig")
	}
	return cfg, nil
}

func (r *SRegion) getConfig() (aws.Config, error) {
	return r.client.getConfig(context.Background(), r.RegionId, true)
}

func (r *SRegion) GetS3Client() (*s3.Client, error) {
	cfg, err := r.getConfig()
	if err != nil {
		return nil, errors.Wrap(err, "getConfig")
	}
	return s3.NewFromConfig(cfg), nil
}
