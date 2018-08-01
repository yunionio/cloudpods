package modules

var (
	Verifications ResourceManager
)

func init() {
	Verifications = NewNotifyManager("verification", "verifications",
		[]string{"id", "cid", "sent_at", "expire_at", "status", "create_at", "update_at", "delete_at", "create_by", "update_by", "delete_by", "is_deleted", "remark"},
		[]string{})

	register(&Verifications)
}
