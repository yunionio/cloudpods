package api

type ReservedResourcesArgs struct {
	Name   string
	Remove string
}

type ReservedResourcesResult struct {
	Resources interface{} `json:"resources"`
}
