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

import "yunion.io/x/onecloud/pkg/apis"

type TemplateCreateInput struct {
	apis.StandaloneAnonResourceCreateInput

	// description: Contact type, specifically, setting it to all means all contact type
	// require: true
	// example: email
	ContactType string `json:"contact_type"`
	// description: Template type
	// enum: title,content,remote
	// example: title
	TemplateType string `json:"template_type"`

	// description: Template topic
	// required: true
	// example: IMAGE_ACTIVE
	Topic string `json:"topic"`

	// description: Template content
	// required: true
	// example: 镜像 {{.name}} 上传完成
	Content string `json:"content"`
	// description: Example for using this template
	// required: true
	// example: {"name": "centos7.6"}
	Example string `json:"example"`
	// description: Language
	// enum: cn,en
	Lang string `json:"lang"`
}

type TemplateManagerSaveInput struct {
	ContactType string
	Templates   []TemplateCreateInput
	Force       bool
}

type TemplateListInput struct {
	apis.StandaloneAnonResourceListInput

	// description: Contact type, specifically, setting it to all means all contact type
	// require: true
	// example: email
	ContactType string `json:"contact_type"`

	// description: Template type
	// enum: title,content,remote
	// example: title
	TemplateType string `json:"template_type"`

	// description: template topic
	// required: true
	// example: IMAGE_ACTIVE
	Topic string `json:"topic"`

	// description: Language
	// enum: cn,en
	Lang string `json:"lang"`
}

type TemplateUpdateInput struct {
	apis.StandaloneAnonResourceBaseUpdateInput

	// description: template content
	// required: true
	// example: 镜像 {{.name}} 上传完成
	Content string `json:"content"`
	// description: all example for using this template
	// required: true
	// example: {"name": "centos7.6"}
	Example string `json:"example"`
}

type TemplateDetails struct {
	apis.StandaloneResourceDetails

	STemplate
}
