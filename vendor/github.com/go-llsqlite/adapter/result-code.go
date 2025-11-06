package sqlite

// ResultCode is an abstraction over what is exposed in crawshaw or zombiezen. For now the type
// wraps the definitions there, but we add some extra common methods on top here.

func (me ResultCode) ToPrimary() ResultCode {
	return me & 0xff
}
