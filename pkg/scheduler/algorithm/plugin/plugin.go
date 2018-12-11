package plugin

import (
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type BasePlugin struct{}

// Customize priority
func (p BasePlugin) OnPriorityEnd(u *core.Unit, c core.Candidater) {}

func (p BasePlugin) OnSelectEnd(u *core.Unit, c core.Candidater, count int64) {}
