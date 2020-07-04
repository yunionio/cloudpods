package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

var (
	ExtraUsers             modulebase.ResourceManager
	ExtraProcessDefinition modulebase.ResourceManager
	ExtraProcessInstance   modulebase.ResourceManager
	ExtraJira              modulebase.ResourceManager
)

func init() {
	ExtraUsers = NewITSMManager("extra-user", "extra-users",
		[]string{},
		[]string{},
	)

	ExtraProcessDefinition = NewITSMManager("extra-process-definition", "extra-process-definitions",
		[]string{},
		[]string{},
	)

	ExtraProcessInstance = NewITSMManager("extra-process-instance", "extra-process-instances",
		[]string{},
		[]string{},
	)

	ExtraJira = NewITSMManager("extra-jira", "extra-jiras",
		[]string{},
		[]string{},
	)

	mods := []modulebase.ResourceManager{
		ExtraJira,
		ExtraUsers,
		ExtraProcessDefinition,
		ExtraProcessInstance,
	}
	for i := range mods {

		register(&mods[i])
		registerV2(&mods[i])
	}
}
