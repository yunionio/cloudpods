package modules

var (
	Baremetalagents ResourceManager
)

func init() {
	Baremetalagents = NewComputeManager(
		"baremetalagent",
		"baremetalagents",
		[]string{"ID", "Name", "Access_ip", "Manager_URI", "Status"},
		[]string{},
	)
	registerCompute(&Baremetalagents)
}
