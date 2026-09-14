package regclient

import (
	"context"

	"github.com/regclient/regclient/scheme"
	"github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/repo"
)

type repoLister interface {
	RepoList(ctx context.Context, hostname string, opts ...scheme.RepoOpts) (*repo.RepoList, error)
}

// RepoList returns a list of repositories on a registry
// Note the underlying "_catalog" API is not supported on many cloud registries
func (rc *RegClient) RepoList(ctx context.Context, hostname string, opts ...scheme.RepoOpts) (*repo.RepoList, error) {
	schemeAPI, err := rc.schemeGet("reg")
	if err != nil {
		return nil, err
	}
	rl, ok := schemeAPI.(repoLister)
	if !ok {
		return nil, types.ErrNotImplemented
	}
	return rl.RepoList(ctx, hostname, opts...)

}
