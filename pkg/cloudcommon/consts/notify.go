package consts

var (
	NotifyTemplateDir = "/opt/yunion/share/notify_templates"
)

func SetNotifyTemplateDir(dir string) {
	NotifyTemplateDir = dir
}
