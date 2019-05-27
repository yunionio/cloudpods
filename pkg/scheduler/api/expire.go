package api

type ExpireArgs struct {
	DirtyHosts      []string
	DirtyBaremetals []string
	SessionId       string
}

type ExpireResult struct {
}
