package jws

import (
	"context"

	"github.com/lestrrat-go/iter/mapiter"
	"github.com/lestrrat-go/jwx/internal/iter"
)

// Iterate returns a channel that successively returns all the
// header name and values.
func (h *stdHeaders) Iterate(ctx context.Context) Iterator {
	ch := make(chan *HeaderPair)
	go h.iterate(ctx, ch)
	return mapiter.New(ch)
}

func (h *stdHeaders) Walk(ctx context.Context, visitor Visitor) error {
	return iter.WalkMap(ctx, h, visitor)
}

func (h *stdHeaders) AsMap(ctx context.Context) (map[string]interface{}, error) {
	return iter.AsMap(ctx, h)
}
