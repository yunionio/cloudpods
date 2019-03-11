package modules

var (
	Metadatas ResourceManager
)

func init() {
	Metadatas = NewComputeManager("metadata", "metadata",
		[]string{"id", "key", "value"},
		[]string{})
	registerCompute(&Metadatas)
}
