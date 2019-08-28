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

package handlers

import (
	"context"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/s3cli"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/s3gateway/models"
)

type sBucketInfo struct {
	Name      string
	CreatedAt time.Time
}

func listService(ctx context.Context, userCred mcclient.TokenCredential, query s3cli.ListBucketsInput) (*s3cli.ListAllMyBucketsResult, error) {
	result, err := models.BucketManager.List(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "models.BucketManager.List")
	}
	resp := s3cli.ListAllMyBucketsResult{}
	resp.Owner.ID = userCred.GetProjectId()
	resp.Owner.DisplayName = userCred.GetProjectName()
	resp.Buckets.Bucket = make([]s3cli.BucketInfo, 0)
	for i := range result {
		info := result[i]
		resp.Buckets.Bucket = append(resp.Buckets.Bucket, s3cli.BucketInfo{
			Name:         info.Name,
			CreationDate: info.CreatedAt,
		})
	}
	return &resp, nil
}
