package jwt

import (
	"github.com/lestrrat-go/iter/mapiter"
	"github.com/lestrrat-go/jwx/internal/iter"
)

type ClaimPair = mapiter.Pair
type Iterator = mapiter.Iterator
type Visitor = iter.MapVisitor
type VisitorFunc iter.MapVisitorFunc
