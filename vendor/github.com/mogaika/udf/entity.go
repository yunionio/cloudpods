package udf

type EntityID struct {
	Flags            uint8
	Identifier       [23]byte
	IdentifierSuffix [8]byte
}

func NewEntityID(b []byte) EntityID {
	e := EntityID{Flags: b[0]}
	copy(e.Identifier[:], b[1:24])
	copy(e.IdentifierSuffix[:], b[24:32])
	return e
}
