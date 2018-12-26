package pending_delete

type SPendingDeleteOptions struct {
	EnablePendingDelete            bool `default:"true" help:"Turn on/off pending-delete resource, default is on" alias:"delayed_delete"`
	PendingDeleteCheckSeconds      int  `default:"3600" help:"How long to wait to scan pending-delete resource, default is 1 hour"`
	PendingDeleteExpireSeconds     int  `default:"259200" help:"How long a pending-delete resource cleaned automatically, default 3 days" alias:"scrub_time"`
	PendingDeleteMaxCleanBatchSize int  `default:"50" help:"How many pending-delete items can be clean in a batch"`
}
