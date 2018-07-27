package modules

var (
	Notifications ResourceManager
)

func init() {
	Notifications = NewNotifyManager("notification", "notifications",
		[]string{"id", "uid", "contact_type", "topic", "priority", "msg", "received_at", "send_by", "status", "create_at", "update_at", "delete_at", "create_by", "update_by", "delete_by", "is_deleted", "remark"},
		[]string{})

	register(&Notifications)
}
