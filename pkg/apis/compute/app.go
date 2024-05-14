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

package compute

import (
	"time"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	APP_TYPE_WEB = "web"

	APP_STATUS_READY   = "ready"
	APP_STATUS_UNKNOWN = "unknown"
)

type AppListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	apis.EnabledResourceBaseListInput

	ManagedResourceListInput
	RegionalFilterListInput

	TechStack string `json:"tech_stack"`
}

type AppDetails struct {
	apis.VirtualResourceDetails
	ManagedResourceInfo
	CloudregionResourceInfo
	Network string
	VpcId   string
	Vpc     string
}

type AppEnvironmentListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput

	AppId        string
	InstanceType string
}

type AppEnvironmentDetails struct {
	apis.VirtualResourceDetails
}

type AppHybirdConnection struct {
	Id        string
	Name      string
	Hostname  string
	Namespace string
	Port      int
}

type AppHybirdConnectionOutput struct {
	Data []AppHybirdConnection
}

type AppBackup struct {
	Id   string
	Name string
	Type string
}

type AppBackupOutput struct {
	Data         []AppBackup
	BackupConfig struct {
		Enabled               bool
		FrequencyInterval     int
		FrequencyUnit         string
		RetentionPeriodInDays int
	}
}

type AppCertificate struct {
	Id          string
	Name        string
	SubjectName string
	Issuer      string
	IssueDate   time.Time
	Thumbprint  string
	ExpireTime  time.Time
}

type AppCertificateOutput struct {
	Data []AppCertificate
}

type AppDomain struct {
	Id       string
	Name     string
	Status   string
	SslState string
}

type AppDomainOutput struct {
	Data []AppDomain
}
