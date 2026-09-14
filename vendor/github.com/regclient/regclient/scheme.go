package regclient

import (
	"context"
	"fmt"

	"github.com/regclient/regclient/scheme"
	"github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/ref"
)

func (rc *RegClient) schemeGet(scheme string) (scheme.API, error) {
	s, ok := rc.schemes[scheme]
	if !ok {
		return nil, fmt.Errorf("%w: unknown scheme \"%s\"", types.ErrNotImplemented, scheme)
	}
	return s, nil
}

func (rc *RegClient) schemeInfo(r ref.Ref) (scheme.Info, error) {
	schemeAPI, err := rc.schemeGet(r.Scheme)
	if err != nil {
		return scheme.Info{}, err
	}
	return schemeAPI.Info(), nil
}

// Close is used to free resources associated with a reference
// With ocidir, this may trigger a garbage collection process
func (rc *RegClient) Close(ctx context.Context, r ref.Ref) error {
	schemeAPI, err := rc.schemeGet(r.Scheme)
	if err != nil {
		return err
	}
	// verify Closer api is defined, noop if missing
	sc, ok := schemeAPI.(scheme.Closer)
	if !ok {
		return nil
	}
	return sc.Close(ctx, r)
}
