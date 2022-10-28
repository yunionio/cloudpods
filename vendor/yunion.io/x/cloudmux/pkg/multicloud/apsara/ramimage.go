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

package apsara

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	ApsaraECSImageImportRole         = "ApsaraECSImageImportDefaultRole"
	ApsaraECSImageImportRoleDocument = `{
"Statement": [
{
"Action": "sts:AssumeRole",
"Effect": "Allow",
"Principal": {
 "Service": [
   "ecs.apsaracs.com"
 ]
}
}
],
"Version": "1"
}`

	ApsaraECSImageImportRolePolicyType     = "System"
	ApsaraECSImageImportRolePolicy         = "ApsaraECSImageImportRolePolicy"
	ApsaraECSImageImportRolePolicyDocument = `{
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

func (self *SApsaraClient) EnableImageImport() error {
	_, err := self.GetRole(ApsaraECSImageImportRole)
	if err != nil {
		if err != cloudprovider.ErrNotFound {
			return err
		}
		_, err = self.CreateRole(ApsaraECSImageImportRole,
			ApsaraECSImageImportRoleDocument,
			"Allow Import External Image from OSS")
		if err != nil {
			return err
		}
	}

	_, err = self.GetPolicy(ApsaraECSImageImportRolePolicyType, ApsaraECSImageImportRolePolicy)
	if err != nil {
		/*if err != cloudprovider.ErrNotFound {
			return err
		}
		_, err = self.createPolicy(ApsaraECSImageImportRolePolicy,
			ApsaraECSImageImportRolePolicyDocument,
			"Allow Import External Image policy")
		if err != nil {
			return err
		}*/
		return err
	}

	policies, err := self.ListPoliciesForRole(ApsaraECSImageImportRole)
	if err != nil {
		return err
	}
	for i := 0; i < len(policies); i += 1 {
		if policies[i].PolicyType == ApsaraECSImageImportRolePolicyType &&
			policies[i].PolicyName == ApsaraECSImageImportRolePolicy {
			return nil // find policy
		}
	}

	err = self.AttachPolicy2Role(ApsaraECSImageImportRolePolicyType, ApsaraECSImageImportRolePolicy, ApsaraECSImageImportRole)
	if err != nil {
		return err
	}

	return nil
}

const (
	ApsaraECSImageExportRole         = "ApsaraECSImageExportDefaultRole"
	ApsaraECSImageExportRoleDocument = `{
   "Statement": [
     {
       "Action": "sts:AssumeRole",
       "Effect": "Allow",
       "Principal": {
         "Service": [
           "ecs.apsaracs.com"
         ]
       }
     }
   ],
   "Version": "1"
}`

	ApsaraEmptyRoleDocument = `{
   "Statement": [
     {
       "Action": "sts:AssumeRole",
       "Effect": "Allow",
       "Principal": {
         "Service": [
           "ecs.apsaracs.com"
         ]
       }
     }
   ],
   "Version": "1"
}`

	ApsaraECSImageExportRolePolicyType     = "System"
	ApsaraECSImageExportRolePolicy         = "ApsaraECSImageExportRolePolicy"
	ApsaraECSImageExportRolePolicyDocument = `{
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

func (self *SApsaraClient) EnableImageExport() error {
	_, err := self.GetRole(ApsaraECSImageExportRole)
	if err != nil {
		if err != cloudprovider.ErrNotFound {
			return err
		}
		_, err = self.CreateRole(ApsaraECSImageExportRole,
			ApsaraECSImageExportRoleDocument,
			"Allow Export Import to OSS")
		if err != nil {
			return err
		}
	}

	_, err = self.GetPolicy(ApsaraECSImageExportRolePolicyType, ApsaraECSImageExportRolePolicy)
	if err != nil {
		/*if err != cloudprovider.ErrNotFound {
			return err
		}
		_, err = self.createPolicy(ApsaraECSImageImportRolePolicy,
			ApsaraECSImageImportRolePolicyDocument,
			"Allow Import External Image policy")
		if err != nil {
			return err
		}*/
		return err
	}

	policies, err := self.ListPoliciesForRole(ApsaraECSImageExportRole)
	if err != nil {
		return err
	}
	for i := 0; i < len(policies); i += 1 {
		if policies[i].PolicyType == ApsaraECSImageExportRolePolicyType &&
			policies[i].PolicyName == ApsaraECSImageExportRolePolicy {
			return nil // find policy
		}
	}

	err = self.AttachPolicy2Role(ApsaraECSImageExportRolePolicyType, ApsaraECSImageExportRolePolicy, ApsaraECSImageExportRole)
	if err != nil {
		return err
	}

	return nil
}
