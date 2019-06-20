package models

import (
	"time"

	"yunion.io/x/onecloud/pkg/util/ansible"
)

type AnsiblePlaybook struct {
	VirtualResource

	Playbook  *ansible.Playbook
	Output    string
	StartTime time.Time
	EndTime   time.Time
}
