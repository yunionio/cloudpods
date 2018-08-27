package modules

type NoticeManager struct {
	ResourceManager
}

type NoticeReadMarkManager struct {
	ResourceManager
}

var (
	Notice         NoticeManager
	NoticeReadMark NoticeReadMarkManager
)

func init() {
	Version = VersionManager{NewYunionAgentManager("notice", "notices",
		[]string{},
		[]string{})}
	register(&Notice)

	NoticeReadMark = NoticeReadMarkManager{NewYunionAgentManager("readmark", "readmarks",
		[]string{},
		[]string{})}
	register(&NoticeReadMark)
}
