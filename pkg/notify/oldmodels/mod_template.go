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

package oldmodels

type STemplateManager struct {
	SStandaloneResourceBaseManager
}

var TemplateManager *STemplateManager

func init() {
	TemplateManager = &STemplateManager{
		SStandaloneResourceBaseManager: NewStandaloneResourceBaseManager(
			STemplate{},
			"notify_t_template",
			"oldtemplate",
			"oldtemplates",
		),
	}
	TemplateManager.SetVirtualObject(TemplateManager)
}

const (
	TEMPLATE_TYPE_TITLE   = "title"
	TEMPLATE_TYPE_CONTENT = "content"
	TEMPLATE_TYPE_REMOTE  = "remote"
	CONTACTTYPE_ALL       = "all"
)

type STemplate struct {
	SStandaloneResourceBase

	ContactType string `width:"16" nullable:"false" create:"required" update:"user" list:"user"`
	Topic       string `width:"20" nullable:"false" create:"required" update:"user" list:"user"`

	// title | content | remote
	TemplateType string `width:"10" nullable:"false" create:"required" update:"user" list:"user"`
	Content      string `length:"text" nullable:"false" create:"required" get:"user" list:"user" update:"user"`
}
