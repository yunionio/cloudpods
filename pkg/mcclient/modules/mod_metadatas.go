package modules

var (
	Metadatas ResourceManager
)

func init() {
	Metadatas = NewComputeManager("metadata", "metadatas",
		[]string{"id", "key", "value"},
		[]string{})
	registerCompute(&Metadatas)
}
