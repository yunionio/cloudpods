package types

type Atomic string

const (
	String  Atomic = "string"
	Integer Atomic = "integer"
	Real    Atomic = "real"
	Boolean Atomic = "boolean"
	Uuid    Atomic = "uuid"
)

var (
	atomicGoMap = map[Atomic]string{
		String:  "string",
		Integer: "int64",
		Real:    "float64",
		Boolean: "bool",
		Uuid:    "string",
	}

	atomicFmtStrs = map[Atomic]string{
		String:  "%q",
		Integer: "%d",
		Real:    "%f",
		Boolean: "%v",
		Uuid:    "%s",
	}

	atomicUnmarshalTyp = map[Atomic]string{
		String:  "string",
		Integer: "float64",
		Real:    "float64",
		Boolean: "bool",
		Uuid:    "string",
	}

	atomicZeroVals = map[Atomic]string{
		String:  `""`,
		Integer: `0`,
		Real:    `0`,
		Boolean: `false`,
		Uuid:    `""`,
	}

	atomics = []Atomic{
		String,
		Integer,
		Real,
		Boolean,
		Uuid,
	}
)

func (a Atomic) exportName() string {
	return ExportName(string(a))
}

func (a Atomic) goType() string {
	return atomicGoMap[a]
}

func (a Atomic) fmtStr() string {
	return atomicFmtStrs[a]
}

func (a Atomic) zeroVal() string {
	return atomicZeroVals[a]
}

func (a Atomic) isValid() bool {
	_, ok := atomicGoMap[a]
	return ok
}
