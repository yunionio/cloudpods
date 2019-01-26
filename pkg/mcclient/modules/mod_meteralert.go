package modules

var (
	MeterAlert ResourceManager
)

func init() {
	MeterAlert = NewMeterAlertManager("meteralert", "meteralerts",
		[]string{"id", "type", "provider", "account", "account_id", "comparator", "threshold", "recipients", "level", "channel", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark"},
		[]string{})

	register(&MeterAlert)
}
