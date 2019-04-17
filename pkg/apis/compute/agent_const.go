package compute

const (
	BAREMETAL_AGENT_ENABLED  = "enabled"
	BAREMETAL_AGENT_DISABLED = "disabled"
	BAREMETAL_AGENT_OFFLINE  = "offline"
)

type TAgentType string

const (
	AgentTypeBaremetal = TAgentType("baremetal")
	AgentTypeEsxi      = TAgentType("esxiagent")
	AgentTypeDefault   = AgentTypeBaremetal
)
