package modules

var (
	UnderutilizedInstances ResourceManager
)

func init() {
	UnderutilizedInstances = NewCloudmonManager("underutilizedinstance", "underutilizedinstances",
		[]string{"id", "vm_id", "vm_name", "time", "advices", "vm_cpu", "vm_disk", "vm_memory", "vm_provider"},
		[]string{})

	register(&UnderutilizedInstances)
}
