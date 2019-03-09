package modules

var (
	Schedtaghosts    JointResourceManager
	Schedtagstorages JointResourceManager
)

func newSchedtagJointManager(keyword, keywordPlural string, columns, adminColumns []string, slave Manager) JointResourceManager {
	columns = append(columns, "Schedtag_ID", "Schedtag")
	return NewJointComputeManager(keyword, keywordPlural,
		columns, adminColumns, &Schedtags, slave)
}

func init() {
	Schedtaghosts = newSchedtagJointManager("schedtaghost", "schedtaghosts",
		[]string{"Host_ID", "Host"},
		[]string{},
		&Hosts)

	Schedtagstorages = newSchedtagJointManager("schedtagstorage", "schedtagstorages",
		[]string{"Storage_ID", "Storage"},
		[]string{},
		&Storages)

	registerCompute(&Schedtaghosts)

	registerCompute(&Schedtagstorages)
}
