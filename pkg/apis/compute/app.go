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
	Network string `json:"network"`
	VpcId   string `json:"vpc_id"`
	Vpc     string `json:"vpc"`
}

type AppEnvironmentListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput

	AppId        string `json:"app_id"`
	InstanceType string `json:"instance_type"`
}

type AppEnvironmentDetails struct {
	apis.VirtualResourceDetails
}

type AppHybirdConnection struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Hostname  string `json:"hostname"`
	Namespace string `json:"namespace"`
	Port      int    `json:"port"`
}

type AppHybirdConnectionOutput struct {
	Data []AppHybirdConnection `json:"data"`
}

type AppBackup struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type AppBackupOutput struct {
	Data         []AppBackup `json:"data"`
	BackupConfig struct {
		Enabled               bool   `json:"enabled"`
		FrequencyInterval     int    `json:"frequency_interval"`
		FrequencyUnit         string `json:"frequency_unit"`
		RetentionPeriodInDays int    `json:"retention_period_in_days"`
	} `json:"backup_config"`
}

type AppCertificate struct {
	Id          string    `json:"id"`
	Name        string    `json:"name"`
	SubjectName string    `json:"subject_name"`
	Issuer      string    `json:"issuer"`
	IssueDate   time.Time `json:"issue_date"`
	Thumbprint  string    `json:"thumbprint"`
	ExpireTime  time.Time `json:"expire_time"`
}

type AppCertificateOutput struct {
	Data []AppCertificate `json:"data"`
}

type AppDomain struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	SslState string `json:"ssl_state"`
}

type AppDomainOutput struct {
	Data []AppDomain `json:"data"`
}
