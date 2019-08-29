package s3cli

type Tag struct {
	Key   string
	Value string
}

type Tagging struct {
	TagSet []Tag
}
