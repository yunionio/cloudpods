package notify

type TNotifyPriority string

type TNotifyChannel string

const (
	NotifyPriorityImportant = TNotifyPriority("important")
	NotifyPriorityCritical  = TNotifyPriority("fatal")
	NotifyPriorityNormal    = TNotifyPriority("normal")

	NotifyByEmail      = TNotifyChannel("email")
	NotifyByMobile     = TNotifyChannel("mobile")
	NotifyByDingTalk   = TNotifyChannel("dingtalk")
	NotifyByWebConsole = TNotifyChannel("webconsole")
)
