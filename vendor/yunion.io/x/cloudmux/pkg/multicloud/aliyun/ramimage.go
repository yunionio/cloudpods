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
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	AliyunECSImageImportRole         = "AliyunECSImageImportDefaultRole"
	AliyunECSImageImportRoleDocument = `{
"Statement": [
{
"Action": "sts:AssumeRole",
"Effect": "Allow",
"Principal": {
 "Service": [
   "ecs.aliyuncs.com"
 ]
}
}
],
"Version": "1"
}`

	AliyunECSImageImportRolePolicyType     = "System"
	AliyunECSImageImportRolePolicy         = "AliyunECSImageImportRolePolicy"
	AliyunECSImageImportRolePolicyDocument = `{
"Version": "1",
"Statement": [
{
"Action": [
 "oss:GetObject",
 "oss:GetBucketLocation"
],
"Resource": "*",
"Effect": "Allow"
}
]
}`
)

func (self *SAliyunClient) EnableImageImport() error {
	_, err := self.GetRole(AliyunECSImageImportRole)
	if err != nil {
		if errors.Cause(err) != cloudprovider.ErrNotFound {
			return err
		}
		_, err = self.CreateRole(AliyunECSImageImportRole,
			AliyunECSImageImportRoleDocument,
			"Allow Import External Image from OSS")
		if err != nil {
			return err
		}
	}

	_, err = self.GetPolicy(AliyunECSImageImportRolePolicyType, AliyunECSImageImportRolePolicy)
	if err != nil {
		/*if err != cloudprovider.ErrNotFound {
			return err
		}
		_, err = self.createPolicy(AliyunECSImageImportRolePolicy,
			AliyunECSImageImportRolePolicyDocument,
			"Allow Import External Image policy")
		if err != nil {
			return err
		}*/
		return err
	}

	policies, err := self.ListPoliciesForRole(AliyunECSImageImportRole)
	if err != nil {
		return err
	}
	for i := 0; i < len(policies); i += 1 {
		if policies[i].PolicyType == AliyunECSImageImportRolePolicyType &&
			policies[i].PolicyName == AliyunECSImageImportRolePolicy {
			return nil // find policy
		}
	}

	err = self.AttachPolicy2Role(AliyunECSImageImportRolePolicyType, AliyunECSImageImportRolePolicy, AliyunECSImageImportRole)
	if err != nil {
		return err
	}

	return nil
}

const (
	AliyunECSImageExportRole         = "AliyunECSImageExportDefaultRole"
	AliyunECSImageExportRoleDocument = `{
   "Statement": [
     {
       "Action": "sts:AssumeRole",
       "Effect": "Allow",
       "Principal": {
         "Service": [
           "ecs.aliyuncs.com"
         ]
       }
     }
   ],
   "Version": "1"
}`

	AliyunEmptyRoleDocument = `{
   "Statement": [
     {
       "Action": "sts:AssumeRole",
       "Effect": "Allow",
       "Principal": {
         "Service": [
           "ecs.aliyuncs.com"
         ]
       }
     }
   ],
   "Version": "1"
}`

	AliyunECSImageExportRolePolicyType     = "System"
	AliyunECSImageExportRolePolicy         = "AliyunECSImageExportRolePolicy"
	AliyunECSImageExportRolePolicyDocument = `{
   "Version": "1",
   "Statement": [
     {
       "Action": [
         "oss:GetObject",
         "oss:PutObject",
         "oss:DeleteObject",
         "oss:GetBucketLocation",
         "oss:AbortMultipartUpload",
         "oss:ListMultipartUploads",
         "oss:ListParts"
       ],
       "Resource": "*",
       "Effect": "Allow"
     }
   ]
 }`
)

func (self *SAliyunClient) EnableImageExport() error {
	_, err := self.GetRole(AliyunECSImageExportRole)
	if err != nil {
		if err != cloudprovider.ErrNotFound {
			return err
		}
		_, err = self.CreateRole(AliyunECSImageExportRole,
			AliyunECSImageExportRoleDocument,
			"Allow Export Import to OSS")
		if err != nil {
			return err
		}
	}

	_, err = self.GetPolicy(AliyunECSImageExportRolePolicyType, AliyunECSImageExportRolePolicy)
	if err != nil {
		/*if err != cloudprovider.ErrNotFound {
			return err
		}
		_, err = self.createPolicy(AliyunECSImageImportRolePolicy,
			AliyunECSImageImportRolePolicyDocument,
			"Allow Import External Image policy")
		if err != nil {
			return err
		}*/
		return err
	}

	policies, err := self.ListPoliciesForRole(AliyunECSImageExportRole)
	if err != nil {
		return err
	}
	for i := 0; i < len(policies); i += 1 {
		if policies[i].PolicyType == AliyunECSImageExportRolePolicyType &&
			policies[i].PolicyName == AliyunECSImageExportRolePolicy {
			return nil // find policy
		}
	}

	err = self.AttachPolicy2Role(AliyunECSImageExportRolePolicyType, AliyunECSImageExportRolePolicy, AliyunECSImageExportRole)
	if err != nil {
		return err
	}

	return nil
}
