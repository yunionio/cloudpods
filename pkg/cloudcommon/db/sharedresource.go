package db

// sharing resoure between project
type SSharedResource struct {
	SResourceBase

	Id int64 `primary:"true" auto_increment:"true" list:"user"`

	ResourceType    string `width:"32" charset:"ascii" nullable:"false" list:"user"`
	ResourceId      string `width:"128" charset:"ascii" nullable:"false" index:"true" list:"user"`
	OwnerProjectId  string `width:"128" charset:"ascii" nullable:"false" index:"true" list:"user"`
	TargetProjectId string `width:"128" charset:"ascii" nullable:"false" index:"true" list:"user"`
}

type SSharedResourceManager struct {
	SResourceBaseManager
}

var SharedResourceManager *SSharedResourceManager

func init() {
	SharedResourceManager = &SSharedResourceManager{
		SResourceBaseManager: NewResourceBaseManager(
			SSharedResource{},
			"shared_resources_tbl",
			"shared_resource",
			"shared_resources",
		),
	}
}
