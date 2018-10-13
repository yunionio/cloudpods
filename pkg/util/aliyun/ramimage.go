package aliyun

import (
	"yunion.io/x/onecloud/pkg/cloudprovider"
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
		if err != cloudprovider.ErrNotFound {
			return err
		}
		_, err = self.createRole(AliyunECSImageImportRole,
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

	policies, err := self.ListPolicies("", AliyunECSImageImportRole)
	if err != nil {
		return err
	}
	for i := 0; i < len(policies); i += 1 {
		if policies[i].PolicyType == AliyunECSImageImportRolePolicyType &&
			policies[i].PolicyName == AliyunECSImageImportRolePolicy {
			return nil // find policy
		}
	}

	err = self.attachPolicy2Role(AliyunECSImageImportRolePolicyType, AliyunECSImageImportRolePolicy, AliyunECSImageImportRole)
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
		_, err = self.createRole(AliyunECSImageExportRole,
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

	policies, err := self.ListPolicies("", AliyunECSImageExportRole)
	if err != nil {
		return err
	}
	for i := 0; i < len(policies); i += 1 {
		if policies[i].PolicyType == AliyunECSImageExportRolePolicyType &&
			policies[i].PolicyName == AliyunECSImageExportRolePolicy {
			return nil // find policy
		}
	}

	err = self.attachPolicy2Role(AliyunECSImageExportRolePolicyType, AliyunECSImageExportRolePolicy, AliyunECSImageExportRole)
	if err != nil {
		return err
	}

	return nil
}
