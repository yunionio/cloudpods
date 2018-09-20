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
	Notice = NoticeManager{NewYunionAgentManager("notice", "notices",
		[]string{"id", "created_at", "updated_at", "author_id", "author", "title", "content"},
		[]string{})}
	register(&Notice)

	NoticeReadMark = NoticeReadMarkManager{NewYunionAgentManager("readmark", "readmarks",
		[]string{},
		[]string{"notice_id", "user_id"})}
	register(&NoticeReadMark)
}
