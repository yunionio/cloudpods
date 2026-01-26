package udf

type Extent struct {
	Length   uint32
	Location uint32
}

func NewExtent(b []byte) Extent {
	return Extent{
		Length:   rl_u32(b[0:]),
		Location: rl_u32(b[4:]),
	}
}

type ExtentSmall struct {
	Length   uint16
	Location uint64
}

func NewExtentSmall(b []byte) ExtentSmall {
	return ExtentSmall{
		Length:   rl_u16(b[0:]),
		Location: rl_u48(b[2:]),
	}
}

type ExtentLong struct {
	Length   uint32
	Location uint64
}

func NewExtentLong(b []byte) ExtentLong {
	return ExtentLong{
		Length:   rl_u32(b[0:]),
		Location: rl_u48(b[4:]),
	}
}
