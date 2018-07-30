package api

type ExpireArgs struct {
	DirtyHosts      []string
	DirtyBaremetals []string
}

type ExpireResult struct {
}
